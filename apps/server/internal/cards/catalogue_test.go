package cards

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/store/storetest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/validation"
)

// testCatalogue is a small card database with a little of everything.
func testCatalogue(t *testing.T) (dir string, playable []EngineCard) {
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
	hulcus["rarity"] = "Rare"
	hulcus["rarity_code"] = "r"

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
	solarReprint["rarity"] = "Super Rare"
	solarReprint["rarity_code"] = "sr"

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

	return dir, []EngineCard{
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

		moreEngine := append(append([]EngineCard(nil), playable...), EngineCard{EngineID: "engine-future", Name: "Future_Card"})
		changed, err := Sync(ctx, db, source, moreEngine, storetest.Logger())
		require.NoError(t, err)
		assert.True(t, changed.Copied, "the engine plays a new card")

		future, err := NewCatalogue(db, dir).Card(ctx, "1500")
		require.NoError(t, err)
		assert.Equal(t, "engine-future", future.EngineID)
	})
}

func TestCardsFindsManyCardsAtOnce(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		catalogue, _ := syncTestCatalogue(t, db)

		found, err := catalogue.Cards(ctx, []string{"0079", "0029", "0079", "no-such-card"})
		require.NoError(t, err)
		require.Len(t, found, 2)
		assert.Equal(t, "Bolshack Dragon", found["0079"].Name)
		assert.Len(t, found["0029"].Printings, 2, "with their details")

		none, err := catalogue.Cards(ctx, nil)
		require.NoError(t, err)
		assert.Empty(t, none)
	})
}

func TestByNamesMatchesNamesAsPlayersTypeThem(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		catalogue, _ := syncTestCatalogue(t, db)

		found, err := catalogue.ByNames(ctx, []string{
			"Bolshack Dragon",     // exactly
			"aqua hulcus",         // in another case
			"Uberdragon Zaschack", // without the accent
			"Solar-Ray",           // with punctuation
			"Bolshack",            // only part of a name
			"!!!",                 // nothing but punctuation
		})
		require.NoError(t, err)

		ids := map[string]string{}
		for name, card := range found {
			ids[name] = card.ID
		}
		assert.Equal(t, map[string]string{
			"Bolshack Dragon":     "0079",
			"aqua hulcus":         "0033",
			"Uberdragon Zaschack": "9026",
			"Solar-Ray":           "0029",
		}, ids)
	})
}

func TestCardsAndByNamesLeaveOutHiddenCards(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		catalogue, _ := syncTestCatalogue(t, db)

		smaller, err := ReadSource(writeCardDatabase(t, []map[string]any{printing("0079", "Bolshack Dragon", "DM-01", "69/110")}, []string{"0079"}, nil))
		require.NoError(t, err)
		_, err = Sync(ctx, db, smaller, []EngineCard{{EngineID: "engine-bolshack", Name: "Bolshack Dragon"}}, storetest.Logger())
		require.NoError(t, err)

		found, err := catalogue.Cards(ctx, []string{"0079", "0033"})
		require.NoError(t, err)
		assert.Len(t, found, 1)
		assert.Contains(t, found, "0079")

		named, err := catalogue.ByNames(ctx, []string{"Aqua Hulcus"})
		require.NoError(t, err)
		assert.Empty(t, named)
	})
}

func TestSyncKeepsCardsThatLeaveTheCardDatabaseHidden(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		catalogue, _ := syncTestCatalogue(t, db)

		smaller, err := ReadSource(writeCardDatabase(t, []map[string]any{printing("0079", "Bolshack Dragon", "DM-01", "69/110")}, []string{"0079"}, nil))
		require.NoError(t, err)
		_, err = Sync(ctx, db, smaller, []EngineCard{{EngineID: "engine-bolshack", Name: "Bolshack Dragon"}}, storetest.Logger())
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

			{"exactly one civilization", Filter{Civilizations: []string{"Fire"}, CivilizationMatch: CivilizationsExactly}, []string{"Bolshack Dragon", "Future_Card"}},
			{"exactly two civilizations", Filter{Civilizations: []string{"Fire", "Darkness"}, CivilizationMatch: CivilizationsExactly}, []string{"Überdragon Zaschack"}},
			{"exactly, naming one twice", Filter{Civilizations: []string{"Water", "Water"}, CivilizationMatch: CivilizationsExactly}, []string{"Aqua Hulcus"}},
			{"only these civilizations", Filter{Civilizations: []string{"Fire", "Darkness"}, CivilizationMatch: CivilizationsOnly}, []string{"Bolshack Dragon", "Future_Card", "Überdragon Zaschack"}},
			{"only, with a card of another too", Filter{Civilizations: []string{"Darkness"}, CivilizationMatch: CivilizationsOnly}, []string{}},
			{"how civilizations match means nothing without any", Filter{CivilizationMatch: CivilizationsExactly, Types: []string{"Spell"}}, []string{"Solar Ray"}},
			{"a single civilization", Filter{CivilizationCount: CivilizationCountSingle}, []string{"Aqua Hulcus", "Bolshack Dragon", "Future_Card", "Solar Ray"}},
			{"several civilizations", Filter{CivilizationCount: CivilizationCountMulti}, []string{"Überdragon Zaschack"}},
			{"several, with the one chosen", Filter{Civilizations: []string{"Fire"}, CivilizationCount: CivilizationCountMulti}, []string{"Überdragon Zaschack"}},

			{"at least some power, leaving out spells", Filter{MinPower: intPointer(6000)}, []string{"Bolshack Dragon"}},
			{"at most some power, leaving out spells", Filter{MaxPower: intPointer(2000)}, []string{"Aqua Hulcus", "Future_Card", "Überdragon Zaschack"}},
			{"a rarity", Filter{Rarities: []string{"r"}}, []string{"Aqua Hulcus"}},
			{"a rarity of any printing", Filter{Rarities: []string{"sr"}}, []string{"Solar Ray"}},

			{"text with a typing mistake", Filter{Text: "draw a crad"}, []string{"Aqua Hulcus"}},
			{"text in another order", Filter{Text: "card draw"}, []string{"Aqua Hulcus"}},
			{"text is not the name", Filter{Text: "Hulcus"}, []string{}},
			{"text and another filter", Filter{Text: "blocker", Civilizations: []string{"Darkness"}}, []string{"Überdragon Zaschack"}},
			{"text of nothing but punctuation", Filter{Text: "..."}, []string{}},

			{"a format's sets", Filter{Legal: &Legality{Sets: []string{"DM-10"}}}, []string{"Solar Ray"}},
			{"a format's most civilizations", Filter{Civilizations: []string{"Fire"}, Legal: &Legality{AllSets: true, MaxCivilizations: 1}}, []string{"Bolshack Dragon", "Future_Card"}},
			{"a format's most, with room", Filter{Civilizations: []string{"Fire"}, Legal: &Legality{AllSets: true, MaxCivilizations: 2}}, []string{"Bolshack Dragon", "Future_Card", "Überdragon Zaschack"}},
			{"a format's banned cards", Filter{Legal: &Legality{AllSets: true, Banned: []string{"0079", "0033"}}}, []string{"Future_Card", "Solar Ray", "Überdragon Zaschack"}},
			{"a format with neither", Filter{Legal: &Legality{AllSets: true}}, []string{"Aqua Hulcus", "Bolshack Dragon", "Future_Card", "Solar Ray", "Überdragon Zaschack"}},
			{"a format allowing no sets", Filter{Legal: &Legality{}}, []string{}},

			{"only these cards", Filter{OnlyIDs: []string{"0079", "0033"}}, []string{"Aqua Hulcus", "Bolshack Dragon"}},
			{"only a card there is not", Filter{OnlyIDs: []string{"none"}}, []string{}},
			{"not these cards", Filter{ExceptIDs: []string{"0079", "0033", "1500"}}, []string{"Solar Ray", "Überdragon Zaschack"}},
		} {
			page, err := catalogue.Search(ctx, test.filter)
			require.NoError(t, err, test.name)
			assert.Equal(t, test.want, names(page.Cards), test.name)
			assert.Equal(t, len(test.want), page.Total, test.name)
		}
	})
}

func TestSearchSorts(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		catalogue, _ := syncTestCatalogue(t, db)

		for _, test := range []struct {
			name   string
			filter Filter
			want   []string
		}{
			{"by name", Filter{Sort: SortByName}, []string{"Aqua Hulcus", "Bolshack Dragon", "Future_Card", "Solar Ray", "Überdragon Zaschack"}},
			{"by name, backwards", Filter{Sort: SortByName, Descending: true}, []string{"Überdragon Zaschack", "Solar Ray", "Future_Card", "Bolshack Dragon", "Aqua Hulcus"}},
			{"by cost", Filter{Sort: SortByCost}, []string{"Solar Ray", "Aqua Hulcus", "Future_Card", "Bolshack Dragon", "Überdragon Zaschack"}},
			{"by cost, backwards", Filter{Sort: SortByCost, Descending: true}, []string{"Überdragon Zaschack", "Bolshack Dragon", "Future_Card", "Aqua Hulcus", "Solar Ray"}},
			{"by power, ties by name and no power last", Filter{Sort: SortByPower}, []string{"Aqua Hulcus", "Future_Card", "Überdragon Zaschack", "Bolshack Dragon", "Solar Ray"}},
			{"by power, backwards, no power still last", Filter{Sort: SortByPower, Descending: true}, []string{"Bolshack Dragon", "Aqua Hulcus", "Future_Card", "Überdragon Zaschack", "Solar Ray"}},
		} {
			page, err := catalogue.Search(ctx, test.filter)
			require.NoError(t, err, test.name)
			assert.Equal(t, test.want, names(page.Cards), test.name)
		}
	})
}

func TestSearchPagesKeepTheSort(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		catalogue, _ := syncTestCatalogue(t, db)

		var seen []string
		for page := 1; page <= 3; page++ {
			found, err := catalogue.Search(ctx, Filter{Sort: SortByPower, Page: page, PageSize: 2})
			require.NoError(t, err)
			seen = append(seen, names(found.Cards)...)
		}

		assert.Equal(t, []string{"Aqua Hulcus", "Future_Card", "Überdragon Zaschack", "Bolshack Dragon", "Solar Ray"}, seen)
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
			{Filter{Text: string(make([]rune, MaxQueryLength+1))}, "text"},
			{Filter{MinPower: intPointer(-1)}, "minPower"},
			{Filter{MaxPower: intPointer(-1)}, "maxPower"},
			{Filter{MinPower: intPointer(5), MaxPower: intPointer(4)}, "minPower"},
			{Filter{OnlyIDs: make([]string, MaxListedCards+1)}, "cardId"},
			{Filter{ExceptIDs: make([]string, MaxListedCards+1)}, "exceptCardId"},
			{Filter{CivilizationMatch: "some"}, "civilizationMatch"},
			{Filter{CivilizationCount: "two"}, "civilizationCount"},
			{Filter{Sort: "colour"}, "sort"},
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
		assert.Equal(t, []Rarity{
			{Code: "c", Name: "Common"},
			{Code: "r", Name: "Rare"},
			{Code: "sr", Name: "Super Rare"},
		}, values.Rarities)
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
