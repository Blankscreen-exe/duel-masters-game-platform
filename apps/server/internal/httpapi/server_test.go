package httpapi

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/auth"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/cards"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/cards/cardstest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/chat"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/clocktest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/config"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/decks"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/formats"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/games"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/games/gamestest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/lobby"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/matchmaking"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/ratings"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/realtime"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/rematch"
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
	games    *games.Service
	// clock is the time the features see, which only moves when a test
	// moves it, so answers that depend on time are exact.
	clock *clocktest.Clock
	// cardDatabase is the card database's folder, when the test made one.
	cardDatabase string
}

// catalogueMaker copies a card database into a test's database and returns
// its catalogue and folder.
type catalogueMaker func(t *testing.T, db *gorm.DB) (*cards.Catalogue, string)

func newTestServer(t *testing.T, secureCookies bool) testServer {
	t.Helper()

	return newTestServerWith(t, secureCookies, syncedCatalogue(testCardDatabase(t), []cards.EngineCard{{EngineID: "engine-bolshack", Name: "Bolshack Dragon", Set: "dm-01"}}))
}

// syncedCatalogue copies a card database folder with the given engine cards.
func syncedCatalogue(dir string, playable []cards.EngineCard) catalogueMaker {
	return func(t *testing.T, db *gorm.DB) (*cards.Catalogue, string) {
		t.Helper()

		source, err := cards.ReadSource(dir)
		require.NoError(t, err)
		_, err = cards.Sync(context.Background(), db, source, playable, storetest.Logger())
		require.NoError(t, err)
		return cards.NewCatalogue(db, dir), dir
	}
}

// newTestServerWith is newTestServer with another card database.
func newTestServerWith(t *testing.T, secureCookies bool, makeCatalogue catalogueMaker) testServer {
	t.Helper()

	db := storetest.Open(t, config.SQLite)
	sqlDB, err := db.DB()
	require.NoError(t, err)

	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	siteSettings, err := settings.NewService(ctx, db, logger, slices.Concat(auth.Settings(), decks.Settings(), games.Settings(), lobby.Settings(), ratings.Settings(), rematch.Settings())...)
	require.NoError(t, err)

	clock := clocktest.New(time.Now())
	authService := auth.NewService(db, users.NewService(db, logger), siteSettings, auth.NewPasswordHasher(4), clock.Now, logger)

	catalogue, cardDatabase := makeCatalogue(t, db)
	deckService := decks.NewService(db, catalogue, siteSettings, logger)
	formatService := formats.NewService(db)

	hub := realtime.NewHub(realtime.Options{OfflineAfter: time.Minute, Logger: logger})
	t.Cleanup(func() { require.NoError(t, hub.Close(context.Background())) })
	live := NewLiveUpdates(hub)

	ratingService := ratings.NewService(db, siteSettings)
	gameService := games.NewService(db, catalogue, siteSettings, ratingService, logger)
	gamestest.Run(t, gameService)
	lobbyService := lobby.NewService(lobby.Dependencies{
		Decks: deckService, Formats: formatService, Games: gameService, Directory: live,
		Settings: siteSettings, Listener: live, Logger: logger, Now: clock.Now,
	})
	quickMatch := matchmaking.NewService(matchmaking.Dependencies{
		Lobby: lobbyService, Games: gameService, Directory: live, Listener: live, Logger: logger, Now: clock.Now,
	})
	rematches := rematch.NewService(rematch.Dependencies{
		Lobby: lobbyService, Games: gameService, Directory: live, Settings: siteSettings, Listener: live, Logger: logger, Now: clock.Now,
	})

	return testServer{
		handler: NewHandler(Dependencies{
			DB:            sqlDB,
			Auth:          authService,
			Catalogue:     catalogue,
			Formats:       formatService,
			Decks:         deckService,
			Chat:          chat.NewService(db, live, clock.Now),
			Games:         gameService,
			Lobby:         lobbyService,
			QuickMatch:    quickMatch,
			Rematches:     rematches,
			Hub:           hub,
			Upgrader:      realtime.NewUpgrader([]string{testSiteOrigin}),
			Logger:        logger,
			SecureCookies: secureCookies,
		}),
		settings:     siteSettings,
		hub:          hub,
		games:        gameService,
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
