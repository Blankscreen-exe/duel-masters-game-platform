// Package matchmaking runs quick match: a player picks a format and a deck
// and waits, and is paired with the player waiting in the same format whose
// rating is closest to theirs, within a range that widens the longer they
// wait. It shares the lobby's register of what players are doing, so a
// player waits here or hosts or plays, one at a time.
package matchmaking

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/games"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/lobby"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/ratings"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/settings"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/users"
)

// gameName names every game quick match pairs.
const gameName = "Quick match"

// defaultSweepInterval is how often the waiting players are paired, as their
// rating ranges widen, and checked for having left.
const defaultSweepInterval = time.Second

// Ticket is a player waiting for a quick match.
type Ticket struct {
	UserID     string
	Username   string
	FormatID   string
	FormatName string
	DeckID     string
	// Rating is the player's rating as they started waiting, which cannot
	// change while they wait.
	Rating int
	Since  time.Time
}

// Entered is how entering quick match went: paired with a waiting player at
// once, with the game started, or waiting.
type Entered struct {
	// Game is the game started, nil while waiting.
	Game *games.Game
	// Ticket is the player's place in the queue, while waiting.
	Ticket Ticket
}

// Listener passes what happens in quick match on to the players. It is
// called on the service's goroutines and must not block.
type Listener interface {
	// QueueChanged shows everyone how many players wait in each format.
	QueueChanged(waiting map[string]int)
	// GameStarted tells a paired game's two players to go to it.
	GameStarted(games.Game)
}

// Dependencies are what quick match needs.
type Dependencies struct {
	Lobby     *lobby.Service
	Games     *games.Service
	Ratings   *ratings.Service
	Directory lobby.Directory
	Settings  *settings.Service
	Listener  Listener
	Logger    *slog.Logger
	// Now is the clock; time.Now outside tests.
	Now func() time.Time
	// SweepEvery is how often the waiting players are paired and checked
	// for having left; 0 is every second.
	SweepEvery time.Duration
}

// Service runs quick match. It is safe for concurrent use.
type Service struct {
	lobby      *lobby.Service
	games      *games.Service
	ratings    *ratings.Service
	directory  lobby.Directory
	settings   *settings.Service
	listener   Listener
	logger     *slog.Logger
	now        func() time.Time
	sweepEvery time.Duration

	mu sync.Mutex
	// queue holds the waiting players of each format, the longest waiting
	// first.
	queue map[string][]Ticket
}

// NewService returns quick match.
func NewService(deps Dependencies) *Service {
	s := &Service{
		lobby:      deps.Lobby,
		games:      deps.Games,
		ratings:    deps.Ratings,
		directory:  deps.Directory,
		settings:   deps.Settings,
		listener:   deps.Listener,
		logger:     deps.Logger,
		now:        deps.Now,
		sweepEvery: deps.SweepEvery,
		queue:      map[string][]Ticket{},
	}
	if s.sweepEvery <= 0 {
		s.sweepEvery = defaultSweepInterval
	}

	return s
}

// Enter puts a player in quick match with one of their decks, legal in the
// format. If someone suitable is already waiting in the format, the two are
// paired and the game starts, the player who waited longer hosting;
// otherwise the player waits, to be paired as soon as someone suits them. It
// returns the errors lobby.Service.PrepareDeck returns, a *lobby.BusyError
// for a player already doing something, and lobby.ErrPaused while new games
// are paused.
func (s *Service) Enter(ctx context.Context, player users.User, formatID, deckID string) (Entered, error) {
	if s.lobby.Paused() {
		return Entered{}, lobby.ErrPaused
	}

	format, deck, err := s.lobby.PrepareDeck(ctx, player, formatID, deckID)
	if err != nil {
		return Entered{}, err
	}
	standing, err := s.ratings.Standing(ctx, player.ID)
	if err != nil {
		return Entered{}, err
	}

	ticket := Ticket{
		UserID: player.ID, Username: player.Username, FormatID: format.ID, FormatName: format.Name, DeckID: deck.ID,
		Rating: standing.Rank.Rating, Since: s.now(),
	}
	activities := s.lobby.Activities()
	if err := activities.Claim(player.ID, lobby.Activity{Kind: lobby.Queued, FormatName: format.Name}); err != nil {
		return Entered{}, err
	}

	for {
		opponent, paired := s.pairOrWait(ticket)
		if !paired {
			s.queueChanged()
			return Entered{Ticket: ticket}, nil
		}

		game, err := s.start(ctx, opponent, ticket)
		var illegal *games.IllegalDeckError
		switch {
		case err == nil:
			return Entered{Game: &game}, nil
		case errors.As(err, &illegal) && illegal.Username == opponent.Username:
			// The waiting player's deck stopped keeping to the format while
			// they waited: they leave the queue, and the search goes on.
			activities.Release(opponent.UserID, lobby.Queued, "")
			s.queueChanged()
			continue
		default:
			s.putBack(opponent)
			activities.Release(player.ID, lobby.Queued, "")
			return Entered{}, err
		}
	}
}

// pairOrWait takes the waiting player online who suits the ticket best out
// of its format's queue, or else adds the ticket to it, in one step, so two
// players arriving at once are always paired rather than both left waiting.
func (s *Service) pairOrWait(ticket Ticket) (Ticket, bool) {
	rules := s.pairing()
	s.mu.Lock()
	defer s.mu.Unlock()

	waiting := s.queue[ticket.FormatID]
	if i := rules.best(ticket, waiting, s.directory.IsOnline); i >= 0 {
		s.queue[ticket.FormatID] = slices.Delete(slices.Clone(waiting), i, i+1)
		return waiting[i], true
	}

	s.queue[ticket.FormatID] = append(waiting, ticket)
	return Ticket{}, false
}

// takePair takes the next two waiting players to pair out of the queue: the
// longest waiting player online whom anyone suits, and the one who suits
// them best. The first hosts.
func (s *Service) takePair() (host, guest Ticket, ok bool) {
	rules := s.pairing()
	s.mu.Lock()
	defer s.mu.Unlock()

	for formatID, waiting := range s.queue {
		for i, ticket := range waiting {
			if !s.directory.IsOnline(ticket.UserID) {
				continue
			}
			// Those waiting longer suit nobody here, so suit not this one.
			later := waiting[i+1:]
			j := rules.best(ticket, later, s.directory.IsOnline)
			if j < 0 {
				continue
			}

			host, guest = ticket, later[j]
			kept := slices.Delete(slices.Clone(waiting), i+1+j, i+2+j)
			s.queue[formatID] = slices.Delete(kept, i, i+1)
			return host, guest, true
		}
	}
	return Ticket{}, Ticket{}, false
}

// pairWaiting pairs waiting players who have come to suit each other as
// their ranges widened, and starts their games. A player whose deck can no
// longer be played leaves the queue, and the other waits on.
func (s *Service) pairWaiting(ctx context.Context) {
	for {
		host, guest, ok := s.takePair()
		if !ok {
			return
		}

		_, err := s.start(ctx, host, guest)
		if err == nil {
			continue
		}

		var illegal *games.IllegalDeckError
		broken := errors.As(err, &illegal)
		for _, ticket := range []Ticket{host, guest} {
			if broken && illegal.Username == ticket.Username {
				s.lobby.Activities().Release(ticket.UserID, lobby.Queued, "")
			} else {
				s.putBack(ticket)
			}
		}
		s.queueChanged()
		if !broken {
			// Something else went wrong, such as the server stopping: they
			// wait on, to be tried again at the next sweep.
			s.logger.ErrorContext(ctx, "pairing a quick match failed", "error", err, "host", host.Username, "guest", guest.Username)
			return
		}
	}
}

// pairing is who suits whom now.
func (s *Service) pairing() pairing {
	return pairing{start: int(s.settings.Number(RatingRange)), now: s.now()}
}

// putBack returns a waiting player to their queue, where they were: the
// queue is in the order the players started waiting.
func (s *Service) putBack(ticket Ticket) {
	s.mu.Lock()
	defer s.mu.Unlock()

	waiting := s.queue[ticket.FormatID]
	at, _ := slices.BinarySearchFunc(waiting, ticket.Since, func(waiting Ticket, since time.Time) int {
		return waiting.Since.Compare(since)
	})
	s.queue[ticket.FormatID] = slices.Insert(slices.Clone(waiting), at, ticket)
}

// start starts the game of two paired players, the one who waited longer
// hosting, and tells them.
func (s *Service) start(ctx context.Context, host, guest Ticket) (games.Game, error) {
	hostUser := users.User{ID: host.UserID, Username: host.Username}
	guestUser := users.User{ID: guest.UserID, Username: guest.Username}

	format, hostDeck, err := s.lobby.PrepareChosenDeck(ctx, hostUser, host.FormatID, host.DeckID)
	if err != nil {
		return games.Game{}, err
	}
	_, guestDeck, err := s.lobby.PrepareChosenDeck(ctx, guestUser, guest.FormatID, guest.DeckID)
	if err != nil {
		return games.Game{}, err
	}

	game, err := s.games.Start(ctx, games.StartInput{
		Name:        gameName,
		Format:      format,
		Host:        games.Entrant{User: hostUser, Deck: hostDeck},
		Guest:       games.Entrant{User: guestUser, Deck: guestDeck},
		Visible:     true,
		Spectators:  true,
		Matchmaking: true,
	})
	if err != nil {
		return games.Game{}, err
	}

	activities := s.lobby.Activities()
	activities.Change(host.UserID, lobby.Queued, lobby.Activity{Kind: lobby.Playing, GameID: game.ID})
	activities.Change(guest.UserID, lobby.Queued, lobby.Activity{Kind: lobby.Playing, GameID: game.ID})
	s.queueChanged()
	s.listener.GameStarted(game)
	s.logger.InfoContext(ctx, "paired a quick match", "game", game.ID, "format", format.Name, "host", host.Username, "guest", guest.Username)

	return game, nil
}

// Leave takes a player out of quick match, and reports whether they were
// waiting.
func (s *Service) Leave(userID string) bool {
	s.mu.Lock()
	left := false
	for formatID, waiting := range s.queue {
		kept := slices.DeleteFunc(slices.Clone(waiting), func(ticket Ticket) bool { return ticket.UserID == userID })
		if len(kept) != len(waiting) {
			s.queue[formatID] = kept
			left = true
		}
	}
	s.mu.Unlock()

	if left {
		s.lobby.Activities().Release(userID, lobby.Queued, "")
		s.queueChanged()
	}
	return left
}

// Ticket returns a player's place in the queue, if they are waiting.
func (s *Service) Ticket(userID string) (Ticket, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, waiting := range s.queue {
		for _, ticket := range waiting {
			if ticket.UserID == userID {
				return ticket, true
			}
		}
	}
	return Ticket{}, false
}

// Waiting counts the players waiting in each format with anyone waiting.
func (s *Service) Waiting() map[string]int {
	s.mu.Lock()
	defer s.mu.Unlock()

	waiting := map[string]int{}
	for formatID, tickets := range s.queue {
		if len(tickets) > 0 {
			waiting[formatID] = len(tickets)
		}
	}
	return waiting
}

// PresenceChanged takes a player who has left the site out of the queue.
func (s *Service) PresenceChanged(userID string) {
	if !s.directory.IsOnline(userID) {
		s.Leave(userID)
	}
}

// Run pairs waiting players as their ranges widen, and takes players who
// have left out of the queue, a backstop for presence changes, which can
// arrive in any order, until ctx ends.
func (s *Service) Run(ctx context.Context) {
	ticker := time.NewTicker(s.sweepEvery)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for _, userID := range s.waitingPlayers() {
				s.PresenceChanged(userID)
			}
			s.pairWaiting(ctx)
		}
	}
}

func (s *Service) waitingPlayers() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	var players []string
	for _, waiting := range s.queue {
		for _, ticket := range waiting {
			players = append(players, ticket.UserID)
		}
	}
	return players
}

func (s *Service) queueChanged() {
	s.listener.QueueChanged(s.Waiting())
}
