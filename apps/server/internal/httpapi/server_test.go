package httpapi

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/auth"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/cards"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/cards/cardstest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/config"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/decks"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/formats"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/games"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/realtime"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/settings"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/store/storetest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/users"
)

// testSiteOrigin is the site the test server accepts live connections from.
const testSiteOrigin = "http://site.test"

// testServer is the API with every feature, on a fresh SQLite database and a
// small card database. The data layer's own tests cover PostgreSQL.
type testServer struct {
	handler  http.Handler
	settings *settings.Service
	hub      *realtime.Hub
	// clock is the time the features see, which only moves when a test
	// moves it, so answers that depend on time are exact.
	clock *testClock
	// cardDatabase is the card database's folder.
	cardDatabase string
}

func newTestServer(t *testing.T, secureCookies bool) testServer {
	t.Helper()

	return newTestServerWith(t, secureCookies, testCardDatabase(t), []games.PlayableCard{{EngineID: "engine-bolshack", Name: "Bolshack Dragon", Set: "dm-01"}})
}

// newTestServerWith is newTestServer with another card database and engine
// cards.
func newTestServerWith(t *testing.T, secureCookies bool, cardDatabase string, playable []games.PlayableCard) testServer {
	t.Helper()

	db := storetest.Open(t, config.SQLite)
	sqlDB, err := db.DB()
	require.NoError(t, err)

	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	siteSettings, err := settings.NewService(ctx, db, logger, slices.Concat(auth.Settings(), decks.Settings())...)
	require.NoError(t, err)

	clock := &testClock{now: time.Now()}
	authService := auth.NewService(db, users.NewService(db, logger), siteSettings, auth.NewPasswordHasher(4), clock.Now, logger)

	source, err := cards.ReadSource(cardDatabase)
	require.NoError(t, err)
	_, err = cards.Sync(ctx, db, source, playable, logger)
	require.NoError(t, err)
	catalogue := cards.NewCatalogue(db, cardDatabase)
	hub := realtime.NewHub(realtime.Options{OfflineAfter: time.Minute, Logger: logger})
	t.Cleanup(func() { require.NoError(t, hub.Close(context.Background())) })

	return testServer{
		handler: NewHandler(Dependencies{
			DB:            sqlDB,
			Auth:          authService,
			Catalogue:     catalogue,
			Formats:       formats.NewService(db),
			Decks:         decks.NewService(db, catalogue, siteSettings, logger),
			Hub:           hub,
			Upgrader:      realtime.NewUpgrader([]string{testSiteOrigin}),
			Logger:        logger,
			SecureCookies: secureCookies,
		}),
		settings:     siteSettings,
		hub:          hub,
		clock:        clock,
		cardDatabase: cardDatabase,
	}
}

// testCardDatabase has a playable card, a card the engine cannot play yet,
// a reprint and a printing without a picture.
func testCardDatabase(t *testing.T) string {
	t.Helper()

	bolshack := cardstest.Printing("0079", "Bolshack Dragon", "DM-01", "69/110")
	bolshack["cost"] = 6
	bolshack["power"] = 6000
	bolshack["power_text"] = "6000+"
	bolshack["abilities"] = []map[string]any{{"text": "Double breaker", "reminder": "(This creature breaks 2 shields.)"}}

	solar := cardstest.Printing("0029", "Solar Ray", "DM-01", "29/110")
	solar["card_type"] = "Spell"
	solar["civilizations"] = []string{"Light"}
	solar["power"] = nil
	solar["power_text"] = nil
	solar["cost"] = 2
	solarReprint := cardstest.Printing("0695", "Solar Ray", "DM-10", "5/110")
	solarReprint["card_type"] = "Spell"

	return cardstest.Write(t,
		[]map[string]any{bolshack, solar, solarReprint},
		[]string{"0079", "0029"},
		[]string{"dm-01"},
	)
}

// testClock is a clock that stands still until a test moves it.
type testClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *testClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

// Advance moves the clock on.
func (c *testClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}
