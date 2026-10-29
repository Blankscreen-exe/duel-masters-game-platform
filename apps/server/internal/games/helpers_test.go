package games_test

import (
	"context"
	"slices"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/chat"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/decks"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/formats"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/games"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/games/gamestest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/ratings"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/settings"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/store/storetest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/users"
)

const waitTimeout = gamestest.Timeout

// heard keeps what the service tells its listener.
type heard struct {
	mu       sync.Mutex
	ended    []games.Ended
	finished []games.Record
	closed   []games.Game
	// order is what was heard, in order: "ended", "finished" or "closed".
	order []string
}

func (h *heard) GameEnded(ended games.Ended) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.ended = append(h.ended, ended)
	h.order = append(h.order, "ended")
}

func (h *heard) GameFinished(record games.Record) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.finished = append(h.finished, record)
	h.order = append(h.order, "finished")
}

func (h *heard) GameClosed(game games.Game) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.closed = append(h.closed, game)
	h.order = append(h.order, "closed")
}

func (h *heard) records() []games.Record {
	h.mu.Lock()
	defer h.mu.Unlock()
	return slices.Clone(h.finished)
}

func (h *heard) endedGames() []games.Ended {
	h.mu.Lock()
	defer h.mu.Unlock()
	return slices.Clone(h.ended)
}

func (h *heard) heardInOrder() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return slices.Clone(h.order)
}

func (h *heard) closedGames() []games.Game {
	h.mu.Lock()
	defer h.mu.Unlock()
	return slices.Clone(h.closed)
}

// rater rates games with the real ratings, unless a test has made it fail.
type rater struct {
	ratings *ratings.Service

	mu      sync.Mutex
	failure error
	// asked counts the games it was asked to rate.
	asked int
}

var _ games.Rater = (*rater)(nil)

func (r *rater) Rate(tx *gorm.DB, game games.Record) (host, guest games.RatingChange, err error) {
	r.mu.Lock()
	failure := r.failure
	r.asked++
	r.mu.Unlock()
	if failure != nil {
		return games.RatingChange{}, games.RatingChange{}, failure
	}
	return r.ratings.Rate(tx, game)
}

// timesAsked is how many games it was asked to rate.
func (r *rater) timesAsked() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.asked
}

// fail makes rating fail from now on, or with nil work again.
func (r *rater) fail(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.failure = err
}

// fixture is a games service with three players and the formats.
type fixture struct {
	db       *gorm.DB
	games    *games.Service
	heard    *heard
	settings *settings.Service
	rater    *rater
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

	catalogue, ids := gamestest.RealCards(t, db)

	siteSettings, err := settings.NewService(ctx, db, storetest.Logger(), slices.Concat(games.Settings(), chat.Settings(), decks.Settings(), ratings.Settings())...)
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
	rater := &rater{ratings: ratings.NewService(db, siteSettings)}
	service := games.NewService(db, catalogue, siteSettings, rater, storetest.Logger())
	service.Listen(listener)
	gamestest.Run(t, service)

	return fixture{
		db: db, games: service, heard: listener, settings: siteSettings, rater: rater,
		open: list[0], classic: list[1],
		shobu: shobu, kokujo: kokujo, watcher: watcher,
		cardIDs: ids,
	}
}

// standing is a player's rating columns, as a test reads them.
type standing struct{ Rating, Wins, Losses, Streak int }

func (f fixture) standing(t *testing.T, userID string) standing {
	t.Helper()

	var row standing
	require.NoError(t, f.db.Table("users").Select("rating", "wins", "losses", "streak").Where("id = ?", userID).Take(&row).Error)
	return row
}

// deck is a legal deck: 4 copies of each of the ten cards.
func (f fixture) deck(name string) decks.Deck {
	return decks.Deck{ID: name, Name: name, Cards: gamestest.LegalDeck(f.cardIDs)}
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
