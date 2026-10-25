// Package lobby keeps the open games players wait in for an opponent, public
// or reached by invite link, and what every player is doing, so each does
// one thing at a time. It starts games through package games and shows the
// lobby: open games, games running and who is online.
package lobby

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/decks"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/formats"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/games"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/settings"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/users"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/validation"
)

const (
	// NameMaxLength is the longest game name, in characters.
	NameMaxLength = 50
	// OpenGameLifetime is how long an open game waits for an opponent.
	OpenGameLifetime = 10 * time.Minute
	// defaultSweepInterval is how often open games are checked for expiry
	// and for hosts who have left.
	defaultSweepInterval = 5 * time.Second
)

// NewGamesOpen is the admin setting that pauses new games.
var NewGamesOpen = settings.Bool{
	Key:         "lobby.new_games_open",
	Group:       "Lobby",
	Description: "Whether players can start new games, in the lobby or by quick match. Turn off to pause new games, such as before a restart; running games carry on.",
	Default:     true,
}

// Settings are the settings this feature declares.
func Settings() []settings.Definition {
	return []settings.Definition{NewGamesOpen}
}

var (
	// ErrNotFound is returned for an open game that does not exist, or no
	// longer does.
	ErrNotFound = errors.New("open game not found")
	// ErrOwnGame is returned to a host trying to join their own game.
	ErrOwnGame = errors.New("you cannot join your own game")
	// ErrPaused is returned while an admin has paused new games.
	ErrPaused = errors.New("new games are paused")
	// ErrNotHost is returned to someone cancelling a game they do not host.
	ErrNotHost = errors.New("only the host can cancel the game")
	// ErrHostDeckIllegal is returned to a player joining a game whose host's
	// deck has stopped keeping to the format; the game is closed.
	ErrHostDeckIllegal = errors.New("the host's deck no longer keeps to the format")
)

// Person is a player as the lobby shows them.
type Person struct {
	ID       string
	Username string
}

// Directory knows who is online: who has the site open.
type Directory interface {
	Online() []Person
	IsOnline(userID string) bool
}

// OpenGame is a game waiting for an opponent.
type OpenGame struct {
	ID         string
	Name       string
	Host       Person
	FormatID   string
	FormatName string
	// Visible lists the game in the lobby; otherwise only its invite link
	// leads to it.
	Visible bool
	// Spectators lets other players watch the game once it runs.
	Spectators bool
	CreatedAt  time.Time
}

// OnlinePlayer is a player who has the site open, and what they are doing.
type OnlinePlayer struct {
	Person
	// Activity is nil for a player doing nothing in particular.
	Activity *Activity
}

// Snapshot is the lobby as everyone sees it.
type Snapshot struct {
	// OpenGames are the listed open games, the oldest first.
	OpenGames []OpenGame
	// Running are the listed running games, the oldest first.
	Running []games.Game
	// Online are the players online, by name.
	Online []OnlinePlayer
	// GamesRunning counts every running game, listed or not.
	GamesRunning int
	// Paused is set while an admin has paused new games.
	Paused bool
}

// Listener passes what happens in the lobby on to the players. It is called
// on the lobby's goroutines and must not block.
type Listener interface {
	// LobbyChanged shows everyone the lobby as it now is.
	LobbyChanged(Snapshot)
	// GameStarted tells a game's two players to go to it.
	GameStarted(games.Game)
	// OpenGameExpired tells a host nobody joined their game in time.
	OpenGameExpired(OpenGame)
}

// CreateInput is an open game as its host asks for it.
type CreateInput struct {
	// Name is the game's name; empty picks one.
	Name       string
	DeckID     string
	FormatID   string
	Visible    bool
	Spectators bool
}

// openGame is an open game with the host's deck.
type openGame struct {
	OpenGame
	deckID string
}

// Service runs the lobby. It is safe for concurrent use.
type Service struct {
	decks      *decks.Service
	formats    *formats.Service
	games      *games.Service
	directory  Directory
	settings   *settings.Service
	listener   Listener
	logger     *slog.Logger
	now        func() time.Time
	sweepEvery time.Duration
	activities *Activities

	mu   sync.Mutex
	open map[string]*openGame
}

var _ games.Listener = (*Service)(nil)

// Dependencies are what the lobby needs.
type Dependencies struct {
	Decks     *decks.Service
	Formats   *formats.Service
	Games     *games.Service
	Directory Directory
	Settings  *settings.Service
	Listener  Listener
	Logger    *slog.Logger
	// Now is the clock; time.Now outside tests.
	Now func() time.Time
	// SweepEvery is how often open games are checked for expiry and for
	// hosts who have left; 0 is every 5 seconds.
	SweepEvery time.Duration
}

// NewService returns the lobby, listening to the games service for games
// ending.
func NewService(deps Dependencies) *Service {
	s := &Service{
		decks:      deps.Decks,
		formats:    deps.Formats,
		games:      deps.Games,
		directory:  deps.Directory,
		settings:   deps.Settings,
		listener:   deps.Listener,
		logger:     deps.Logger,
		now:        deps.Now,
		sweepEvery: deps.SweepEvery,
		open:       map[string]*openGame{},
	}
	if s.sweepEvery <= 0 {
		s.sweepEvery = defaultSweepInterval
	}
	s.activities = NewActivities(s.broadcast)
	deps.Games.Listen(s)

	return s
}

// Activities is the register of what every player is doing, which quick
// match shares.
func (s *Service) Activities() *Activities {
	return s.activities
}

// Paused reports whether an admin has paused new games.
func (s *Service) Paused() bool {
	return !s.settings.Bool(NewGamesOpen)
}

// Create opens a game for a host with one of their decks, legal in the
// format. It returns a *validation.Error for input breaking the rules, a
// *games.IllegalDeckError for a deck breaking the format, a *BusyError for a
// host already doing something, and ErrPaused while new games are paused.
func (s *Service) Create(ctx context.Context, host users.User, input CreateInput) (OpenGame, error) {
	if s.Paused() {
		return OpenGame{}, ErrPaused
	}

	name, err := normalizeName(input.Name)
	if err != nil {
		return OpenGame{}, err
	}
	if name == "" {
		name = randomName()
	}

	format, deck, err := s.PrepareDeck(ctx, host, input.FormatID, input.DeckID)
	if err != nil {
		return OpenGame{}, err
	}

	game := &openGame{
		OpenGame: OpenGame{
			ID:         uuid.NewString(),
			Name:       name,
			Host:       Person{ID: host.ID, Username: host.Username},
			FormatID:   format.ID,
			FormatName: format.Name,
			Visible:    input.Visible,
			Spectators: input.Spectators,
			CreatedAt:  s.now(),
		},
		deckID: deck.ID,
	}

	// Claimed first, so the game is never listed for a host who is busy.
	if err := s.activities.Claim(host.ID, Activity{Kind: Hosting, GameID: game.ID}); err != nil {
		return OpenGame{}, err
	}

	s.mu.Lock()
	s.open[game.ID] = game
	s.mu.Unlock()
	s.broadcast()

	s.logger.InfoContext(ctx, "opened a game", "game", game.ID, "host", host.Username, "format", format.Name, "listed", game.Visible)
	return game.OpenGame, nil
}

// Join starts an open game with a player and one of their decks: the game
// begins with its coin toss, and both players are told to go to it. It
// returns ErrNotFound for a game no longer open (its host left, or someone
// joined first), ErrOwnGame for the host, ErrHostDeckIllegal when the
// host's deck no longer keeps to the format, and the errors Create returns
// for the joining player's deck.
func (s *Service) Join(ctx context.Context, guest users.User, openGameID, deckID string) (games.Game, error) {
	if s.Paused() {
		return games.Game{}, ErrPaused
	}

	// Taken out of the lobby at once, so two players cannot both join.
	s.mu.Lock()
	open, ok := s.open[openGameID]
	switch {
	case !ok:
		s.mu.Unlock()
		return games.Game{}, ErrNotFound
	case open.Host.ID == guest.ID:
		s.mu.Unlock()
		return games.Game{}, ErrOwnGame
	}
	delete(s.open, openGameID)
	s.mu.Unlock()

	// Put back for the next player when this one cannot join.
	putBack := func() {
		s.mu.Lock()
		s.open[openGameID] = open
		s.mu.Unlock()
		s.broadcast()
	}

	if !s.directory.IsOnline(open.Host.ID) {
		s.activities.Release(open.Host.ID, Hosting, openGameID)
		return games.Game{}, ErrNotFound
	}

	if err := s.activities.Claim(guest.ID, Activity{Kind: Playing, GameID: openGameID}); err != nil {
		putBack()
		return games.Game{}, err
	}
	released := false
	releaseGuest := func() {
		if !released {
			released = true
			s.activities.Release(guest.ID, Playing, openGameID)
		}
	}

	format, guestDeck, err := s.formatAndDeck(ctx, guest.ID, open.FormatID, deckID)
	if err != nil {
		releaseGuest()
		putBack()
		return games.Game{}, err
	}
	hostDeck, err := s.decks.Get(ctx, open.Host.ID, open.deckID)
	if err != nil {
		// The host deleted the deck: the game cannot be played.
		releaseGuest()
		s.activities.Release(open.Host.ID, Hosting, openGameID)
		return games.Game{}, ErrHostDeckIllegal
	}

	game, err := s.games.Start(ctx, games.StartInput{
		Name:       open.Name,
		Format:     format,
		Host:       games.Entrant{User: users.User{ID: open.Host.ID, Username: open.Host.Username}, Deck: hostDeck},
		Guest:      games.Entrant{User: guest, Deck: guestDeck},
		Visible:    open.Visible,
		Spectators: open.Spectators,
	})
	var illegal *games.IllegalDeckError
	switch {
	case errors.As(err, &illegal) && illegal.Username == guest.Username:
		releaseGuest()
		putBack()
		return games.Game{}, err
	case errors.As(err, &illegal):
		releaseGuest()
		s.activities.Release(open.Host.ID, Hosting, openGameID)
		return games.Game{}, ErrHostDeckIllegal
	case err != nil:
		releaseGuest()
		putBack()
		return games.Game{}, err
	}

	s.activities.Change(open.Host.ID, Hosting, Activity{Kind: Playing, GameID: game.ID})
	s.activities.Change(guest.ID, Playing, Activity{Kind: Playing, GameID: game.ID})
	s.listener.GameStarted(game)

	return game, nil
}

// Cancel closes an open game; only its host may. It returns ErrNotFound for
// a game no longer open and ErrNotHost for anyone else.
func (s *Service) Cancel(ctx context.Context, host users.User, openGameID string) error {
	s.mu.Lock()
	open, ok := s.open[openGameID]
	switch {
	case !ok:
		s.mu.Unlock()
		return ErrNotFound
	case open.Host.ID != host.ID:
		s.mu.Unlock()
		return ErrNotHost
	}
	delete(s.open, openGameID)
	s.mu.Unlock()

	s.activities.Release(host.ID, Hosting, openGameID)
	return nil
}

// OpenGame returns an open game, listed or not, as its invite link shows it.
func (s *Service) OpenGame(openGameID string) (OpenGame, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	open, ok := s.open[openGameID]
	if !ok {
		return OpenGame{}, ErrNotFound
	}
	return open.OpenGame, nil
}

// Snapshot is the lobby as everyone sees it now.
func (s *Service) Snapshot() Snapshot {
	s.mu.Lock()
	listed := make([]OpenGame, 0, len(s.open))
	for _, open := range s.open {
		if open.Visible {
			listed = append(listed, open.OpenGame)
		}
	}
	s.mu.Unlock()
	slices.SortFunc(listed, func(a, b OpenGame) int { return a.CreatedAt.Compare(b.CreatedAt) })

	all := s.games.Games()
	running := slices.DeleteFunc(slices.Clone(all), func(game games.Game) bool { return !game.Visible })

	people := s.directory.Online()
	online := make([]OnlinePlayer, 0, len(people))
	for _, person := range people {
		player := OnlinePlayer{Person: person}
		if activity, ok := s.activities.Of(person.ID); ok {
			player.Activity = &activity
		}
		online = append(online, player)
	}

	return Snapshot{OpenGames: listed, Running: running, Online: online, GamesRunning: len(all), Paused: s.Paused()}
}

// Run closes open games nobody joined in time, and those whose host has
// left, until ctx ends.
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

// sweep closes the open games that have waited too long or lost their host.
func (s *Service) sweep() {
	cutoff := s.now().Add(-OpenGameLifetime)

	s.mu.Lock()
	var expired, abandoned []*openGame
	for id, open := range s.open {
		switch {
		case !open.CreatedAt.After(cutoff):
			expired = append(expired, open)
			delete(s.open, id)
		case !s.directory.IsOnline(open.Host.ID):
			abandoned = append(abandoned, open)
			delete(s.open, id)
		}
	}
	s.mu.Unlock()

	for _, open := range slices.Concat(expired, abandoned) {
		s.activities.Release(open.Host.ID, Hosting, open.ID)
	}
	for _, open := range expired {
		s.listener.OpenGameExpired(open.OpenGame)
	}
}

// PresenceChanged closes the open games of a player who has left, and shows
// everyone who is online now.
func (s *Service) PresenceChanged(userID string) {
	if s.directory.IsOnline(userID) {
		s.broadcast()
		return
	}

	s.mu.Lock()
	var left []*openGame
	for id, open := range s.open {
		if open.Host.ID == userID {
			left = append(left, open)
			delete(s.open, id)
		}
	}
	s.mu.Unlock()

	for _, open := range left {
		s.activities.Release(userID, Hosting, open.ID)
	}
	s.broadcast()
}

// GameFinished is part of games.Listener; the lobby waits for the game to
// close.
func (s *Service) GameFinished(games.Record) {}

// GameClosed frees the game's players to start something else.
func (s *Service) GameClosed(game games.Game) {
	s.activities.Release(game.Host.ID, Playing, game.ID)
	s.activities.Release(game.Guest.ID, Playing, game.ID)
	s.broadcast()
}

// broadcast shows everyone the lobby as it now is.
func (s *Service) broadcast() {
	s.listener.LobbyChanged(s.Snapshot())
}

// PrepareDeck finds the format and one of the player's decks, and checks the
// deck keeps to the format, for a player about to play. It returns a
// *validation.Error naming the field for a format or deck that does not exist
// (or is someone else's), and a *games.IllegalDeckError for a deck breaking the
// format.
func (s *Service) PrepareDeck(ctx context.Context, player users.User, formatID, deckID string) (formats.Format, decks.Deck, error) {
	format, deck, err := s.formatAndDeck(ctx, player.ID, formatID, deckID)
	if err != nil {
		return formats.Format{}, decks.Deck{}, err
	}
	if err := s.checkDeck(ctx, player, deck, format); err != nil {
		return formats.Format{}, decks.Deck{}, err
	}

	return format, deck, nil
}

// formatAndDeck finds the format and one of the player's decks, as
// validation errors naming the field for ones that do not exist.
func (s *Service) formatAndDeck(ctx context.Context, userID, formatID, deckID string) (formats.Format, decks.Deck, error) {
	format, err := s.formats.ByID(ctx, formatID)
	if errors.Is(err, formats.ErrNotFound) {
		return formats.Format{}, decks.Deck{}, validation.New("formatId", "must be one of the formats")
	}
	if err != nil {
		return formats.Format{}, decks.Deck{}, err
	}

	deck, err := s.decks.Get(ctx, userID, deckID)
	if errors.Is(err, decks.ErrNotFound) {
		return formats.Format{}, decks.Deck{}, validation.New("deckId", "must be one of your decks")
	}
	if err != nil {
		return formats.Format{}, decks.Deck{}, err
	}

	return format, deck, nil
}

// checkDeck returns a *games.IllegalDeckError for a deck breaking the format.
func (s *Service) checkDeck(ctx context.Context, player users.User, deck decks.Deck, format formats.Format) error {
	problems, err := s.games.CheckDeck(ctx, deck, format)
	if err != nil {
		return err
	}
	if len(problems) > 0 {
		return &games.IllegalDeckError{Username: player.Username, Problems: problems}
	}
	return nil
}

// normalizeName trims a game's name and checks it; empty stays empty.
func normalizeName(name string) (string, error) {
	name = strings.TrimSpace(name)
	switch {
	case !utf8.ValidString(name):
		return "", validation.New("name", "must be valid text")
	case strings.ContainsFunc(name, unicode.IsControl):
		return "", validation.New("name", "must be one line of text")
	case utf8.RuneCountInString(name) > NameMaxLength:
		return "", validation.New("name", fmt.Sprintf("must be at most %d characters", NameMaxLength))
	}
	return name, nil
}
