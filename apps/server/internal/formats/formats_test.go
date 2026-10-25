package formats_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/formats"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/store/storetest"
)

func TestANewSiteHasTheStartingFormats(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		list, err := formats.NewService(db).List(context.Background())
		require.NoError(t, err)
		require.Len(t, list, 2)

		open, classic := list[0], list[1]
		assert.Equal(t, "Open", open.Name)
		assert.True(t, open.AllSets)
		assert.Empty(t, open.Sets)
		assert.True(t, open.AllowsSet("DM-35"))
		assert.True(t, open.AllowsSet("PROMO"))

		assert.Equal(t, "DM-01 to DM-12", classic.Name)
		assert.False(t, classic.AllSets)
		assert.Equal(t, []string{"DM-01", "DM-02", "DM-03", "DM-04", "DM-05", "DM-06", "DM-07", "DM-08", "DM-09", "DM-10", "DM-11", "DM-12"}, classic.Sets)
		assert.True(t, classic.AllowsSet("DM-12"))
		assert.False(t, classic.AllowsSet("DM-13"))
		assert.False(t, classic.AllowsSet("PROMO"))

		for _, format := range list {
			assert.NotEmpty(t, format.ID)
			assert.NotEmpty(t, format.Description)
			assert.Empty(t, format.Limits)
		}
	})
}

func TestFindingAFormat(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		service := formats.NewService(db)
		list, err := service.List(ctx)
		require.NoError(t, err)

		found, err := service.ByID(ctx, list[1].ID)
		require.NoError(t, err)
		assert.Equal(t, list[1], found)

		_, err = service.ByID(ctx, "no-such-format")
		assert.True(t, errors.Is(err, formats.ErrNotFound), "got %v", err)
	})
}

func TestFormatLimits(t *testing.T) {
	format := formats.Format{Limits: map[string]int{"banned": 0, "limited": 1}}

	limit, limited := format.Limit("banned")
	assert.Equal(t, 0, limit)
	assert.True(t, limited)

	limit, limited = format.Limit("limited")
	assert.Equal(t, 1, limit)
	assert.True(t, limited)

	_, limited = format.Limit("free")
	assert.False(t, limited)
}
