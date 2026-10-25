package matchmaking_test

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/games"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/games/gamestest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/lobby"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/lobby/lobbytest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/matchmaking"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/store/storetest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/users"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/validation"
)

// heard keeps what quick match tells its listener.
type heard struct {
	mu      sync.Mutex
	waiting []map[string]int
	started []games.Game
}

func (h *heard) QueueChanged(waiting map[string]int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.waiting = append(h.waiting, waiting)
}

func (h *heard) GameStarted(game games.Game) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.started = append(h.started, game)
}

func (h *heard) latestWaiting() map[string]int {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.waiting) == 0 {
		return nil
	}
	return h.waiting[len(h.waiting)-1]
}

func (h *heard) startedGames() []games.Game {
	h.mu.Lock()
	defer h.mu.Unlock()
	return slices.Clone(h.started)
}

func newQuickMatch(t *testing.T, db *gorm.DB) (*lobbytest.World, *matchmaking.Service, *heard) {
	t.Helper()

	w := lobbytest.NewWorld(t, db)
	listener := &heard{}
	service := matchmaking.NewService(matchmaking.Dependencies{
		Lobby: w.Lobby, Games: w.Games, Directory: w.Directory, Listener: listener,
		Logger: storetest.Logger(), Now: w.Clock.Now, SweepEvery: 10 * time.Millisecond,
	})
	return w, service, listener
}

func TestTheFirstPlayerWaits(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		w, quick, listener := newQuickMatch(t, db)

		entered, err := quick.Enter(context.Background(), w.Shobu, w.Open.ID, w.Legal[w.Shobu.ID].ID)
		require.NoError(t, err)

		assert.Nil(t, entered.Game)
		assert.Equal(t, w.Shobu.ID, entered.Ticket.UserID)
		assert.Equal(t, "Open", entered.Ticket.FormatName)
		assert.Equal(t, lobby.Queued, w.Doing(w.Shobu))
		assert.Equal(t, map[string]int{w.Open.ID: 1}, quick.Waiting())
		assert.Equal(t, map[string]int{w.Open.ID: 1}, listener.latestWaiting(), "everyone sees one player waiting")

		ticket, ok := quick.Ticket(w.Shobu.ID)
		require.True(t, ok)
		assert.Equal(t, entered.Ticket, ticket)
	})
}

func TestTheNextPlayerInTheFormatIsPaired(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		w, quick, listener := newQuickMatch(t, db)
		_, err := quick.Enter(ctx, w.Shobu, w.Open.ID, w.Legal[w.Shobu.ID].ID)
		require.NoError(t, err)

		// A player in another format does not pair with him.
		other, err := quick.Enter(ctx, w.Mimi, w.Classic.ID, w.Legal[w.Mimi.ID].ID)
		require.NoError(t, err)
		assert.Nil(t, other.Game)

		entered, err := quick.Enter(ctx, w.Kokujo, w.Open.ID, w.Legal[w.Kokujo.ID].ID)
		require.NoError(t, err)
		require.NotNil(t, entered.Game)

		game := *entered.Game
		assert.Equal(t, "Quick match", game.Name)
		assert.Equal(t, w.Shobu.ID, game.Host.ID, "the player who waited longer hosts")
		assert.Equal(t, w.Kokujo.ID, game.Guest.ID)
		assert.True(t, game.Matchmaking)
		assert.True(t, game.Watchable())
		assert.Equal(t, []games.Game{game}, listener.startedGames(), "both are told to go to it")

		assert.Equal(t, lobby.Playing, w.Doing(w.Shobu))
		assert.Equal(t, lobby.Playing, w.Doing(w.Kokujo))
		assert.Equal(t, map[string]int{w.Classic.ID: 1}, quick.Waiting(), "only Mimi waits")

		// The game is played on the engine like any other.
		host, guest := gamestest.Play(t, w.Games, game)
		host.Say(map[string]any{"header": "resign"})
		guest.WaitFor(t, "duel_finished")
		require.Eventually(t, func() bool { return w.Doing(w.Shobu) == "" }, gamestest.Timeout, 5*time.Millisecond)
	})
}

func TestPlayersArrivingTogetherArePaired(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		w, quick, listener := newQuickMatch(t, db)

		var wg sync.WaitGroup
		for _, player := range []users.User{w.Shobu, w.Kokujo} {
			wg.Go(func() {
				_, err := quick.Enter(ctx, player, w.Open.ID, w.Legal[player.ID].ID)
				assert.NoError(t, err)
			})
		}
		wg.Wait()

		assert.Len(t, listener.startedGames(), 1, "never both left waiting")
		assert.Empty(t, quick.Waiting())
	})
}

func TestLeavingQuickMatch(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		w, quick, listener := newQuickMatch(t, db)
		_, err := quick.Enter(context.Background(), w.Shobu, w.Open.ID, w.Legal[w.Shobu.ID].ID)
		require.NoError(t, err)

		assert.True(t, quick.Leave(w.Shobu.ID))
		assert.False(t, quick.Leave(w.Shobu.ID), "already gone")

		assert.Empty(t, quick.Waiting())
		assert.Empty(t, listener.latestWaiting())
		assert.Empty(t, w.Doing(w.Shobu))
		_, ok := quick.Ticket(w.Shobu.ID)
		assert.False(t, ok)
	})
}

func TestAPlayerWhoLeavesTheSiteLeavesTheQueue(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		w, quick, _ := newQuickMatch(t, db)
		_, err := quick.Enter(context.Background(), w.Shobu, w.Open.ID, w.Legal[w.Shobu.ID].ID)
		require.NoError(t, err)

		w.Directory.Set(w.Shobu, false)
		quick.PresenceChanged(w.Shobu.ID)

		assert.Empty(t, quick.Waiting())
		assert.Empty(t, w.Doing(w.Shobu))
	})
}

func TestTheQueueSweepCatchesPlayersWhoLeft(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		w, quick, _ := newQuickMatch(t, db)
		_, err := quick.Enter(context.Background(), w.Shobu, w.Open.ID, w.Legal[w.Shobu.ID].ID)
		require.NoError(t, err)

		ctx, stop := context.WithCancel(context.Background())
		defer stop()
		go quick.Run(ctx)

		w.Directory.Set(w.Shobu, false)
		require.Eventually(t, func() bool { return len(quick.Waiting()) == 0 }, gamestest.Timeout, 5*time.Millisecond)
	})
}

func TestQuickMatchKeepsToTheRules(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		w, quick, _ := newQuickMatch(t, db)

		_, err := quick.Enter(ctx, w.Shobu, w.Open.ID, w.Short[w.Shobu.ID].ID)
		var illegal *games.IllegalDeckError
		require.True(t, errors.As(err, &illegal), "got %v", err)

		_, err = quick.Enter(ctx, w.Shobu, "no-such-format", w.Legal[w.Shobu.ID].ID)
		invalid, ok := validation.As(err)
		require.True(t, ok, "got %v", err)
		assert.Equal(t, "formatId", invalid.Field)

		// Hosting an open game, he cannot also wait here.
		_, err = w.Lobby.Create(ctx, w.Shobu, lobby.CreateInput{DeckID: w.Legal[w.Shobu.ID].ID, FormatID: w.Open.ID})
		require.NoError(t, err)
		_, err = quick.Enter(ctx, w.Shobu, w.Open.ID, w.Legal[w.Shobu.ID].ID)
		assert.True(t, lobby.IsBusy(err), "got %v", err)

		_, err = w.Settings.Set(ctx, lobby.NewGamesOpen.Key, "false")
		require.NoError(t, err)
		_, err = quick.Enter(ctx, w.Kokujo, w.Open.ID, w.Legal[w.Kokujo.ID].ID)
		assert.ErrorIs(t, err, lobby.ErrPaused)

		assert.Empty(t, quick.Waiting())
	})
}

func TestAWaitingPlayerWhoseDeckBrokeIsSkipped(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		w, quick, listener := newQuickMatch(t, db)
		_, err := quick.Enter(ctx, w.Shobu, w.Open.ID, w.Legal[w.Shobu.ID].ID)
		require.NoError(t, err)

		// While he waits, Shobu deletes the deck he waits with.
		require.NoError(t, w.Decks.Delete(ctx, w.Shobu.ID, w.Legal[w.Shobu.ID].ID))

		entered, err := quick.Enter(ctx, w.Kokujo, w.Open.ID, w.Legal[w.Kokujo.ID].ID)
		require.NoError(t, err)

		assert.Nil(t, entered.Game, "Kokujo waits rather than play a deck that is gone")
		assert.Equal(t, map[string]int{w.Open.ID: 1}, quick.Waiting())
		assert.Empty(t, w.Doing(w.Shobu), "Shobu left the queue")
		assert.Equal(t, lobby.Queued, w.Doing(w.Kokujo))
		assert.Empty(t, listener.startedGames())
	})
}
