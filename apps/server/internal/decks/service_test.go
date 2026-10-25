package decks_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/decks"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/store/storetest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/validation"
)

func TestCreatingAndReadingADeck(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		f := newFixture(t, db)

		created, err := f.service.Create(ctx, f.shobu, decks.Input{
			Name:        "  Fire Rush  ",
			CoverCardID: bolshack,
			Cards:       []decks.Entry{{CardID: bolshack, Count: 4}, {CardID: basic01, Count: 2}},
		})
		require.NoError(t, err)

		assert.NotEmpty(t, created.ID)
		assert.Equal(t, f.shobu, created.UserID)
		assert.Equal(t, "Fire Rush", created.Name, "the name is trimmed")
		assert.Equal(t, bolshack, created.CoverCardID)
		assert.Equal(t, []decks.Entry{{CardID: basic01, Count: 2}, {CardID: bolshack, Count: 4}}, created.Cards)
		assert.False(t, created.CreatedAt.IsZero())

		found, err := f.service.Get(ctx, f.shobu, created.ID)
		require.NoError(t, err)
		assert.Equal(t, created, found)
	})
}

func TestADeckMayBeSavedEmptyOrUnfinished(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		f := newFixture(t, db)

		empty, err := f.service.Create(ctx, f.shobu, decks.Input{Name: "Ideas"})
		require.NoError(t, err)
		assert.Empty(t, empty.Cards)
		assert.Empty(t, empty.CoverCardID)

		full, err := f.service.Create(ctx, f.shobu, decks.Input{Name: "Ready", Cards: legalDeck()})
		require.NoError(t, err)
		assert.Equal(t, 40, decks.Count(full.Cards))
	})
}

func TestDecksArePrivate(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		f := newFixture(t, db)

		deck, err := f.service.Create(ctx, f.shobu, decks.Input{Name: "Mine"})
		require.NoError(t, err)

		_, err = f.service.Get(ctx, f.kokujo, deck.ID)
		assert.ErrorIs(t, err, decks.ErrNotFound)
		_, err = f.service.Update(ctx, f.kokujo, deck.ID, decks.Input{Name: "Stolen"})
		assert.ErrorIs(t, err, decks.ErrNotFound)
		assert.ErrorIs(t, f.service.Delete(ctx, f.kokujo, deck.ID), decks.ErrNotFound)

		list, err := f.service.List(ctx, f.kokujo)
		require.NoError(t, err)
		assert.Empty(t, list)

		still, err := f.service.Get(ctx, f.shobu, deck.ID)
		require.NoError(t, err)
		assert.Equal(t, "Mine", still.Name, "the other player changed nothing")

		_, err = f.service.Get(ctx, f.shobu, "no-such-deck")
		assert.ErrorIs(t, err, decks.ErrNotFound)
	})
}

func TestListingDecksShowsTheLatestChangedFirst(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		f := newFixture(t, db)

		var ids []string
		for _, name := range []string{"First", "Second", "Third"} {
			deck, err := f.service.Create(ctx, f.shobu, decks.Input{Name: name, Cards: []decks.Entry{{CardID: basic01, Count: 1}}})
			require.NoError(t, err)
			ids = append(ids, deck.ID)
			// Times are kept to the microsecond; make each change later.
			time.Sleep(2 * time.Millisecond)
		}
		_, err := f.service.Update(ctx, f.shobu, ids[0], decks.Input{Name: "First, changed"})
		require.NoError(t, err)

		list, err := f.service.List(ctx, f.shobu)
		require.NoError(t, err)
		names := make([]string, 0, len(list))
		for _, deck := range list {
			names = append(names, deck.Name)
		}
		assert.Equal(t, []string{"First, changed", "Third", "Second"}, names)
		assert.Equal(t, []decks.Entry{{CardID: basic01, Count: 1}}, list[1].Cards, "with their cards")
	})
}

func TestUpdatingADeckReplacesIt(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		f := newFixture(t, db)

		deck, err := f.service.Create(ctx, f.shobu, decks.Input{Name: "Draft", CoverCardID: basic01, Cards: []decks.Entry{{CardID: basic01, Count: 4}, {CardID: basic02, Count: 4}}})
		require.NoError(t, err)

		time.Sleep(2 * time.Millisecond)
		updated, err := f.service.Update(ctx, f.shobu, deck.ID, decks.Input{Name: "Final", Cards: []decks.Entry{{CardID: basic02, Count: 1}, {CardID: hulcus, Count: 3}}})
		require.NoError(t, err)

		assert.Equal(t, "Final", updated.Name)
		assert.Empty(t, updated.CoverCardID, "the cover is cleared when not given")
		assert.Equal(t, []decks.Entry{{CardID: basic02, Count: 1}, {CardID: hulcus, Count: 3}}, updated.Cards)
		assert.Equal(t, deck.CreatedAt, updated.CreatedAt)
		assert.True(t, updated.UpdatedAt.After(deck.UpdatedAt))

		found, err := f.service.Get(ctx, f.shobu, deck.ID)
		require.NoError(t, err)
		assert.Equal(t, updated, found)
	})
}

func TestAFailedUpdateChangesNothing(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		f := newFixture(t, db)

		deck, err := f.service.Create(ctx, f.shobu, decks.Input{Name: "Keep", Cards: []decks.Entry{{CardID: basic01, Count: 4}}})
		require.NoError(t, err)

		_, err = f.service.Update(ctx, f.shobu, deck.ID, decks.Input{Name: "Broken", Cards: []decks.Entry{{CardID: future, Count: 1}}})
		require.Error(t, err)

		found, err := f.service.Get(ctx, f.shobu, deck.ID)
		require.NoError(t, err)
		assert.Equal(t, deck, found)
	})
}

func TestDeletingADeck(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		f := newFixture(t, db)

		deck, err := f.service.Create(ctx, f.shobu, decks.Input{Name: "Gone soon", Cards: legalDeck()})
		require.NoError(t, err)

		require.NoError(t, f.service.Delete(ctx, f.shobu, deck.ID))

		_, err = f.service.Get(ctx, f.shobu, deck.ID)
		assert.ErrorIs(t, err, decks.ErrNotFound)
		assert.ErrorIs(t, f.service.Delete(ctx, f.shobu, deck.ID), decks.ErrNotFound)

		var left int64
		require.NoError(t, db.Table("deck_cards").Where("deck_id = ?", deck.ID).Count(&left).Error)
		assert.Zero(t, left, "its cards go with it")
	})
}

func TestSavingADeckChecksTheRules(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		f := newFixture(t, db)
		existing, err := f.service.Create(ctx, f.shobu, decks.Input{Name: "Existing"})
		require.NoError(t, err)

		for name, test := range map[string]struct {
			input  decks.Input
			field  string
			reason string
		}{
			"no name":           {decks.Input{Name: "   "}, "name", "must not be empty"},
			"a long name":       {decks.Input{Name: strings.Repeat("é", decks.NameMaxLength+1)}, "name", "at most 40"},
			"a name not text":   {decks.Input{Name: "Deck \xff"}, "name", "valid text"},
			"a card twice":      {decks.Input{Name: "D", Cards: []decks.Entry{{CardID: basic01, Count: 1}, {CardID: basic01, Count: 2}}}, "cards", "each card once"},
			"too many cards":    {decks.Input{Name: "D", Cards: append(legalDeck(), decks.Entry{CardID: hulcus, Count: 1})}, "cards", "at most 40, not 41"},
			"an unknown card":   {decks.Input{Name: "D", Cards: []decks.Entry{{CardID: "nope", Count: 1}}}, "cards", `"nope", which is not a card`},
			"an unplayable one": {decks.Input{Name: "D", Cards: []decks.Entry{{CardID: future, Count: 1}}}, "cards", "Future Card, which cannot be played yet"},
			"no copies":         {decks.Input{Name: "D", Cards: []decks.Entry{{CardID: basic01, Count: 0}}}, "cards", "1 to 4 copies of Basic 01, not 0"},
			"five copies":       {decks.Input{Name: "D", Cards: []decks.Entry{{CardID: basic01, Count: 5}}}, "cards", "1 to 4 copies of Basic 01, not 5"},
			"a cover not in it": {decks.Input{Name: "D", CoverCardID: hulcus, Cards: []decks.Entry{{CardID: basic01, Count: 1}}}, "coverCardId", "one of the deck's cards"},
		} {
			_, err := f.service.Create(ctx, f.shobu, test.input)
			invalid, ok := validation.As(err)
			require.True(t, ok, "%s: got %v", name, err)
			assert.Equal(t, test.field, invalid.Field, name)
			assert.Contains(t, invalid.Reason, test.reason, name)

			_, err = f.service.Update(ctx, f.shobu, existing.ID, test.input)
			_, ok = validation.As(err)
			assert.True(t, ok, "%s: updating got %v", name, err)
		}

		_, err = f.service.Create(ctx, f.shobu, decks.Input{Name: strings.Repeat("é", decks.NameMaxLength)})
		assert.NoError(t, err, "a name of exactly the most characters")

		list, err := f.service.List(ctx, f.shobu)
		require.NoError(t, err)
		assert.Len(t, list, 2, "nothing invalid was saved")
	})
}

func TestAPlayerCanHaveOnlySoManyDecks(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		f := newFixture(t, db)
		_, err := f.settings.Set(ctx, decks.MaxPerPlayer.Key, "2")
		require.NoError(t, err)

		for range 2 {
			_, err := f.service.Create(ctx, f.shobu, decks.Input{Name: "Deck"})
			require.NoError(t, err)
		}

		_, err = f.service.Create(ctx, f.shobu, decks.Input{Name: "One too many"})
		var tooMany *decks.TooManyDecksError
		require.True(t, errors.As(err, &tooMany), "got %v", err)
		assert.Equal(t, int64(2), tooMany.Limit)

		_, err = f.service.Create(ctx, f.kokujo, decks.Input{Name: "Another player's"})
		assert.NoError(t, err, "the limit is per player")
	})
}

func TestTheDeckLimitHoldsForDecksCreatedAtOnce(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		f := newFixture(t, db)
		_, err := f.settings.Set(ctx, decks.MaxPerPlayer.Key, "3")
		require.NoError(t, err)

		var wg sync.WaitGroup
		var mu sync.Mutex
		created, refused := 0, 0
		for range 8 {
			wg.Go(func() {
				_, err := f.service.Create(ctx, f.shobu, decks.Input{Name: "Rush"})
				mu.Lock()
				defer mu.Unlock()
				var tooMany *decks.TooManyDecksError
				switch {
				case err == nil:
					created++
				case errors.As(err, &tooMany):
					refused++
				default:
					t.Errorf("creating a deck: %v", err)
				}
			})
		}
		wg.Wait()

		assert.Equal(t, 3, created)
		assert.Equal(t, 5, refused)
	})
}

func TestImportReadsADeckList(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		f := newFixture(t, db)

		imported, err := f.service.Import(ctx, strings.Join([]string{
			"4 Bolshack Dragon",
			"3x aqua hulcus",       // any case
			"4 X Revolver Dragon",  // a name starting with X
			"2X Revolver Dragon",   // an x touching the count
			"750 Man",              // a name starting with a number
			"1 Bolshack Dragon",    // a fifth copy
			"2 Unknown Thing",      // no such card
			"2 Future Card",        // cannot be played
			"0 Basic 01",           // no copies
			"# Sideboard thoughts", // a comment
		}, "\n"))
		require.NoError(t, err)

		assert.Equal(t, []decks.Entry{
			{CardID: bolshack, Count: 4},
			{CardID: hulcus, Count: 3},
			{CardID: xDragon, Count: 4},
			{CardID: revolver, Count: 2},
			{CardID: man750, Count: 1},
		}, imported.Cards)
		assert.Equal(t, []decks.LineProblem{
			{Number: 6, Text: "1 Bolshack Dragon", Reason: "A deck can have at most 4 copies of Bolshack Dragon; 1 left out."},
			{Number: 7, Text: "2 Unknown Thing", Reason: `No card is called "Unknown Thing".`},
			{Number: 8, Text: "2 Future Card", Reason: "Future Card cannot be played yet."},
			{Number: 9, Text: "0 Basic 01", Reason: "The count must be between 1 and 40."},
		}, imported.Problems)
	})
}

func TestImportOfNothing(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db)

		imported, err := f.service.Import(context.Background(), "\n# only a comment\n")
		require.NoError(t, err)
		assert.Empty(t, imported.Cards)
		assert.Empty(t, imported.Problems)
		assert.NotNil(t, imported.Cards, "an empty list, not a missing one")
	})
}

func TestImportRefusesTextBeyondTheLimits(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db)

		_, err := f.service.Import(context.Background(), strings.Repeat("a", decks.TextMaxLength+1))
		invalid, ok := validation.As(err)
		require.True(t, ok, "got %v", err)
		assert.Equal(t, "text", invalid.Field)
	})
}
