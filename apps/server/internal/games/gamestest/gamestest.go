// Package gamestest helps tests play real games: a catalogue of real engine
// cards, and connections a test talks to games over, as the duel board would.
package gamestest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/cards"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/cards/cardstest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/decks"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/games"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/store/storetest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/users"
)

// Timeout bounds every wait for something a game does on its own.
const Timeout = 5 * time.Second

// RealCards copies into db a card database of ten real engine cards from
// DM-01, enough for a legal deck that the engine really plays, and returns
// its catalogue and the cards' ids.
func RealCards(t *testing.T, db *gorm.DB) (*cards.Catalogue, []string) {
	t.Helper()

	var dm01 []cards.EngineCard
	for _, card := range games.PlayableCards() {
		if card.Set == "dm-01" {
			dm01 = append(dm01, card)
		}
	}
	require.GreaterOrEqual(t, len(dm01), 10)
	dm01 = dm01[:10]

	printings := make([]map[string]any, 0, len(dm01))
	ids := make([]string, 0, len(dm01))
	for i, card := range dm01 {
		id := fmt.Sprintf("%c001", 'A'+i)
		printings = append(printings, cardstest.Printing(id, card.Name, "DM-01", id+"/110"))
		ids = append(ids, id)
	}

	dir := cardstest.Write(t, printings, ids, nil)
	source, err := cards.ReadSource(dir)
	require.NoError(t, err)
	_, err = cards.Sync(context.Background(), db, source, dm01, storetest.Logger())
	require.NoError(t, err)

	return cards.NewCatalogue(db, dir), ids
}

// LegalDeck is 4 copies of each of the cards: 40 cards.
func LegalDeck(ids []string) []decks.Entry {
	entries := make([]decks.Entry, 0, len(ids))
	for _, id := range ids {
		entries = append(entries, decks.Entry{CardID: id, Count: 4})
	}
	return entries
}

// Run runs the games service until the test ends, then closes it, so its
// games are saved.
func Run(t *testing.T, service *games.Service) {
	t.Helper()

	ctx, stop := context.WithCancel(context.Background())
	running := make(chan struct{})
	go func() {
		defer close(running)
		service.Run(ctx)
	}()

	t.Cleanup(func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), Timeout)
		defer cancel()
		require.NoError(t, service.Close(closeCtx))
		stop()
		<-running
	})
}

// Pipe is a connection a test types into, as the duel board would, and reads
// what the engine sent back. It satisfies the engine's server.Connection.
type Pipe struct {
	incoming chan []byte
	closed   chan struct{}
	once     sync.Once

	mu   sync.Mutex
	sent []map[string]any
	// arrived is closed, and replaced, whenever a message arrives, to wake
	// whoever waits for one.
	arrived chan struct{}
}

// NewPipe returns an open pipe.
func NewPipe() *Pipe {
	return &Pipe{incoming: make(chan []byte, 16), closed: make(chan struct{}), arrived: make(chan struct{})}
}

func (p *Pipe) SetReadLimit(int64)                {}
func (p *Pipe) SetReadDeadline(time.Time) error   { return nil }
func (p *Pipe) SetPongHandler(func(string) error) {}
func (p *Pipe) SetWriteDeadline(time.Time) error  { return nil }
func (p *Pipe) WriteMessage(int, []byte) error    { return nil }

func (p *Pipe) ReadMessage() (int, []byte, error) {
	select {
	case data := <-p.incoming:
		return 1, data, nil
	case <-p.closed:
		return 0, nil, errors.New("closed")
	}
}

func (p *Pipe) WriteJSON(v any) error {
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
	close(p.arrived)
	p.arrived = make(chan struct{})
	return nil
}

// find returns the first message with the header, and a channel closed when
// the next message arrives.
func (p *Pipe) find(header string) (map[string]any, <-chan struct{}) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, message := range p.sent {
		if message["header"] == header {
			return message, nil
		}
	}
	return nil, p.arrived
}

func (p *Pipe) Close() error {
	p.once.Do(func() { close(p.closed) })
	return nil
}

// IsClosed reports whether the pipe has been closed.
func (p *Pipe) IsClosed() bool {
	select {
	case <-p.closed:
		return true
	default:
		return false
	}
}

// Say sends a message to the game, as the duel board would.
func (p *Pipe) Say(message map[string]any) {
	data, err := json.Marshal(message)
	if err != nil {
		panic(err)
	}
	p.incoming <- data
}

// WaitFor waits for a message with the header and returns it.
func (p *Pipe) WaitFor(t *testing.T, header string) map[string]any {
	t.Helper()

	deadline := time.After(Timeout)
	for {
		message, arrived := p.find(header)
		if message != nil {
			return message
		}

		select {
		case <-arrived:
		case <-deadline:
			t.Fatalf("expected a %q message", header)
			return nil
		}
	}
}

// Connect plays or watches a game over a new pipe, joining as the duel board
// does. Connect's error, if any, arrives on the channel once it returns.
func Connect(service *games.Service, user users.User, gameID string) (*Pipe, <-chan error) {
	pipe := NewPipe()
	done := make(chan error, 1)
	go func() { done <- service.Connect(pipe, user, gameID) }()
	pipe.Say(map[string]any{"header": "join_match"})
	return pipe, done
}

// Play connects a game's two players and takes it through the coin toss:
// the caller calls heads and whoever wins plays first. It returns the
// host's and guest's pipes once play has begun.
func Play(t *testing.T, service *games.Service, game games.Game) (host, guest *Pipe) {
	t.Helper()

	host, _ = Connect(service, users.User{ID: game.Host.ID, Username: game.Host.Username}, game.ID)
	guest, _ = Connect(service, users.User{ID: game.Guest.ID, Username: game.Guest.Username}, game.ID)

	// Whoever is asked to call does so.
	caller := callerOf(t, host, guest)
	caller.Say(map[string]any{"header": "toss_chosen", "prediction": 1})
	result := host.WaitFor(t, "toss_result")

	winner := host
	if result["winner"].(map[string]any)["uid"] == game.Guest.ID {
		winner = guest
	}
	winner.WaitFor(t, "choose_turn")
	winner.Say(map[string]any{"header": "turn_chosen", "player": 1})
	host.WaitFor(t, "state_update")

	return host, guest
}

// callerOf waits until one of the players is asked to call the coin toss,
// and returns them.
func callerOf(t *testing.T, host, guest *Pipe) *Pipe {
	t.Helper()

	deadline := time.After(Timeout)
	for {
		hostAsked, hostArrived := host.find("choose_toss")
		guestAsked, guestArrived := guest.find("choose_toss")
		switch {
		case hostAsked != nil:
			return host
		case guestAsked != nil:
			return guest
		}

		select {
		case <-hostArrived:
		case <-guestArrived:
		case <-deadline:
			t.Fatal("nobody was asked to call the coin toss")
			return nil
		}
	}
}
