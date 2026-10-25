package decks_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/cards"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/cards/cardstest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/decks"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/games"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/settings"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/store/storetest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/users"
)

// Card ids in the test card database.
const (
	// basic01 to basic10 are ten playable DM-01 cards costing 1 to 10.
	basic01 = "0001"
	basic02 = "0002"
	basic03 = "0003"
	basic10 = "0010"

	bolshack = "0079" // Bolshack Dragon, Fire, cost 6, DM-01
	hulcus   = "0033" // Aqua Hulcus, Water, cost 3, DM-01
	reprint  = "0029" // Solar Ray, Light: DM-01, reprinted in PROMO
	promo    = "9001" // Promo Hero, Nature: PROMO only
	late     = "1300" // Late Arrival, Darkness: DM-13 only
	future   = "1500" // Future Card: the engine cannot play it
	man750   = "0750" // 750 Man, whose name starts with a number
	xDragon  = "0751" // X Revolver Dragon, whose name starts with an X
	revolver = "0752" // Revolver Dragon
)

// basics are the ten basic card ids, cost 1 to 10.
func basics() []string {
	ids := make([]string, 0, 10)
	for i := 1; i <= 10; i++ {
		ids = append(ids, fmt.Sprintf("%04d", i))
	}
	return ids
}

// legalDeck is 4 copies of each basic card: 40 cards, legal in every format.
func legalDeck() []decks.Entry {
	entries := make([]decks.Entry, 0, 10)
	for _, id := range basics() {
		entries = append(entries, decks.Entry{CardID: id, Count: 4})
	}
	return entries
}

// testCardDatabase lays out the test card database and the engine cards
// linked to it.
func testCardDatabase(t *testing.T) (string, []games.PlayableCard) {
	t.Helper()

	var printings []map[string]any
	var playable []games.PlayableCard
	add := func(id, name, set string, cost int, civilization string, canPlay bool) map[string]any {
		printing := cardstest.Printing(id, name, set, id+"/110")
		printing["cost"] = cost
		printing["civilizations"] = []string{civilization}
		printings = append(printings, printing)
		if canPlay {
			playable = append(playable, games.PlayableCard{EngineID: "engine-" + id, Name: name, Set: set})
		}
		return printing
	}

	for i, id := range basics() {
		add(id, fmt.Sprintf("Basic %02d", i+1), "DM-01", i+1, "Fire", true)
	}
	add(bolshack, "Bolshack Dragon", "DM-01", 6, "Fire", true)
	add(hulcus, "Aqua Hulcus", "DM-01", 3, "Water", true)
	add(reprint, "Solar Ray", "DM-01", 2, "Light", true)
	printings = append(printings, cardstest.Printing("9002", "Solar Ray", "PROMO", "P2/Y1"))
	add(promo, "Promo Hero", "PROMO", 5, "Nature", true)
	add(late, "Late Arrival", "DM-13", 4, "Darkness", true)
	add(future, "Future Card", "DM-20", 4, "Fire", false)
	add(man750, "750 Man", "DM-01", 7, "Darkness", true)
	add(xDragon, "X Revolver Dragon", "DM-01", 8, "Fire", true)
	add(revolver, "Revolver Dragon", "DM-01", 7, "Fire", true)

	pictures := make([]string, 0, len(printings))
	for _, printing := range printings {
		pictures = append(pictures, printing["id"].(string))
	}

	return cardstest.Write(t, printings, pictures, nil), playable
}

// testCatalogue copies the test card database into db.
func testCatalogue(t *testing.T, db *gorm.DB) *cards.Catalogue {
	t.Helper()

	dir, playable := testCardDatabase(t)
	source, err := cards.ReadSource(dir)
	require.NoError(t, err)
	_, err = cards.Sync(context.Background(), db, source, playable, storetest.Logger())
	require.NoError(t, err)

	return cards.NewCatalogue(db, dir)
}

// known reads the test card database's cards, by id, without a database.
func known(t *testing.T) map[string]cards.Card {
	t.Helper()

	dir, playable := testCardDatabase(t)
	source, err := cards.ReadSource(dir)
	require.NoError(t, err)

	built, _ := cards.Build(source, playable)
	byID := make(map[string]cards.Card, len(built))
	for _, card := range built {
		byID[card.ID] = card
	}

	return byID
}

// fixture is a decks service on a fresh database, with two players.
type fixture struct {
	service  *decks.Service
	settings *settings.Service
	shobu    string
	kokujo   string
}

func newFixture(t *testing.T, db *gorm.DB) fixture {
	t.Helper()

	ctx := context.Background()
	siteSettings, err := settings.NewService(ctx, db, storetest.Logger(), decks.Settings()...)
	require.NoError(t, err)

	accounts := users.NewService(db, storetest.Logger())
	shobu, err := accounts.Create(ctx, "Shobu", "hash")
	require.NoError(t, err)
	kokujo, err := accounts.Create(ctx, "Kokujo", "hash")
	require.NoError(t, err)

	return fixture{
		service:  decks.NewService(db, testCatalogue(t, db), siteSettings, storetest.Logger()),
		settings: siteSettings,
		shobu:    shobu.ID,
		kokujo:   kokujo.ID,
	}
}
