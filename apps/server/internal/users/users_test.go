package users_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/store/storetest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/users"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/validation"
)

func newService(db *gorm.DB) *users.Service {
	return users.NewService(db, storetest.Logger())
}

func TestNormalizeUsername(t *testing.T) {
	for raw, want := range map[string][2]string{
		"Shobu":                {"Shobu", "shobu"},
		"  Kokujo  ":           {"Kokujo", "kokujo"},
		"blank_screen":         {"blank_screen", "blank_screen"},
		"Hakuoh-99":            {"Hakuoh-99", "hakuoh-99"},
		"abc":                  {"abc", "abc"},
		"A2345678901234567890": {"A2345678901234567890", "a2345678901234567890"},
	} {
		name, key, err := users.NormalizeUsername(raw)
		require.NoError(t, err, "%q", raw)
		assert.Equal(t, want[0], name, "%q", raw)
		assert.Equal(t, want[1], key, "%q", raw)
	}

	for raw, reason := range map[string]string{
		"ab":                    "at least 3",
		"   ":                   "at least 3",
		"A23456789012345678901": "at most 20",
		"two words":             "letters, numbers",
		"émile":                 "letters, numbers",
		"name!":                 "letters, numbers",
	} {
		_, _, err := users.NormalizeUsername(raw)
		invalid, ok := validation.As(err)
		require.True(t, ok, "%q: got %v", raw, err)
		assert.Equal(t, "username", invalid.Field)
		assert.Contains(t, invalid.Reason, reason, "%q", raw)
	}
}

func TestTheFirstAccountIsTheAdmin(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		service := newService(db)

		exist, err := service.AnyExist(ctx)
		require.NoError(t, err)
		assert.False(t, exist)

		first, err := service.Create(ctx, "Shobu", "hash-1")
		require.NoError(t, err)
		second, err := service.Create(ctx, "Kokujo", "hash-2")
		require.NoError(t, err)

		assert.Equal(t, users.RoleAdmin, first.Role)
		assert.True(t, first.IsAdmin())
		assert.Equal(t, users.RolePlayer, second.Role)
		assert.False(t, second.IsAdmin())

		exist, err = service.AnyExist(ctx)
		require.NoError(t, err)
		assert.True(t, exist)
	})
}

func TestOnlyOneAccountBecomesAdminWhenSeveralAreCreatedAtOnce(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		service := newService(db)

		var wg sync.WaitGroup
		created := make(chan users.User, 8)
		for i := range 8 {
			wg.Go(func() {
				user, err := service.Create(ctx, "player-"+string(rune('a'+i)), "hash")
				if assert.NoError(t, err) {
					created <- user
				}
			})
		}
		wg.Wait()
		close(created)

		admins := 0
		for user := range created {
			if user.IsAdmin() {
				admins++
			}
		}
		assert.Equal(t, 1, admins)
	})
}

func TestCreateStoresTheAccount(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		service := newService(db)

		created, err := service.Create(ctx, "  Shobu ", "the-hash")
		require.NoError(t, err)
		assert.Equal(t, "Shobu", created.Username, "surrounding spaces are trimmed")
		assert.Len(t, created.ID, 36)

		byName, err := service.ByUsername(ctx, "SHOBU")
		require.NoError(t, err)
		assert.Equal(t, created.ID, byName.ID, "names are found in any case")
		assert.Equal(t, "the-hash", byName.PasswordHash)
		assert.False(t, byName.CreatedAt.IsZero())

		byID, err := service.ByID(ctx, created.ID)
		require.NoError(t, err)
		assert.Equal(t, "Shobu", byID.Username)
	})
}

func TestCreateRefusesATakenNameInAnyCase(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		service := newService(db)

		_, err := service.Create(ctx, "Shobu", "hash")
		require.NoError(t, err)

		_, err = service.Create(ctx, "sHoBu", "hash")
		assert.ErrorIs(t, err, users.ErrUsernameTaken)
	})
}

func TestCreateRefusesAnInvalidName(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		_, err := newService(db).Create(context.Background(), "no spaces allowed", "hash")

		_, ok := validation.As(err)
		assert.True(t, ok, "got %v", err)
	})
}

func TestLookupsReportMissingAccounts(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		service := newService(db)

		_, err := service.ByUsername(ctx, "nobody")
		assert.ErrorIs(t, err, users.ErrNotFound)

		_, err = service.ByID(ctx, "00000000-0000-0000-0000-000000000000")
		assert.ErrorIs(t, err, users.ErrNotFound)

		err = service.SetPasswordHash(ctx, "00000000-0000-0000-0000-000000000000", "hash")
		assert.True(t, errors.Is(err, users.ErrNotFound))
	})
}

func TestSetPasswordHashReplacesTheHash(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		service := newService(db)

		user, err := service.Create(ctx, "Shobu", "old-hash")
		require.NoError(t, err)

		require.NoError(t, service.SetPasswordHash(ctx, user.ID, "new-hash"))

		reloaded, err := service.ByID(ctx, user.ID)
		require.NoError(t, err)
		assert.Equal(t, "new-hash", reloaded.PasswordHash)
	})
}
