package lobbytest

import (
	"context"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/cards"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/chat"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/clocktest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/decks"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/formats"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/games"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/games/gamestest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/lobby"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/ratings"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/settings"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/store/storetest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/users"
)

// Heard keeps what the lobby tells its listener. It is safe for concurrent
// use.
type Heard struct {
	mu        sync.Mutex
	snapshots []lobby.Snapshot
	started   []games.Game
	expired   []lobby.OpenGame
}

var _ lobby.Listener = (*Heard)(nil)

func (h *Heard) LobbyChanged(snapshot lobby.Snapshot) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.snapshots = append(h.snapshots, snapshot)
}

func (h *Heard) GameStarted(game games.Game) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.started = append(h.started, game)
}

func (h *Heard) OpenGameExpired(open lobby.OpenGame) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.expired = append(h.expired, open)
}

// Latest is the lobby as last shown, empty before anything was shown.
func (h *Heard) Latest() lobby.Snapshot {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.snapshots) == 0 {
		return lobby.Snapshot{}
	}
	return h.snapshots[len(h.snapshots)-1]
}

// Started are the games the players were told to go to.
// Changes counts the lobbies shown.
func (h *Heard) Changes() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.snapshots)
}

func (h *Heard) Started() []games.Game {
	h.mu.Lock()
	defer h.mu.Unlock()
	return slices.Clone(h.started)
}

// Expired are the open games whose hosts were told nobody joined.
func (h *Heard) Expired() []lobby.OpenGame {
	h.mu.Lock()
	defer h.mu.Unlock()
	return slices.Clone(h.expired)
}

// World is a lobby and its games over real engine cards: three players
// online, Shobu, Kokujo and Mimi, each with a legal deck and an unfinished
// one.
type World struct {
	DB        *gorm.DB
	Lobby     *lobby.Service
	Games     *games.Service
	Ratings   *ratings.Service
	Decks     *decks.Service
	Formats   *formats.Service
	Catalogue *cards.Catalogue
	Settings  *settings.Service
	Directory *Directory
	Heard     *Heard
	Clock     *clocktest.Clock
	Open      formats.Format
	Classic   formats.Format
	Shobu     users.User
	Kokujo    users.User
	Mimi      users.User
	// Legal is each player's legal deck, by user id; Short their unfinished
	// one, of a few cards.
	Legal map[string]decks.Deck
	Short map[string]decks.Deck
	// CardIDs are the cards the decks are made of.
	CardIDs []string
}

// NewWorld builds the world on db, with the extra settings of features built
// on the lobby. The games service runs until the test ends; the lobby's
// sweeps run only when a test runs them.
func NewWorld(t *testing.T, db *gorm.DB, extra ...settings.Definition) *World {
	t.Helper()
	ctx := context.Background()

	catalogue, ids := gamestest.RealCards(t, db)
	siteSettings, err := settings.NewService(ctx, db, storetest.Logger(), slices.Concat(lobby.Settings(), games.Settings(), chat.Settings(), decks.Settings(), ratings.Settings(), extra)...)
	require.NoError(t, err)
	formatService := formats.NewService(db)
	list, err := formatService.List(ctx)
	require.NoError(t, err)

	w := &World{
		DB:        db,
		Decks:     decks.NewService(db, catalogue, siteSettings, storetest.Logger()),
		Formats:   formatService,
		Catalogue: catalogue,
		Settings:  siteSettings,
		Directory: NewDirectory(),
		Heard:     &Heard{},
		Clock:     clocktest.New(time.Now()),
		Open:      list[0],
		Classic:   list[1],
		Legal:     map[string]decks.Deck{},
		Short:     map[string]decks.Deck{},
		CardIDs:   ids,
	}

	accounts := users.NewService(db, storetest.Logger())
	newPlayer := func(name string) users.User {
		player, err := accounts.Create(ctx, name, "hash")
		require.NoError(t, err)

		w.Legal[player.ID], err = w.Decks.Create(ctx, player.ID, decks.Input{Name: name + "'s deck", Cards: gamestest.LegalDeck(ids)})
		require.NoError(t, err)
		w.Short[player.ID], err = w.Decks.Create(ctx, player.ID, decks.Input{Name: "Unfinished", Cards: gamestest.LegalDeck(ids)[:3]})
		require.NoError(t, err)

		w.Directory.Set(player, true)
		return player
	}
	w.Shobu = newPlayer("Shobu")
	w.Kokujo = newPlayer("Kokujo")
	w.Mimi = newPlayer("Mimi")

	w.Ratings = ratings.NewService(db, siteSettings)
	w.Games = games.NewService(db, catalogue, siteSettings, w.Ratings, storetest.Logger())
	w.Lobby = lobby.NewService(lobby.Dependencies{
		Decks:      w.Decks,
		Formats:    formatService,
		Games:      w.Games,
		Directory:  w.Directory,
		Settings:   siteSettings,
		Listener:   w.Heard,
		Logger:     storetest.Logger(),
		Now:        w.Clock.Now,
		SweepEvery: 10 * time.Millisecond,
	})
	gamestest.Run(t, w.Games)

	return w
}

// Doing is what a player is doing, empty for nothing.
func (w *World) Doing(user users.User) lobby.Kind {
	activity, ok := w.Lobby.Activities().Of(user.ID)
	if !ok {
		return ""
	}
	return activity.Kind
}
