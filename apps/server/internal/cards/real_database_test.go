package cards_test

// These tests read the card database linked into the repository, with every
// card the engine implements. They are outside package cards so they can ask
// package games for the engine's cards, which itself uses package cards.

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/cards"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/cards/cardstest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/games"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/store/storetest"
)

// readRealCardDatabase reads the card database linked into the repository.
func readRealCardDatabase(t *testing.T) cards.Source {
	t.Helper()

	source, err := cards.ReadSource(cardstest.RealDatabase(t))
	require.NoError(t, err)
	return source
}

func TestTheRealCardDatabaseHasEveryEngineCard(t *testing.T) {
	source := readRealCardDatabase(t)

	playable := games.PlayableCards()
	built, problems := cards.Build(source, playable)
	assert.Empty(t, problems)

	linked := 0
	for _, card := range built {
		if card.Playable() {
			linked++
			require.NotEmpty(t, card.Printings[0].ImagePath, "%s has a picture", card.Name)
		}
	}
	assert.Equal(t, len(playable), linked, "every card the engine plays is in the catalogue")
}

func TestSyncCopiesTheRealCardDatabase(t *testing.T) {
	source := readRealCardDatabase(t)
	playable := games.PlayableCards()

	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()

		started := time.Now()
		result, err := cards.Sync(ctx, db, source, playable, storetest.Logger())
		require.NoError(t, err)
		t.Logf("copied %d cards in %s", result.Cards, time.Since(started).Round(time.Millisecond))

		assert.Empty(t, result.Problems)
		assert.Equal(t, len(playable), result.Playable)

		catalogue := cards.NewCatalogue(db, cardstest.RealDatabase(t))
		page, err := catalogue.Search(ctx, cards.Filter{Query: "bolshack dragon", PlayableOnly: true})
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
