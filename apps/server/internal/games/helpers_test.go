package games_test

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/cards"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/cards/cardstest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/decks"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/formats"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/games"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/settings"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/store/storetest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/users"
)

const waitTimeout = 5 * time.Second

// heard keeps what the service tells its listener.
type heard struct {
	mu       sync.Mutex
	finished []games.Record
	closed   []games.Game
}

func (h *heard) GameFinished(record games.Record) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.finished = append(h.finished, record)
}

func (h *heard) GameClosed(game games.Game) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.closed = append(h.closed, game)
}

func (h *heard) records() []games.Record {
	h.mu.Lock()
	defer h.mu.Unlock()
	return slices.Clone(h.finished)
}

func (h *heard) closedGames() []games.Game {
	h.mu.Lock()
	defer h.mu.Unlock()
	return slices.Clone(h.closed)
}

// fixture is a games service with two players and the formats.
type fixture struct {
	db       *gorm.DB
	games    *games.Service
	heard    *heard
	settings *settings.Service
	open     formats.Format
	classic  formats.Format
	shobu    users.User
	kokujo   users.User
	watcher  users.User
	// cardIDs are ten catalogue cards the engine plays, all from DM-01.
	cardIDs []string
}

func newFixture(t *testing.T, db *gorm.DB) fixture {
	t.Helper()
	ctx := context.Background()

	// Ten real engine cards from DM-01, so games are played with real cards.
	var dm01 []cards.EngineCard
	for _, card := range games.PlayableCards() {
		if card.Set == "dm-01" {
			dm01 = append(dm01, card)
		}
	}
	require.GreaterOrEqual(t, len(dm01), 10)
	dm01 = dm01[:10]

	printings := make([]map[string]any, 0, len(dm01))
	pictures := make([]string, 0, len(dm01))
	ids := make([]string, 0, len(dm01))
	for i, card := range dm01 {
		id := string(rune('A'+i)) + "001"
		printings = append(printings, cardstest.Printing(id, card.Name, "DM-01", id+"/110"))
		pictures = append(pictures, id)
		ids = append(ids, id)
	}
	dir := cardstest.Write(t, printings, pictures, nil)
	source, err := cards.ReadSource(dir)
	require.NoError(t, err)
	_, err = cards.Sync(ctx, db, source, dm01, storetest.Logger())
	require.NoError(t, err)

	siteSettings, err := settings.NewService(ctx, db, storetest.Logger(), slices.Concat(games.Settings(), decks.Settings())...)
	require.NoError(t, err)

	list, err := formats.NewService(db).List(ctx)
	require.NoError(t, err)

	accounts := users.NewService(db, storetest.Logger())
	shobu, err := accounts.Create(ctx, "Shobu", "hash")
	require.NoError(t, err)
	kokujo, err := accounts.Create(ctx, "Kokujo", "hash")
	require.NoError(t, err)
	watcher, err := accounts.Create(ctx, "Watcher", "hash")
	require.NoError(t, err)

	listener := &heard{}
	service := games.NewService(db, cards.NewCatalogue(db, dir), siteSettings, listener, storetest.Logger())

	runCtx, stop := context.WithCancel(context.Background())
	running := make(chan struct{})
	go func() {
		defer close(running)
		service.Run(runCtx)
	}()
	t.Cleanup(func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), waitTimeout)
		defer cancel()
		require.NoError(t, service.Close(closeCtx))
		stop()
		<-running
	})

	return fixture{
		db: db, games: service, heard: listener, settings: siteSettings,
		open: list[0], classic: list[1],
		shobu: shobu, kokujo: kokujo, watcher: watcher,
		cardIDs: ids,
	}
}

// deck is a legal deck: 4 copies of each of the ten cards.
func (f fixture) deck(name string) decks.Deck {
	entries := make([]decks.Entry, 0, len(f.cardIDs))
	for _, id := range f.cardIDs {
		entries = append(entries, decks.Entry{CardID: id, Count: 4})
	}
	return decks.Deck{ID: name, Name: name, Cards: entries}
}

// start starts a game between Shobu (the host, who calls the toss) and
// Kokujo.
func (f fixture) start(t *testing.T, visible, spectators bool) games.Game {
	t.Helper()

	game, err := f.games.Start(context.Background(), games.StartInput{
		Name:       "Friendly",
		Format:     f.open,
		Host:       games.Entrant{User: f.shobu, Deck: f.deck("Fire Rush")},
		Guest:      games.Entrant{User: f.kokujo, Deck: f.deck("Water Wall")},
		Visible:    visible,
		Spectators: spectators,
		TossCaller: f.shobu.ID,
	})
	require.NoError(t, err)
	return game
}

// pipe is a connection a test types into, as the duel board would, and reads
// what the engine sent back. It satisfies the engine's server.Connection.
type pipe struct {
	incoming chan []byte
	closed   chan struct{}
	once     sync.Once

	mu   sync.Mutex
	sent []map[string]any
}

func newPipe() *pipe {
	return &pipe{incoming: make(chan []byte, 16), closed: make(chan struct{})}
}

func (p *pipe) SetReadLimit(int64)                {}
func (p *pipe) SetReadDeadline(time.Time) error   { return nil }
func (p *pipe) SetPongHandler(func(string) error) {}
func (p *pipe) SetWriteDeadline(time.Time) error  { return nil }
func (p *pipe) WriteMessage(int, []byte) error    { return nil }

func (p *pipe) ReadMessage() (int, []byte, error) {
	select {
	case data := <-p.incoming:
		return 1, data, nil
	case <-p.closed:
		return 0, nil, errors.New("closed")
	}
}

func (p *pipe) WriteJSON(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	var message map[string]any
	if err := json.Unmarshal(data, &message); err != nil {
		return err
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	p.sent = append(p.sent, message)
	return nil
}

func (p *pipe) Close() error {
	p.once.Do(func() { close(p.closed) })
	return nil
}

func (p *pipe) isClosed() bool {
	select {
	case <-p.closed:
		return true
	default:
		return false
	}
}

func (p *pipe) say(message map[string]any) {
	data, err := json.Marshal(message)
	if err != nil {
		panic(err)
	}
	p.incoming <- data
}

// waitFor waits for a message with the header and returns it.
func (p *pipe) waitFor(t *testing.T, header string) map[string]any {
	t.Helper()

	var found map[string]any
	require.Eventually(t, func() bool {
		p.mu.Lock()
		defer p.mu.Unlock()
		for _, message := range p.sent {
			if message["header"] == header {
				found = message
				return true
			}
		}
		return false
	}, waitTimeout, 5*time.Millisecond, "expected a %q message", header)

	return found
}

// connect plays or watches a game over a new pipe, joining as the duel board
// does. It returns once the connection is open; connect's error, if any,
// arrives on the channel.
func connect(f fixture, user users.User, gameID string) (*pipe, <-chan error) {
	p := newPipe()
	done := make(chan error, 1)
	go func() { done <- f.games.Connect(p, user, gameID) }()
	p.say(map[string]any{"header": "join_match"})
	return p, done
}

// play takes a started game through the coin toss: Shobu calls heads, and
// whoever wins plays first. It returns both players' pipes.
func play(t *testing.T, f fixture, game games.Game) (shobu, kokujo *pipe) {
	t.Helper()

	shobu, _ = connect(f, f.shobu, game.ID)
	kokujo, _ = connect(f, f.kokujo, game.ID)
	shobu.waitFor(t, "choose_toss")
	shobu.say(map[string]any{"header": "toss_chosen", "prediction": 1})

	result := shobu.waitFor(t, "toss_result")
	winner := shobu
	if result["winner"].(map[string]any)["uid"] == f.kokujo.ID {
		winner = kokujo
	}
	winner.waitFor(t, "choose_turn")
	winner.say(map[string]any{"header": "turn_chosen", "player": 1})
	shobu.waitFor(t, "state_update")

	return shobu, kokujo
}
