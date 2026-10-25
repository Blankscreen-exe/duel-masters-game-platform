package decks_test

// These tests seed the starter decks from the card database linked into the
// repository, whose cards they name.

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/cards"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/decks"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/formats"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/games"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/settings"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/store"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/store/storetest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/users"
)

// realCardDatabase is the card database linked into the repository.
const realCardDatabase = "../../../../resources/duelmasters_db"

// starterFixture is the decks service on the real card catalogue, with the
// starter decks seeded.
type starterFixture struct {
	db        *gorm.DB
	decks     *decks.Service
	catalogue *cards.Catalogue
	shobu     users.User
	kokujo    users.User
}

func newStarterFixture(t *testing.T, db *gorm.DB) starterFixture {
	t.Helper()
	ctx := context.Background()

	if _, err := os.Stat(realCardDatabase); err != nil {
		if os.Getenv("CI") != "" {
			t.Fatalf("the card database must be checked out on CI (git submodule update --init): %v", err)
		}
		t.Skip("the card database is not checked out; run git submodule update --init")
	}
	source, err := cards.ReadSource(realCardDatabase)
	require.NoError(t, err)
	_, err = cards.Sync(ctx, db, source, games.PlayableCards(), storetest.Logger())
	require.NoError(t, err)
	require.NoError(t, store.RunSeeds(db, decks.Seeds(storetest.Logger())...))

	siteSettings, err := settings.NewService(ctx, db, storetest.Logger(), decks.Settings()...)
	require.NoError(t, err)
	catalogue := cards.NewCatalogue(db, realCardDatabase)
	accounts := users.NewService(db, storetest.Logger())
	shobu, err := accounts.Create(ctx, "Shobu", "hash")
	require.NoError(t, err)
	kokujo, err := accounts.Create(ctx, "Kokujo", "hash")
	require.NoError(t, err)

	return starterFixture{
		db:        db,
		decks:     decks.NewService(db, catalogue, siteSettings, storetest.Logger()),
		catalogue: catalogue,
		shobu:     shobu,
		kokujo:    kokujo,
	}
}

func TestEveryCivilizationHasALegalStarterDeck(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		f := newStarterFixture(t, db)

		starters, err := f.decks.Starters(ctx)
		require.NoError(t, err)
		require.Len(t, starters, 5)
		assert.Equal(t, "Fire: Bolshack Rush", starters[0].Name, "Fire's is first, the default")

		allFormats, err := formats.NewService(db).List(ctx)
		require.NoError(t, err)
		require.Len(t, allFormats, 2)

		var civilizations []string
		for _, starter := range starters {
			assert.True(t, starter.Starter)
			assert.Empty(t, starter.UserID)
			assert.NotEmpty(t, starter.CoverCardID)
			assert.Equal(t, decks.Size, decks.Count(starter.Cards), starter.Name)

			ids := make([]string, 0, len(starter.Cards))
			for _, entry := range starter.Cards {
				ids = append(ids, entry.CardID)
			}
			known, err := f.catalogue.Cards(ctx, ids)
			require.NoError(t, err)

			shown := decks.Civilizations(starter.Cards, known)
			require.Len(t, shown, 1, "%s is of one civilization", starter.Name)
			civilizations = append(civilizations, shown[0])
			for _, format := range allFormats {
				assert.Empty(t, decks.Check(starter.Cards, known, format), "%s keeps to %s", starter.Name, format.Name)
			}
		}
		assert.Equal(t, []string{"Fire", "Water", "Nature", "Light", "Darkness"}, civilizations)
	})
}

func TestStarterDecksAreSeededOnce(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		f := newStarterFixture(t, db)

		require.NoError(t, store.RunSeeds(db, decks.Seeds(storetest.Logger())...))
		starters, err := f.decks.Starters(ctx)
		require.NoError(t, err)
		assert.Len(t, starters, 5, "seeding again adds nothing")

		// Starter decks an admin has removed stay removed.
		require.NoError(t, db.Table("starter_decks").Where("id IS NOT NULL").Delete(map[string]any{}).Error)
		require.NoError(t, store.RunSeeds(db, decks.Seeds(storetest.Logger())...))
		starters, err = f.decks.Starters(ctx)
		require.NoError(t, err)
		assert.Empty(t, starters)
	})
}

func TestAnyPlayerPlaysAStarterDeck(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		f := newStarterFixture(t, db)
		starters, err := f.decks.Starters(ctx)
		require.NoError(t, err)
		fire := starters[0]

		for _, player := range []users.User{f.shobu, f.kokujo} {
			deck, err := f.decks.Playable(ctx, player.ID, fire.ID)
			require.NoError(t, err)
			assert.Equal(t, fire, deck)
		}

		_, err = f.decks.Get(ctx, f.shobu.ID, fire.ID)
		assert.ErrorIs(t, err, decks.ErrNotFound, "a starter deck is nobody's own")

		own, err := f.decks.Create(ctx, f.shobu.ID, decks.Input{Name: "Mine", Cards: fire.Cards})
		require.NoError(t, err)
		deck, err := f.decks.Playable(ctx, f.shobu.ID, own.ID)
		require.NoError(t, err)
		assert.Equal(t, own.ID, deck.ID)
		_, err = f.decks.Playable(ctx, f.kokujo.ID, own.ID)
		assert.ErrorIs(t, err, decks.ErrNotFound, "another player's deck is not playable")

		require.NoError(t, db.Table("starter_decks").Where("id = ?", fire.ID).Update("enabled", false).Error)
		_, err = f.decks.Playable(ctx, f.kokujo.ID, fire.ID)
		assert.ErrorIs(t, err, decks.ErrNotFound, "a starter deck no longer offered is not playable")
		starters, err = f.decks.Starters(ctx)
		require.NoError(t, err)
		assert.Len(t, starters, 4)
	})
}

func TestAPlayerCopiesAStarterDeck(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		f := newStarterFixture(t, db)
		starters, err := f.decks.Starters(ctx)
		require.NoError(t, err)
		water := starters[1]

		copied, err := f.decks.CopyStarter(ctx, f.shobu.ID, water.ID)
		require.NoError(t, err)
		assert.NotEqual(t, water.ID, copied.ID)
		assert.Equal(t, f.shobu.ID, copied.UserID)
		assert.False(t, copied.Starter)
		assert.Equal(t, water.Name, copied.Name)
		assert.Equal(t, water.CoverCardID, copied.CoverCardID)
		assert.ElementsMatch(t, water.Cards, copied.Cards)

		own, err := f.decks.List(ctx, f.shobu.ID)
		require.NoError(t, err)
		assert.Len(t, own, 1)

		_, err = f.decks.CopyStarter(ctx, f.shobu.ID, copied.ID)
		assert.ErrorIs(t, err, decks.ErrNotFound, "only a starter deck is copied")
	})
}
