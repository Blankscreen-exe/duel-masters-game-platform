package cards

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/games"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/store/storetest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/validation"
)

// testCatalogue is a small card database with a little of everything.
func testCatalogue(t *testing.T) (dir string, playable []games.PlayableCard) {
	t.Helper()

	bolshack := printing("0079", "Bolshack Dragon", "DM-01", "69/110")
	bolshack["cost"] = 6
	bolshack["power"] = 6000
	bolshack["power_text"] = "6000+"
	bolshack["races"] = []string{"Armored Dragon"}
	bolshack["keywords"] = []string{"double_break"}
	bolshack["rules_text"] = "Double breaker"
	bolshack["abilities"] = []map[string]any{{"text": "Double breaker", "reminder": "(This creature breaks 2 shields.)"}}

	hulcus := printing("0033", "Aqua Hulcus", "DM-01", "33/110")
	hulcus["civilizations"] = []string{"Water"}
	hulcus["races"] = []string{"Liquid People"}
	hulcus["keywords"] = []string{}
	hulcus["rules_text"] = "When you put this creature into the battle zone, you may draw a card."

	solar := printing("0029", "Solar Ray", "DM-01", "29/110")
	solar["card_type"] = "Spell"
	solar["civilizations"] = []string{"Light"}
	solar["races"] = []string{}
	solar["power"] = nil
	solar["power_text"] = nil
	solar["cost"] = 2
	solar["keywords"] = []string{"shield_trigger"}
	solar["rules_text"] = "Shield trigger. Tap 1 of your opponent's creatures; 100% sure."
	solarReprint := printing("0695", "Solar Ray", "DM-10", "5/110")
	solarReprint["card_type"] = "Spell"
	solarReprint["civilizations"] = []string{"Light"}

	zaschack := printing("9026", "Überdragon Zaschack", "PROMO", "L6/Y2")
	zaschack["cost"] = 9
	zaschack["civilizations"] = []string{"Fire", "Darkness"}

	unplayable := printing("1500", "Future_Card", "DM-20", "1/1")
	unplayable["cost"] = 4

	dir = writeCardDatabase(t,
		[]map[string]any{bolshack, hulcus, solar, solarReprint, zaschack, unplayable},
		[]string{"0079", "0033", "0029", "0695", "1500"},
		[]string{"dm-01", "dm-10"},
	)

	return dir, []games.PlayableCard{
		{EngineID: "engine-bolshack", Name: "Bolshack Dragon", Set: "dm-01"},
		{EngineID: "engine-hulcus", Name: "Aqua Hulcus", Set: "dm-01"},
		{EngineID: "engine-solar", Name: "Solar Ray", Set: "dm-01"},
		{EngineID: "engine-zaschack", Name: "Uberdragon Zaschack", Set: "promo"},
	}
}

func syncTestCatalogue(t *testing.T, db *gorm.DB) (*Catalogue, string) {
	t.Helper()

	dir, playable := testCatalogue(t)
	source, err := ReadSource(dir)
	require.NoError(t, err)

	result, err := Sync(context.Background(), db, source, playable, storetest.Logger())
	require.NoError(t, err)
	require.True(t, result.Copied)

	return NewCatalogue(db, dir), dir
}

func names(cards []Card) []string {
	out := make([]string, 0, len(cards))
	for _, card := range cards {
		out = append(out, card.Name)
	}
	return out
}

func intPointer(n int) *int { return &n }

func TestSyncCopiesTheCardDatabase(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		dir, playable := testCatalogue(t)
		source, err := ReadSource(dir)
		require.NoError(t, err)

		result, err := Sync(ctx, db, source, playable, storetest.Logger())
		require.NoError(t, err)
		assert.True(t, result.Copied)
		assert.Equal(t, 5, result.Cards, "Solar Ray's two printings are one card")
		assert.Equal(t, 4, result.Playable)
		assert.Equal(t, []string{`printing 9026 of "Überdragon Zaschack" has no picture`}, result.Problems)

		catalogue := NewCatalogue(db, dir)
		bolshack, err := catalogue.Card(ctx, "0079")
		require.NoError(t, err)
		assert.Equal(t, "Bolshack Dragon", bolshack.Name)
		assert.Equal(t, "Creature", bolshack.Type)
		assert.Equal(t, 6, bolshack.Cost)
		require.NotNil(t, bolshack.Power)
		assert.Equal(t, 6000, *bolshack.Power)
		assert.Equal(t, "6000+", bolshack.PowerText)
		assert.Equal(t, []string{"Fire"}, bolshack.Civilizations)
		assert.Equal(t, []string{"Armored Dragon"}, bolshack.Races)
		assert.Equal(t, []string{"double_break"}, bolshack.Keywords)
		assert.Equal(t, []Ability{{Text: "Double breaker", Reminder: "(This creature breaks 2 shields.)"}}, bolshack.Abilities)
		assert.Equal(t, "engine-bolshack", bolshack.EngineID)
		assert.True(t, bolshack.Playable())

		solar, err := catalogue.Card(ctx, "0029")
		require.NoError(t, err)
		assert.Nil(t, solar.Power, "spells have no power")
		require.Len(t, solar.Printings, 2)
		assert.Equal(t, "DM-01", solar.Printings[0].SetCode)
		assert.Equal(t, "DM-10", solar.Printings[1].SetCode)

		future, err := catalogue.Card(ctx, "1500")
		require.NoError(t, err)
		assert.False(t, future.Playable())

		_, err = catalogue.Card(ctx, "0695")
		assert.ErrorIs(t, err, ErrNotFound, "a card is known by its first printing only")
	})
}

func TestSyncSkipsAnUnchangedCatalogue(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		dir, playable := testCatalogue(t)
		source, err := ReadSource(dir)
		require.NoError(t, err)

		_, err = Sync(ctx, db, source, playable, storetest.Logger())
		require.NoError(t, err)

		again, err := Sync(ctx, db, source, playable, storetest.Logger())
		require.NoError(t, err)
		assert.False(t, again.Copied, "nothing changed")

		moreEngine := append(append([]games.PlayableCard(nil), playable...), games.PlayableCard{EngineID: "engine-future", Name: "Future_Card"})
		changed, err := Sync(ctx, db, source, moreEngine, storetest.Logger())
		require.NoError(t, err)
		assert.True(t, changed.Copied, "the engine plays a new card")

		future, err := NewCatalogue(db, dir).Card(ctx, "1500")
		require.NoError(t, err)
		assert.Equal(t, "engine-future", future.EngineID)
	})
}

func TestSyncKeepsCardsThatLeaveTheCardDatabaseHidden(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		catalogue, _ := syncTestCatalogue(t, db)

		smaller, err := ReadSource(writeCardDatabase(t, []map[string]any{printing("0079", "Bolshack Dragon", "DM-01", "69/110")}, []string{"0079"}, nil))
		require.NoError(t, err)
		_, err = Sync(ctx, db, smaller, []games.PlayableCard{{EngineID: "engine-bolshack", Name: "Bolshack Dragon"}}, storetest.Logger())
		require.NoError(t, err)

		_, err = catalogue.Card(ctx, "0033")
		assert.ErrorIs(t, err, ErrNotFound, "a card gone from the card database is not shown")

		page, err := catalogue.Search(ctx, Filter{})
		require.NoError(t, err)
		assert.Equal(t, []string{"Bolshack Dragon"}, names(page.Cards))

		var kept int64
		require.NoError(t, db.Table("cards").Where("id = ?", "0033").Count(&kept).Error)
		assert.Equal(t, int64(1), kept, "its row stays, so nothing referring to it breaks")
	})
}

func TestSyncCanMoveAnEngineIDToAnotherCard(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		catalogue, _ := syncTestCatalogue(t, db)

		// The engine renames its card, so the same engine id now belongs to
		// another card in the card database.
		dir, playable := testCatalogue(t)
		source, err := ReadSource(dir)
		require.NoError(t, err)
		for i := range playable {
			if playable[i].EngineID == "engine-hulcus" {
				playable[i].Name = "Future_Card"
			}
		}
		_, err = Sync(ctx, db, source, playable, storetest.Logger())
		require.NoError(t, err)

		hulcus, err := catalogue.Card(ctx, "0033")
		require.NoError(t, err)
		assert.Empty(t, hulcus.EngineID)
		future, err := catalogue.Card(ctx, "1500")
		require.NoError(t, err)
		assert.Equal(t, "engine-hulcus", future.EngineID)
	})
}

func TestSearchFilters(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		catalogue, _ := syncTestCatalogue(t, db)

		for _, test := range []struct {
			name   string
			filter Filter
			want   []string
		}{
			{"everything, in name order", Filter{}, []string{"Aqua Hulcus", "Bolshack Dragon", "Future_Card", "Solar Ray", "Überdragon Zaschack"}},
			{"any of the civilizations", Filter{Civilizations: []string{"Water", "Darkness"}}, []string{"Aqua Hulcus", "Überdragon Zaschack"}},
			{"type", Filter{Types: []string{"Spell"}}, []string{"Solar Ray"}},
			{"race", Filter{Races: []string{"Armored Dragon"}}, []string{"Bolshack Dragon"}},
			{"keyword", Filter{Keywords: []string{"shield_trigger"}}, []string{"Solar Ray"}},
			{"a printing in the set", Filter{Sets: []string{"DM-10"}}, []string{"Solar Ray"}},
			{"cost range", Filter{MinCost: intPointer(3), MaxCost: intPointer(6)}, []string{"Aqua Hulcus", "Bolshack Dragon", "Future_Card"}},
			{"playable only", Filter{PlayableOnly: true}, []string{"Aqua Hulcus", "Bolshack Dragon", "Solar Ray", "Überdragon Zaschack"}},
			{"name, ignoring case", Filter{Query: "BOLSHACK"}, []string{"Bolshack Dragon"}},
			{"name, ignoring accents", Filter{Query: "uberdragon"}, []string{"Überdragon Zaschack"}},
			{"rules text", Filter{Query: "draw a card"}, []string{"Aqua Hulcus"}},
			{"% matches only itself", Filter{Query: "100%"}, []string{"Solar Ray"}},
			{"_ matches only itself", Filter{Query: "e_c"}, []string{"Future_Card"}},
			{"filters combine", Filter{Civilizations: []string{"Fire"}, PlayableOnly: true, MaxCost: intPointer(6)}, []string{"Bolshack Dragon"}},
			{"nothing matches", Filter{Query: "no such card"}, []string{}},
		} {
			page, err := catalogue.Search(ctx, test.filter)
			require.NoError(t, err, test.name)
			assert.Equal(t, test.want, names(page.Cards), test.name)
			assert.Equal(t, len(test.want), page.Total, test.name)
		}
	})
}

func TestSearchPages(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		catalogue, _ := syncTestCatalogue(t, db)

		first, err := catalogue.Search(ctx, Filter{PageSize: 2})
		require.NoError(t, err)
		assert.Equal(t, []string{"Aqua Hulcus", "Bolshack Dragon"}, names(first.Cards))
		assert.Equal(t, 5, first.Total)
		assert.Equal(t, 1, first.Page)
		assert.Equal(t, 2, first.PageSize)

		last, err := catalogue.Search(ctx, Filter{Page: 3, PageSize: 2})
		require.NoError(t, err)
		assert.Equal(t, []string{"Überdragon Zaschack"}, names(last.Cards))

		beyond, err := catalogue.Search(ctx, Filter{Page: 9, PageSize: 2})
		require.NoError(t, err)
		assert.Empty(t, beyond.Cards)
		assert.Equal(t, 5, beyond.Total)

		defaults, err := catalogue.Search(ctx, Filter{})
		require.NoError(t, err)
		assert.Equal(t, DefaultPageSize, defaults.PageSize)
	})
}

func TestSearchRefusesFiltersBeyondTheLimits(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		catalogue, _ := syncTestCatalogue(t, db)

		for _, test := range []struct {
			filter Filter
			field  string
		}{
			{Filter{Page: -1}, "page"},
			{Filter{PageSize: MaxPageSize + 1}, "pageSize"},
			{Filter{PageSize: -1}, "pageSize"},
			{Filter{Query: string(make([]rune, MaxQueryLength+1))}, "query"},
			{Filter{MinCost: intPointer(-1)}, "minCost"},
			{Filter{MaxCost: intPointer(-1)}, "maxCost"},
			{Filter{MinCost: intPointer(5), MaxCost: intPointer(4)}, "minCost"},
		} {
			_, err := catalogue.Search(context.Background(), test.filter)
			invalid, ok := validation.As(err)
			require.True(t, ok, "%+v: got %v", test.filter, err)
			assert.Equal(t, test.field, invalid.Field)
		}
	})
}

func TestFilterValues(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		catalogue, _ := syncTestCatalogue(t, db)

		values, err := catalogue.FilterValues(context.Background())
		require.NoError(t, err)

		assert.Equal(t, []string{"Darkness", "Fire", "Light", "Water"}, values.Civilizations)
		assert.Equal(t, []string{"Creature", "Spell"}, values.Types)
		assert.Equal(t, []string{"Armored Dragon", "Human", "Liquid People"}, values.Races)
		assert.Equal(t, []string{"blocker", "double_break", "shield_trigger"}, values.Keywords)
		assert.Equal(t, []Set{
			{Code: "DM-01", HasIcon: true},
			{Code: "DM-10", HasIcon: true},
			{Code: "DM-20"},
			{Code: "PROMO"},
		}, values.Sets)
	})
}

func TestImageFile(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		catalogue, dir := syncTestCatalogue(t, db)

		byPrinting, err := catalogue.ImageFile(ctx, "0695")
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(dir, "data", "images", "0695.webp"), byPrinting)

		byEngine, err := catalogue.ImageFile(ctx, "engine-solar")
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(dir, "data", "images", "0029.webp"), byEngine, "the engine id gives the first printing")
		content, err := os.ReadFile(byEngine)
		require.NoError(t, err)
		assert.Equal(t, "picture 0029", string(content))

		_, err = catalogue.ImageFile(ctx, "9026")
		assert.ErrorIs(t, err, ErrNotFound, "a printing without a picture")
		_, err = catalogue.ImageFile(ctx, "no-such-card")
		assert.ErrorIs(t, err, ErrNotFound)
	})
}

func TestSetIconFile(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		catalogue, dir := syncTestCatalogue(t, db)

		icon, err := catalogue.SetIconFile(ctx, "DM-01")
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(dir, "data", "icons", "sets", "dm-01.png"), icon)

		_, err = catalogue.SetIconFile(ctx, "DM-20")
		assert.ErrorIs(t, err, ErrNotFound, "a set without an icon")
		_, err = catalogue.SetIconFile(ctx, "DM-99")
		assert.ErrorIs(t, err, ErrNotFound)
	})
}

func TestSyncCopiesTheRealCardDatabase(t *testing.T) {
	if _, err := os.Stat(realCardDatabase); err != nil {
		if os.Getenv("CI") != "" {
			t.Fatalf("the card database must be checked out on CI: %v", err)
		}
		t.Skip("the card database is not checked out; run git submodule update --init")
	}

	source, err := ReadSource(realCardDatabase)
	require.NoError(t, err)
	playable := games.PlayableCards()

	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()

		started := time.Now()
		result, err := Sync(ctx, db, source, playable, storetest.Logger())
		require.NoError(t, err)
		t.Logf("copied %d cards in %s", result.Cards, time.Since(started).Round(time.Millisecond))

		assert.Empty(t, result.Problems)
		assert.Equal(t, len(playable), result.Playable)

		catalogue := NewCatalogue(db, realCardDatabase)
		page, err := catalogue.Search(ctx, Filter{Query: "bolshack dragon", PlayableOnly: true})
		require.NoError(t, err)
		require.NotEmpty(t, page.Cards)

		picture, err := catalogue.ImageFile(ctx, page.Cards[0].EngineID)
		require.NoError(t, err)
		assert.FileExists(t, picture)

		values, err := catalogue.FilterValues(ctx)
		require.NoError(t, err)
		for _, set := range values.Sets {
			if set.Code == "DM-01" || set.Code == "PROMO" {
				assert.True(t, set.HasIcon, "%s has its icon", set.Code)
			}
		}
		icon, err := catalogue.SetIconFile(ctx, "DM-01")
		require.NoError(t, err)
		assert.FileExists(t, icon)
	})
}
