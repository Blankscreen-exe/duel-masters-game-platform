package games

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"sync"
	"time"

	"gorm.io/gorm"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/cards"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/chat"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/computer"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/decks"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/server"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/formats"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/settings"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/users"
)

// DisconnectGrace is the admin setting for how long a disconnected player has
// to come back before losing.
var DisconnectGrace = settings.Number{
	Key:         "games.disconnect_grace_seconds",
	Group:       "Games",
	Description: "How long, in seconds, a player who loses their connection has to come back before losing the game.",
	Default:     120,
	Min:         15,
	Max:         900,
}

// MinimumCountedDuration is the admin setting for how long a game must last
// to count.
var MinimumCountedDuration = settings.Number{
	Key:         "games.minimum_counted_seconds",
	Group:       "Games",
	Description: "Games shorter than this, in seconds, stay in the match history but do not count for ratings, so leaving straight away cannot farm rating.",
	Default:     60,
	Min:         0,
	Max:         1800,
}

// MaxComputerGames is the admin setting for how many games against the
// computer may run at once.
var MaxComputerGames = settings.Number{
	Key:         "computer.max_games",
	Group:       "Computer opponents",
	Description: "The most games against the computer running at once, so a busy evening cannot overload the server. At 0, nobody can start one.",
	Default:     100,
	Min:         0,
	Max:         1000,
}

// Settings are the settings this feature declares. The chat in games also
// follows the chat's own settings (chat.Settings), which must be declared too.
func Settings() []settings.Definition {
	return []settings.Definition{DisconnectGrace, MinimumCountedDuration, MessagesPerSecond, MessageBurst, MaxComputerGames}
}

// Why a game ended without a winner, as its players are told.
const (
	abortMessage   = "The server is restarting, so this game was stopped. It does not count."
	stoppedMessage = "An admin stopped this game. It does not count."
)

var (
	// ErrNotFound is returned for a game that is not running.
	ErrNotFound = errors.New("game not found")
	// ErrNotWatchable is returned to someone who is not playing a game that
	// does not allow spectators, or is not listed.
	ErrNotWatchable = errors.New("this game cannot be watched")
	// ErrClosed is returned once the server has started stopping.
	ErrClosed = errors.New("the server is stopping")
	// ErrComputerGamesFull is returned for a game against the computer while
	// as many as the settings allow are running.
	ErrComputerGamesFull = errors.New("too many games against the computer are running")
)

// IllegalDeckError is returned when a player's deck breaks the format, or,
// without problems, when a deck chosen earlier has gone since.
type IllegalDeckError struct {
	Username string
	Problems []decks.Problem
}

func (e *IllegalDeckError) Error() string {
	if len(e.Problems) == 0 {
		return fmt.Sprintf("%s's deck can no longer be played in this format", e.Username)
	}
	return fmt.Sprintf("%s's deck is not legal in this format: %s", e.Username, e.Problems[0].Message())
}

// Entrant is a player starting a game with one of their decks.
type Entrant struct {
	User users.User
	Deck decks.Deck
	// Computer, when set, is how the computer plays this seat, which the
	// service then seats itself. A game against the computer never counts.
	Computer *computer.Options
}

// StartInput is a game to start.
type StartInput struct {
	Name   string
	Format formats.Format
	Host   Entrant
	Guest  Entrant
	// Visible lists the game in the lobby; Spectators lets other players
	// watch it, if it is listed.
	Visible    bool
	Spectators bool
	// Matchmaking marks a game paired by quick match.
	Matchmaking bool
	// TossCaller is the id of the player who calls the coin toss, or empty
	// for either at random.
	TossCaller string
}

// Player is a player of a running game.
type Player struct {
	ID       string
	Username string
	DeckID   string
	DeckName string
	// Civilizations are those of the deck's cards, in the game's order.
	Civilizations []string
	// Computer marks a seat the computer plays.
	Computer bool
}

// Game is a running game.
type Game struct {
	ID          string
	Name        string
	FormatID    string
	FormatName  string
	Host        Player
	Guest       Player
	Visible     bool
	Spectators  bool
	Matchmaking bool
	StartedAt   time.Time
}

// Has reports whether a user plays in the game.
func (g Game) Has(userID string) bool {
	return g.Host.ID == userID || g.Guest.ID == userID
}

// AgainstComputer reports whether the computer plays one of the seats.
func (g Game) AgainstComputer() bool {
	return g.Host.Computer || g.Guest.Computer
}

// Watchable reports whether players other than its own may watch the game.
func (g Game) Watchable() bool {
	return g.Visible && g.Spectators
}

// Role is how someone takes part in a game.
type Role string

const (
	RolePlayer    Role = "player"
	RoleSpectator Role = "spectator"
)

// Ended is a game that started and has just ended.
type Ended struct {
	Game Game
	// WinnerID is empty for a game without a winner.
	WinnerID string
	Ending   Ending
}

// LoserID is the id of the player who lost, empty for a game without a winner.
func (e Ended) LoserID() string {
	switch e.WinnerID {
	case e.Game.Host.ID:
		return e.Game.Guest.ID
	case e.Game.Guest.ID:
		return e.Game.Host.ID
	default:
		return ""
	}
}

// Listener hears about games ending. Its methods are called on the service's
// own goroutines and must not block.
type Listener interface {
	// GameEnded reports a game that started and has just ended, at once:
	// before it is saved, and before it closes.
	GameEnded(Ended)
	// GameFinished reports a game that started and ended, once it is saved.
	GameFinished(Record)
	// GameClosed reports that a game is no longer running, finished or not.
	GameClosed(Game)
}

// liveGame is a running game with what is needed to record it.
type liveGame struct {
	game      Game
	match     *match.Match
	hostDeck  []decks.Entry
	guestDeck []decks.Entry
}

// finished is a result on its way to the database.
type finished struct {
	result match.Result
	live   liveGame
}

// resultQueue is how many finished games may wait to be saved.
const resultQueue = 256

// Service runs the games: it starts them on the engine, connects players and
// spectators to them, and saves them when they end. It is safe for
// concurrent use. Run must be running for finished games to be saved.
type Service struct {
	db        *gorm.DB
	catalogue *cards.Catalogue
	settings  *settings.Service
	rater     Rater
	logger    *slog.Logger
	system    *match.MatchSystem
	// accounts reads who is chatting, so a mute applies as soon as an admin
	// gives it. chatGate refuses muted players and limits how fast each player
	// chats in games, all their games together, as fast as in the lobby chat
	// but counted on its own.
	accounts *users.Service
	chatGate *chat.Gate

	mu        sync.Mutex
	live      map[string]*liveGame
	listeners []Listener
	closing   bool
	// computerGames counts the games against the computer running or being
	// started, for the setting that limits them.
	computerGames int
	// connections are the open connections to games, by user, so a player
	// put out of the site loses those they watch.
	connections map[string]map[*connection]struct{}

	results chan finished
	// computers are the goroutines of the computer players, which Close waits
	// for. They are added to under mu, and never once closing is set.
	computers sync.WaitGroup
}

var _ match.Observer = (*Service)(nil)

// NewService returns the games service, saving the games that count with
// the ratings rater gives them. Listeners join with Listen.
func NewService(db *gorm.DB, catalogue *cards.Catalogue, siteSettings *settings.Service, rater Rater, logger *slog.Logger) *Service {
	registerCards()

	s := &Service{
		db:          db,
		catalogue:   catalogue,
		settings:    siteSettings,
		rater:       rater,
		logger:      logger,
		live:        map[string]*liveGame{},
		connections: map[string]map[*connection]struct{}{},
		results:     make(chan finished, resultQueue),
		accounts:    users.NewService(db, logger),
		chatGate:    chat.NewGate(siteSettings, time.Now),
	}
	s.system = match.NewSystem(match.SystemOptions{Observer: s})

	return s
}

// Listen adds a listener to hear about games ending, before the first game
// starts.
func (s *Service) Listen(listener Listener) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.listeners = append(s.listeners, listener)
}

// notify tells every listener.
func (s *Service) notify(tell func(Listener)) {
	s.mu.Lock()
	listeners := slices.Clone(s.listeners)
	s.mu.Unlock()

	for _, listener := range listeners {
		tell(listener)
	}
}

// Run closes games that never get going and saves finished games, until ctx
// ends. Close the service first, so its last games are saved too: Run saves
// what is waiting before it returns.
func (s *Service) Run(ctx context.Context) {
	ticker := make(chan struct{})
	go func() {
		defer close(ticker)
		s.system.RunTicker(ctx)
	}()

	for {
		select {
		case done := <-s.results:
			s.save(done)
		case <-ctx.Done():
			for {
				select {
				case done := <-s.results:
					s.save(done)
				default:
					<-ticker
					return
				}
			}
		}
	}
}

// CheckDeck lists every way a deck breaks the format, with the same check
// Start makes: nothing for a deck that can be played.
func (s *Service) CheckDeck(ctx context.Context, deck decks.Deck, format formats.Format) ([]decks.Problem, error) {
	known, err := s.catalogue.Cards(ctx, cardIDs(deck.Cards))
	if err != nil {
		return nil, err
	}

	return decks.Check(deck.Cards, known, format), nil
}

// Civilizations lists the civilizations of a deck's cards, in the game's
// order, as a game shows its players' decks.
func (s *Service) Civilizations(ctx context.Context, deck decks.Deck) ([]string, error) {
	known, err := s.catalogue.Cards(ctx, cardIDs(deck.Cards))
	if err != nil {
		return nil, err
	}

	return decks.Civilizations(deck.Cards, known), nil
}

// Start checks both decks against the format and starts the game. Players
// then connect to it, the computer at once for a seat it plays, and it begins
// with the coin toss. It returns an *IllegalDeckError for a deck breaking the
// format, ErrClosed once the server is stopping, and ErrComputerGamesFull for
// a game against the computer while as many as the settings allow are running.
func (s *Service) Start(ctx context.Context, input StartInput) (Game, error) {
	againstComputer := input.Host.Computer != nil || input.Guest.Computer != nil

	s.mu.Lock()
	switch {
	case s.closing:
		s.mu.Unlock()
		return Game{}, ErrClosed
	case againstComputer && int64(s.computerGames) >= s.settings.Number(MaxComputerGames):
		s.mu.Unlock()
		return Game{}, ErrComputerGamesFull
	case againstComputer:
		s.computerGames++
	}
	s.mu.Unlock()

	game, err := s.start(ctx, input, againstComputer)
	if err != nil && againstComputer {
		s.mu.Lock()
		s.computerGames--
		s.mu.Unlock()
	}
	return game, err
}

// start starts a game Start has let through.
func (s *Service) start(ctx context.Context, input StartInput, againstComputer bool) (Game, error) {
	ids := slices.Concat(cardIDs(input.Host.Deck.Cards), cardIDs(input.Guest.Deck.Cards))
	known, err := s.catalogue.Cards(ctx, ids)
	if err != nil {
		return Game{}, err
	}

	seats := make([]match.Seat, 0, 2)
	players := make([]Player, 0, 2)
	for _, entrant := range []Entrant{input.Host, input.Guest} {
		if problems := decks.Check(entrant.Deck.Cards, known, input.Format); len(problems) > 0 {
			return Game{}, &IllegalDeckError{Username: entrant.User.Username, Problems: problems}
		}
		seats = append(seats, match.Seat{ID: entrant.User.ID, Username: entrant.User.Username, Deck: engineDeck(entrant.Deck.Cards, known)})
		players = append(players, Player{
			ID:            entrant.User.ID,
			Username:      entrant.User.Username,
			DeckID:        entrant.Deck.ID,
			DeckName:      entrant.Deck.Name,
			Civilizations: decks.Civilizations(entrant.Deck.Cards, known),
			Computer:      entrant.Computer != nil,
		})
	}

	m, err := s.system.NewMatch(match.Options{
		Name:                   input.Name,
		Host:                   seats[0],
		Guest:                  seats[1],
		Format:                 match.FormatDescriptor{ID: input.Format.ID, Name: input.Format.Name},
		Visible:                input.Visible,
		Matchmaking:            input.Matchmaking,
		TossCaller:             input.TossCaller,
		MinimumCountedDuration: time.Duration(s.settings.Number(MinimumCountedDuration)) * time.Second,
		Practice:               againstComputer,
		DisconnectGrace:        time.Duration(s.settings.Number(DisconnectGrace)) * time.Second,
		ChatRefusal:            s.chatRefusal,
	})
	if err != nil {
		return Game{}, fmt.Errorf("starting a game: %w", err)
	}

	game := Game{
		ID:          m.ID,
		Name:        input.Name,
		FormatID:    input.Format.ID,
		FormatName:  input.Format.Name,
		Host:        players[0],
		Guest:       players[1],
		Visible:     input.Visible,
		Spectators:  input.Spectators,
		Matchmaking: input.Matchmaking,
		StartedAt:   time.Now(),
	}

	s.mu.Lock()
	if s.closing {
		// Close did not see this game, so it would not stop it.
		s.mu.Unlock()
		m.Dispose()
		return Game{}, ErrClosed
	}
	s.live[m.ID] = &liveGame{game: game, match: m, hostDeck: input.Host.Deck.Cards, guestDeck: input.Guest.Deck.Cards}
	for _, entrant := range []Entrant{input.Host, input.Guest} {
		if entrant.Computer != nil {
			s.seatComputer(entrant, m.ID)
		}
	}
	s.mu.Unlock()

	s.logger.InfoContext(ctx, "started a game", "game", game.ID, "host", game.Host.Username, "guest", game.Guest.Username, "format", game.FormatName)

	return game, nil
}

// seatComputer has the computer play a seat of a game, joining it as a
// browser would, until the game closes its connection. s.mu must be held.
func (s *Service) seatComputer(entrant Entrant, gameID string) {
	logger := s.logger.With("game", gameID, "computer", entrant.User.Username)

	options := *entrant.Computer
	options.Logger = logger
	player := computer.NewSeat(options)

	s.computers.Go(player.Play)
	s.computers.Go(func() {
		if err := s.Connect(player.Connection(), entrant.User, gameID); err != nil {
			logger.Warn("the computer could not take its seat", "error", err)
		}
	})
}

// Game returns a running game.
func (s *Service) Game(id string) (Game, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	live, ok := s.live[id]
	if !ok {
		return Game{}, false
	}
	return live.game, true
}

// Games lists the running games, the oldest first.
func (s *Service) Games() []Game {
	s.mu.Lock()
	games := make([]Game, 0, len(s.live))
	for _, live := range s.live {
		games = append(games, live.game)
	}
	s.mu.Unlock()

	slices.SortFunc(games, func(a, b Game) int { return a.StartedAt.Compare(b.StartedAt) })
	return games
}

// PlayingIn returns the running game a user plays in, if any.
func (s *Service) PlayingIn(userID string) (Game, bool) {
	for _, game := range s.Games() {
		if game.Has(userID) {
			return game, true
		}
	}
	return Game{}, false
}

// RoleIn is how a user may take part in a running game: as one of its
// players, or as a spectator of a game that allows them. It returns
// ErrNotFound for a game that is not running and ErrNotWatchable for one the
// user may not watch.
func (s *Service) RoleIn(userID, gameID string) (Role, error) {
	game, ok := s.Game(gameID)
	switch {
	case !ok:
		return "", ErrNotFound
	case game.Has(userID):
		return RolePlayer, nil
	case game.Watchable():
		return RoleSpectator, nil
	default:
		return "", ErrNotWatchable
	}
}

// Connect plays or watches a game over a connection, already opened, until it
// closes. It returns ErrNotFound and ErrNotWatchable as RoleIn does, closing
// the connection.
func (s *Service) Connect(conn server.Connection, user users.User, gameID string) error {
	if _, err := s.RoleIn(user.ID, gameID); err != nil {
		_ = conn.Close()
		return err
	}

	s.mu.Lock()
	live, ok := s.live[gameID]
	s.mu.Unlock()
	if !ok {
		_ = conn.Close()
		return ErrNotFound
	}

	paced := newPacedConnection(conn, s.settings)
	open := s.track(user.ID, paced)
	defer s.untrack(user.ID, open)

	// The engine takes the connection over until it closes; the duel board
	// joins with its first message.
	server.NewSocket(paced, live.match, user.ID, user.Username).Listen()
	return nil
}

// chatUnavailable is what a player chatting in a game is told when who they
// are cannot be read.
const chatUnavailable = "The chat is not available right now. Try again in a moment."

// chatLookupTimeout bounds reading who is chatting.
const chatLookupTimeout = 5 * time.Second

// chatRefusal refuses the chat message of a player muted by an admin or
// chatting faster than the chat settings allow, telling them why, and lets
// everyone else's through.
func (s *Service) chatRefusal(userID string) string {
	ctx, cancel := context.WithTimeout(context.Background(), chatLookupTimeout)
	defer cancel()

	author, err := s.accounts.ByID(ctx, userID)
	if err != nil {
		s.logger.Error("reading who is chatting in a game", "user", userID, "error", err)
		return chatUnavailable
	}
	return chat.Refusal(s.chatGate.Check(author))
}

// connection is an open connection to a game.
type connection struct {
	conn server.Connection
}

func (s *Service) track(userID string, conn server.Connection) *connection {
	open := &connection{conn: conn}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.connections[userID] == nil {
		s.connections[userID] = map[*connection]struct{}{}
	}
	s.connections[userID][open] = struct{}{}
	return open
}

func (s *Service) untrack(userID string, open *connection) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.connections[userID], open)
	if len(s.connections[userID]) == 0 {
		delete(s.connections, userID)
	}
}

// Stop ends a running game without a winner, as an admin may, telling its
// players and spectators so. It does not count. It returns ErrNotFound for a
// game that is not running.
func (s *Service) Stop(gameID string) (Game, error) {
	s.mu.Lock()
	live, ok := s.live[gameID]
	s.mu.Unlock()
	if !ok {
		return Game{}, ErrNotFound
	}

	live.match.Abort(stoppedMessage)
	s.logger.Info("stopped a game", "game", gameID)
	return live.game, nil
}

// Expel stops the game a player plays in, if any, and closes their
// connections to the games they watch, as when they are banned.
func (s *Service) Expel(userID string) {
	if game, playing := s.PlayingIn(userID); playing {
		// Stop fails only for a game that has ended meanwhile.
		_, _ = s.Stop(game.ID)
	}

	s.mu.Lock()
	open := make([]*connection, 0, len(s.connections[userID]))
	for each := range s.connections[userID] {
		open = append(open, each)
	}
	s.mu.Unlock()

	// Closing a connection ends the engine's reading of it, which lets the
	// spectator go.
	for _, each := range open {
		_ = each.conn.Close()
	}
}

// Close stops every running game without a winner and refuses new ones,
// waiting until they have gone, with the computer players in them, or ctx
// ends. Run then saves them.
func (s *Service) Close(ctx context.Context) error {
	s.mu.Lock()
	s.closing = true
	running := make([]*match.Match, 0, len(s.live))
	for _, live := range s.live {
		running = append(running, live.match)
	}
	s.mu.Unlock()

	for _, m := range running {
		m.Abort(abortMessage)
	}

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		s.mu.Lock()
		left := len(s.live)
		s.mu.Unlock()
		if left == 0 {
			break
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("stopping games: %d still running: %w", left, ctx.Err())
		case <-ticker.C:
		}
	}

	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		s.computers.Wait()
	}()
	select {
	case <-stopped:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("stopping the computer players: %w", ctx.Err())
	}
}

// MatchFinished tells the listeners a game has ended, and queues it to be
// saved. The engine calls it, before the match closes.
func (s *Service) MatchFinished(result match.Result) {
	s.mu.Lock()
	live, ok := s.live[result.MatchID]
	s.mu.Unlock()
	if !ok {
		s.logger.Error("a game finished that the server did not start", "game", result.MatchID)
		return
	}

	ended := Ended{Game: live.game, WinnerID: result.WinnerID, Ending: Ending(result.Ending)}
	s.notify(func(l Listener) { l.GameEnded(ended) })

	s.results <- finished{result: result, live: *live}
}

// MatchClosed forgets a game that is no longer running. The engine calls it.
func (s *Service) MatchClosed(matchID string) {
	s.mu.Lock()
	live, ok := s.live[matchID]
	delete(s.live, matchID)
	if ok && live.game.AgainstComputer() {
		s.computerGames--
	}
	s.mu.Unlock()

	if ok {
		s.notify(func(l Listener) { l.GameClosed(live.game) })
	}
}

// engineDeck is a deck as the engine plays it: the engine id of every copy.
func engineDeck(entries []decks.Entry, known map[string]cards.Card) []string {
	deck := make([]string, 0, decks.Size)
	for _, entry := range entries {
		for range entry.Count {
			deck = append(deck, known[entry.CardID].EngineID)
		}
	}
	return deck
}

func cardIDs(entries []decks.Entry) []string {
	ids := make([]string, 0, len(entries))
	for _, entry := range entries {
		ids = append(ids, entry.CardID)
	}
	return ids
}
