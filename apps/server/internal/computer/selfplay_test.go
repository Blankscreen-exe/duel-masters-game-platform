package computer_test

import (
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/cards"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/cards/cardstest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/computer"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/decks"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/server"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/tests/scenario"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/games"
)

// gameTimeout bounds a whole game between two computer players, which takes
// well under a second unless something is stuck.
const gameTimeout = 2 * time.Minute

// realLibrary is the library of every card the engine plays, from the card
// database linked into the repository. It also registers the engine's cards,
// as the server does at startup, so games can be played with them; call it
// before games start.
func realLibrary(t *testing.T) *computer.Library {
	t.Helper()

	scenario.RegisterCards()

	source, err := cards.ReadSource(cardstest.RealDatabase(t))
	require.NoError(t, err)
	built, problems := cards.Build(source, games.PlayableCards())
	require.Empty(t, problems)
	return computer.NewLibrary(built)
}

// deckOf is a starter deck as the engine takes it: four of each card, by the
// engine's id.
func deckOf(t *testing.T, library *computer.Library, list decks.StarterList) []string {
	t.Helper()

	deck := make([]string, 0, len(list.Cards)*decks.MaxCopies)
	for _, name := range list.Cards {
		card, ok := library.Named(name)
		require.True(t, ok, "the engine plays %s", name)
		for range decks.MaxCopies {
			deck = append(deck, card.EngineID)
		}
	}
	return deck
}

// lockedBuffer is a log written from several goroutines.
type lockedBuffer struct {
	mu    sync.Mutex
	lines []string
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.lines = append(b.lines, strings.TrimRight(string(p), "\n"))
	return len(p), nil
}

// tail is the last n lines written.
func (b *lockedBuffer) tail(n int) string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return strings.Join(b.lines[max(len(b.lines)-n, 0):], "\n")
}

// results keeps the results the match reports.
type results struct {
	mu       sync.Mutex
	finished []match.Result
	closed   chan struct{}
	once     sync.Once
}

func (r *results) MatchFinished(result match.Result) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.finished = append(r.finished, result)
}

func (r *results) MatchClosed(string) {
	r.once.Do(func() { close(r.closed) })
}

// playGame has two computer players play a whole game with the decks, and
// returns how it ended once both have stopped playing.
func playGame(t *testing.T, library *computer.Library, hostDeck, guestDeck []string, seed uint64) match.Result {
	t.Helper()

	reported := &results{closed: make(chan struct{})}
	m, err := match.NewSystem(match.SystemOptions{Observer: reported}).NewMatch(match.Options{
		Name:            "Computer against computer",
		Host:            match.Seat{ID: "host", Username: "Host", Deck: hostDeck},
		Guest:           match.Seat{ID: "guest", Username: "Guest", Deck: guestDeck},
		TossCaller:      "host",
		DisconnectGrace: time.Hour,
	})
	require.NoError(t, err)
	t.Cleanup(m.Dispose)

	// What the players did, shown if the game gets stuck.
	var log lockedBuffer
	var playing sync.WaitGroup
	for i, seat := range []match.Seat{{ID: "host", Username: "Host"}, {ID: "guest", Username: "Guest"}} {
		player := computer.New(computer.Options{
			Library: library,
			Style:   computer.Balanced,
			Seed:    seed + uint64(i),
			Logger:  slog.New(slog.NewTextHandler(&log, &slog.HandlerOptions{Level: slog.LevelDebug})).With("seat", seat.ID),
		})
		socket := server.NewSocket(player.Connection(), m, seat.ID, seat.Username)
		playing.Add(2)
		go func() {
			defer playing.Done()
			socket.Listen()
		}()
		go func() {
			defer playing.Done()
			player.Play()
		}()
	}

	select {
	case <-reported.closed:
	case <-time.After(gameTimeout):
		t.Fatalf("the game did not end within %s; the last the players did:\n%s", gameTimeout, log.tail(60))
	}

	// Closing the match closes both players' connections, which ends their
	// play: nothing of theirs outlives the game.
	stopped := make(chan struct{})
	go func() {
		playing.Wait()
		close(stopped)
	}()
	select {
	case <-stopped:
	case <-time.After(10 * time.Second):
		t.Fatal("the computer players went on after the game closed")
	}

	reported.mu.Lock()
	defer reported.mu.Unlock()
	require.Len(t, reported.finished, 1, "the game reported its end")
	return reported.finished[0]
}

func TestTheComputerPlaysWholeGamesWithEveryStarterDeck(t *testing.T) {
	library := realLibrary(t)
	starters := decks.StartingStarters()

	for i, host := range starters {
		guest := starters[(i+1)%len(starters)]
		for seed := uint64(1); seed <= 2; seed++ {
			t.Run(fmt.Sprintf("%s against %s, game %d", host.Name, guest.Name, seed), func(t *testing.T) {
				t.Parallel()

				result := playGame(t, library, deckOf(t, library, host), deckOf(t, library, guest), seed*100)

				assert.Equal(t, match.EndingPlay, result.Ending, "won by the game's rules, nobody giving up")
				assert.NotEmpty(t, result.WinnerID)
				assert.Greater(t, result.Turns, 4, "a whole game, not one lost at once")
				t.Logf("%s won after %d turns", result.WinnerID, result.Turns)
			})
		}
	}
}
