package adminlog_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/adminlog"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/paging"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/store/storetest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/users"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/validation"
)

func TestTheLogListsActionsTheLatestFirst(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		boss, err := users.NewService(db, storetest.Logger()).Create(ctx, "Boss", "hash")
		require.NoError(t, err)
		log := adminlog.NewService(db)

		require.NoError(t, adminlog.RecordIn(db, boss, adminlog.UserBanned, "Shobu", "For 7 days: Spam"))
		require.NoError(t, log.Record(ctx, boss, adminlog.GameStopped, "Shobu vs Kokujo", ""))
		require.NoError(t, adminlog.RecordIn(db, boss, adminlog.UserUnbanned, "Shobu", ""))

		page, err := log.List(ctx, adminlog.Query{})
		require.NoError(t, err)
		assert.Equal(t, 3, page.Total)
		assert.Equal(t, adminlog.DefaultPageSize, page.PageSize)
		require.Len(t, page.Entries, 3)
		assert.Equal(t, []adminlog.Action{adminlog.UserUnbanned, adminlog.GameStopped, adminlog.UserBanned},
			[]adminlog.Action{page.Entries[0].Action, page.Entries[1].Action, page.Entries[2].Action},
			"the latest first, even when made in the same moment")

		banned := page.Entries[2]
		assert.Equal(t, boss.ID, banned.AdminID)
		assert.Equal(t, "Boss", banned.AdminName)
		assert.Equal(t, "Shobu", banned.Subject)
		assert.Equal(t, "For 7 days: Spam", banned.Details)
		assert.False(t, banned.At.IsZero())

		bans, err := log.List(ctx, adminlog.Query{Action: adminlog.UserBanned})
		require.NoError(t, err)
		assert.Equal(t, 1, bans.Total)

		second, err := log.List(ctx, adminlog.Query{Request: paging.Request{Page: 2, PageSize: 2}})
		require.NoError(t, err)
		require.Len(t, second.Entries, 1)
		assert.Equal(t, adminlog.UserBanned, second.Entries[0].Action)

		for _, query := range []adminlog.Query{
			{Action: "user_teleported"},
			{Request: paging.Request{PageSize: adminlog.MaxPageSize + 1}},
		} {
			_, err := log.List(ctx, query)
			_, ok := validation.As(err)
			assert.True(t, ok, "%+v: got %v", query, err)
		}
	})
}
