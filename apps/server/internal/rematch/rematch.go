// Package rematch offers the two players of a game that has just ended
// another game against each other. Either player offers, and each may switch
// to another of their decks legal in the game's format; once both have agreed
// to the decks shown, the new game starts with the same format and options,
// its coin toss called by the loser of the last. Declining, leaving the site
// or letting the offer run out ends it. While deciding, the players do
// nothing else, as the lobby's register of what players are doing keeps.
package rematch

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/decks"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/formats"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/games"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/lobby"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/settings"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/users"
)

// OfferTime is the admin setting for how long the players have to agree.
var OfferTime = settings.Number{
	Key:         "rematch.offer_seconds",
	Group:       "Games",
	Description: "How long, in seconds, the players of a game that has ended have to agree on a rematch before the offer closes.",
	Default:     120,
	Min:         15,
	Max:         900,
}

// Settings are the settings this feature declares.
func Settings() []settings.Definition {
	return []settings.Definition{OfferTime}
}

// defaultSweepInterval is how often offers are checked for running out and
// for players who have left.
const defaultSweepInterval = time.Second

var (
	// ErrNotFound is returned for a rematch that is not being decided, or
	// not by this player.
	ErrNotFound = errors.New("there is no rematch to decide")
	// ErrOpponentDeckIllegal is returned to a player agreeing to a rematch
	// whose opponent's deck no longer keeps to the format; the opponent must
	// choose another deck, and both agree again.
	ErrOpponentDeckIllegal = errors.New("the other player's deck no longer keeps to the format")
)

// Reason is why a rematch offer ended without a game.
type Reason string

const (
	// Declined: a player said no, or went back to the lobby.
	Declined Reason = "declined"
	// Left: a player left the site.
	Left Reason = "left"
	// Expired: the players did not agree in time.
	Expired Reason = "expired"
)

// Deck is a deck chosen for a rematch, as both players see it: by name and
// civilizations only, besides its id, which only its owner is shown.
type Deck struct {
	ID            string
	Name          string
	Civilizations []string
}

// Seat is a player of a rematch and the deck they would play.
type Seat struct {
	UserID   string
	Username string
	Deck     Deck
	// Ready is set once the player has agreed to play the decks shown.
	Ready bool
}

// Rematch is a rematch being decided.
type Rematch struct {
	// GameID is the game that ended.
	GameID      string
	Name        string
	FormatID    string
	FormatName  string
	Visible     bool
	Spectators  bool
	Matchmaking bool
	// Host and Guest sit as they did in the game that ended.
	Host  Seat
	Guest Seat
	// CallerID calls the new game's coin toss: the loser of the last game, or
	// empty, after a game without a winner, for either at random.
	CallerID  string
	ExpiresAt time.Time
}

// Has reports whether a user is one of the rematch's players.
func (r Rematch) Has(userID string) bool {
	return r.Host.UserID == userID || r.Guest.UserID == userID
}

// Seats returns the player's own seat and their opponent's. The player must
// be one of the rematch's.
func (r *Rematch) Seats(userID string) (own, opponent *Seat) {
	if r.Host.UserID == userID {
		return &r.Host, &r.Guest
	}
	return &r.Guest, &r.Host
}

// Listener passes what happens to rematches on to the players. It is called
// on the service's goroutines, sometimes while a game is ending, and must not
// block.
type Listener interface {
	// RematchChanged shows both players the rematch as it now is.
	RematchChanged(Rematch)
	// RematchEnded tells both players the rematch will not be played, and
	// why: byUserID is the player who declined or left, empty when it ran
	// out.
	RematchEnded(rematch Rematch, reason Reason, byUserID string)
	// GameStarted tells the two players to go to their new game.
	GameStarted(games.Game)
}

// Dependencies are what rematches need.
type Dependencies struct {
	Lobby     *lobby.Service
	Games     *games.Service
	Directory lobby.Directory
	Settings  *settings.Service
	Listener  Listener
	Logger    *slog.Logger
	// Now is the clock; time.Now outside tests.
	Now func() time.Time
	// SweepEvery is how often offers are checked for running out and for
	// players who have left; 0 is every second.
	SweepEvery time.Duration
}

// Service runs rematches. It is safe for concurrent use.
type Service struct {
	lobby      *lobby.Service
	games      *games.Service
	directory  lobby.Directory
	settings   *settings.Service
	listener   Listener
	logger     *slog.Logger
	now        func() time.Time
	sweepEvery time.Duration

	mu sync.Mutex
	// open holds the rematches being decided, by the id of the game that
	// ended. One being started is taken out first, so it starts once.
	open map[string]*Rematch
}

var _ games.Listener = (*Service)(nil)

// NewService returns rematches, listening to the games service for games
// ending.
func NewService(deps Dependencies) *Service {
	s := &Service{
		lobby:      deps.Lobby,
		games:      deps.Games,
		directory:  deps.Directory,
		settings:   deps.Settings,
		listener:   deps.Listener,
		logger:     deps.Logger,
		now:        deps.Now,
		sweepEvery: deps.SweepEvery,
		open:       map[string]*Rematch{},
	}
	if s.sweepEvery <= 0 {
		s.sweepEvery = defaultSweepInterval
	}
	deps.Games.Listen(s)

	return s
}

// GameEnded offers a rematch to the players of a game that has just ended,
// when both are still on the site, each with the deck they played. A game
// the server stopped has none.
func (s *Service) GameEnded(ended games.Ended) {
	game := ended.Game
	if ended.Ending == games.EndingAborted || !s.directory.IsOnline(game.Host.ID) || !s.directory.IsOnline(game.Guest.ID) {
		return
	}

	// Moved on from playing at once, before the game closes and frees them,
	// so neither starts anything else in between.
	activities := s.lobby.Activities()
	deciding := lobby.Activity{Kind: lobby.Rematching, GameID: game.ID}
	if !activities.Change(game.Host.ID, lobby.Playing, deciding) {
		return
	}
	if !activities.Change(game.Guest.ID, lobby.Playing, deciding) {
		activities.Release(game.Host.ID, lobby.Rematching, game.ID)
		return
	}

	rematch := &Rematch{
		GameID:      game.ID,
		Name:        game.Name,
		FormatID:    game.FormatID,
		FormatName:  game.FormatName,
		Visible:     game.Visible,
		Spectators:  game.Spectators,
		Matchmaking: game.Matchmaking,
		Host:        seatOf(game.Host),
		Guest:       seatOf(game.Guest),
		CallerID:    ended.LoserID(),
		ExpiresAt:   s.now().Add(time.Duration(s.settings.Number(OfferTime)) * time.Second),
	}

	s.mu.Lock()
	s.open[game.ID] = rematch
	shown := *rematch
	s.mu.Unlock()

	s.listener.RematchChanged(shown)
}

func seatOf(player games.Player) Seat {
	return Seat{
		UserID:   player.ID,
		Username: player.Username,
		Deck:     Deck{ID: player.DeckID, Name: player.DeckName, Civilizations: player.Civilizations},
	}
}

// GameFinished is part of games.Listener; the rematch was offered when the
// game ended.
func (s *Service) GameFinished(games.Record) {}

// GameClosed is part of games.Listener; the rematch was offered when the game
// ended.
func (s *Service) GameClosed(games.Game) {}

// Get returns the rematch of a game being decided by one of its players. It
// returns ErrNotFound for any other.
func (s *Service) Get(userID, gameID string) (Rematch, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rematch, ok := s.open[gameID]
	if !ok || !rematch.Has(userID) {
		return Rematch{}, ErrNotFound
	}
	return *rematch, nil
}

// ChooseDeck switches the player to another of their decks, legal in the
// format. The opponent agreed to the decks they were shown, so a different
// deck takes back their agreement. It returns ErrNotFound as Get does, and
// the errors lobby.Service.PrepareDeck returns for the deck.
func (s *Service) ChooseDeck(ctx context.Context, player users.User, gameID, deckID string) (Rematch, error) {
	current, err := s.Get(player.ID, gameID)
	if err != nil {
		return Rematch{}, err
	}

	chosen, err := s.prepare(ctx, player, current.FormatID, deckID)
	if err != nil {
		return Rematch{}, err
	}

	s.mu.Lock()
	rematch, ok := s.open[gameID]
	if !ok || !rematch.Has(player.ID) {
		s.mu.Unlock()
		return Rematch{}, ErrNotFound
	}
	own, opponent := rematch.Seats(player.ID)
	if own.Deck.ID != chosen.ID {
		opponent.Ready = false
	}
	own.Deck = chosen
	shown := *rematch
	s.mu.Unlock()

	s.listener.RematchChanged(shown)
	return shown, nil
}

// Ready records that the player agrees to play the decks shown: an offer, or
// an acceptance of the opponent's. Once both agree, the new game starts and
// both are told to go to it; Ready then returns it, and otherwise the
// rematch, waiting for the opponent. It returns ErrNotFound as Get does,
// lobby.ErrPaused while new games are paused, the errors
// lobby.Service.PrepareDeck returns for the player's own deck, and
// ErrOpponentDeckIllegal when the opponent's no longer keeps to the format.
func (s *Service) Ready(ctx context.Context, player users.User, gameID string) (*games.Game, Rematch, error) {
	if s.lobby.Paused() {
		return nil, Rematch{}, lobby.ErrPaused
	}

	shown, both, err := s.agree(ctx, player, gameID)
	if err != nil || !both {
		return nil, shown, err
	}

	game, err := s.start(ctx, shown)
	if err != nil {
		// Back to deciding: nobody has agreed to anything that can be played.
		shown.Host.Ready, shown.Guest.Ready = false, false
		s.mu.Lock()
		s.open[gameID] = &shown
		s.mu.Unlock()
		s.listener.RematchChanged(shown)

		var illegal *games.IllegalDeckError
		if errors.As(err, &illegal) && illegal.Username != player.Username {
			return nil, Rematch{}, ErrOpponentDeckIllegal
		}
		return nil, Rematch{}, err
	}

	return &game, Rematch{}, nil
}

// agree marks the player ready with their deck, checked first, and reports
// whether both now are; the rematch is then taken out of the open ones, to
// be started once. Otherwise both players are shown it.
func (s *Service) agree(ctx context.Context, player users.User, gameID string) (Rematch, bool, error) {
	for {
		current, err := s.Get(player.ID, gameID)
		if err != nil {
			return Rematch{}, false, err
		}
		checked, _ := current.Seats(player.ID)
		// The player's deck may have changed, or gone, since it was chosen.
		if _, err := s.prepare(ctx, player, current.FormatID, checked.Deck.ID); err != nil {
			return Rematch{}, false, err
		}

		s.mu.Lock()
		rematch, ok := s.open[gameID]
		if !ok || !rematch.Has(player.ID) || !s.now().Before(rematch.ExpiresAt) {
			s.mu.Unlock()
			return Rematch{}, false, ErrNotFound
		}
		own, opponent := rematch.Seats(player.ID)
		if own.Deck.ID != checked.Deck.ID {
			// Switched again while the deck was checked: check the new one.
			s.mu.Unlock()
			continue
		}
		own.Ready = true
		both := opponent.Ready
		if both {
			delete(s.open, gameID)
		}
		shown := *rematch
		s.mu.Unlock()

		if !both {
			s.listener.RematchChanged(shown)
		}
		return shown, both, nil
	}
}

// start starts the rematch's game and tells both players.
func (s *Service) start(ctx context.Context, rematch Rematch) (games.Game, error) {
	host, hostDeck, format, err := s.entrant(ctx, rematch, rematch.Host)
	if err != nil {
		return games.Game{}, err
	}
	guest, guestDeck, _, err := s.entrant(ctx, rematch, rematch.Guest)
	if err != nil {
		return games.Game{}, err
	}

	game, err := s.games.Start(ctx, games.StartInput{
		Name:        rematch.Name,
		Format:      format,
		Host:        games.Entrant{User: host, Deck: hostDeck},
		Guest:       games.Entrant{User: guest, Deck: guestDeck},
		Visible:     rematch.Visible,
		Spectators:  rematch.Spectators,
		Matchmaking: rematch.Matchmaking,
		TossCaller:  rematch.CallerID,
	})
	if err != nil {
		return games.Game{}, err
	}

	activities := s.lobby.Activities()
	playing := lobby.Activity{Kind: lobby.Playing, GameID: game.ID}
	activities.Change(host.ID, lobby.Rematching, playing)
	activities.Change(guest.ID, lobby.Rematching, playing)
	s.listener.GameStarted(game)
	s.logger.InfoContext(ctx, "started a rematch", "game", game.ID, "rematch of", rematch.GameID, "host", host.Username, "guest", guest.Username)

	return game, nil
}

// entrant is a seat's player and deck, checked against the format once more.
// A deck deleted since it was chosen counts as one that cannot be played.
func (s *Service) entrant(ctx context.Context, rematch Rematch, seat Seat) (users.User, decks.Deck, formats.Format, error) {
	player := users.User{ID: seat.UserID, Username: seat.Username}
	format, deck, err := s.lobby.PrepareDeck(ctx, player, rematch.FormatID, seat.Deck.ID)
	var illegal *games.IllegalDeckError
	if err != nil && !errors.As(err, &illegal) {
		err = &games.IllegalDeckError{Username: seat.Username}
	}
	return player, deck, format, err
}

// prepare checks one of the player's decks for the format, and describes it
// as the rematch shows it.
func (s *Service) prepare(ctx context.Context, player users.User, formatID, deckID string) (Deck, error) {
	_, deck, err := s.lobby.PrepareDeck(ctx, player, formatID, deckID)
	if err != nil {
		return Deck{}, err
	}
	civilizations, err := s.games.Civilizations(ctx, deck)
	if err != nil {
		return Deck{}, err
	}
	return Deck{ID: deck.ID, Name: deck.Name, Civilizations: civilizations}, nil
}

// Decline ends a rematch for both players, as one of them says no or goes
// back to the lobby. It returns ErrNotFound as Get does.
func (s *Service) Decline(userID, gameID string) error {
	s.mu.Lock()
	rematch, ok := s.open[gameID]
	if !ok || !rematch.Has(userID) {
		s.mu.Unlock()
		return ErrNotFound
	}
	delete(s.open, gameID)
	s.mu.Unlock()

	s.ended(*rematch, Declined, userID)
	return nil
}

// PresenceChanged ends the rematches of a player who has left the site.
func (s *Service) PresenceChanged(userID string) {
	if s.directory.IsOnline(userID) {
		return
	}

	s.mu.Lock()
	var left []Rematch
	for id, rematch := range s.open {
		if rematch.Has(userID) {
			left = append(left, *rematch)
			delete(s.open, id)
		}
	}
	s.mu.Unlock()

	for _, rematch := range left {
		s.ended(rematch, Left, userID)
	}
}

// Run ends rematches that ran out, and those of players who have left, until
// ctx ends: a backstop for presence changes, which can arrive in any order.
func (s *Service) Run(ctx context.Context) {
	ticker := time.NewTicker(s.sweepEvery)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.sweep()
		}
	}
}

// sweep ends the rematches that ran out or lost a player.
func (s *Service) sweep() {
	now := s.now()

	type ending struct {
		rematch Rematch
		reason  Reason
		by      string
	}
	var endings []ending

	s.mu.Lock()
	for id, rematch := range s.open {
		switch {
		case !now.Before(rematch.ExpiresAt):
			endings = append(endings, ending{rematch: *rematch, reason: Expired})
		case !s.directory.IsOnline(rematch.Host.UserID):
			endings = append(endings, ending{rematch: *rematch, reason: Left, by: rematch.Host.UserID})
		case !s.directory.IsOnline(rematch.Guest.UserID):
			endings = append(endings, ending{rematch: *rematch, reason: Left, by: rematch.Guest.UserID})
		default:
			continue
		}
		delete(s.open, id)
	}
	s.mu.Unlock()

	for _, e := range endings {
		s.ended(e.rematch, e.reason, e.by)
	}
}

// ended frees the players of a rematch taken out of the open ones, and tells
// them it will not be played.
func (s *Service) ended(rematch Rematch, reason Reason, byUserID string) {
	activities := s.lobby.Activities()
	for _, userID := range []string{rematch.Host.UserID, rematch.Guest.UserID} {
		activities.Release(userID, lobby.Rematching, rematch.GameID)
	}
	s.listener.RematchEnded(rematch, reason, byUserID)
}
