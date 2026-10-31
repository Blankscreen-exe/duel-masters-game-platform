package users_test

import (
	"context"
	"errors"
	"maps"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/avatars"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/paging"
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

func TestAnAccountChoosesAPicture(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		service := newService(db)
		user, err := service.Create(ctx, "Shobu", "hash")
		require.NoError(t, err)
		assert.Empty(t, user.AvatarID, "none until they choose")

		picture := avatars.IDs()[0]
		require.NoError(t, service.SetAvatar(ctx, user.ID, picture))
		found, err := service.ByID(ctx, user.ID)
		require.NoError(t, err)
		assert.Equal(t, picture, found.AvatarID)

		for _, id := range []string{"", "nothing"} {
			invalid, ok := validation.As(service.SetAvatar(ctx, user.ID, id))
			require.True(t, ok, id)
			assert.Equal(t, "avatarId", invalid.Field)
		}
		found, err = service.ByID(ctx, user.ID)
		require.NoError(t, err)
		assert.Equal(t, picture, found.AvatarID, "a refused choice changes nothing")

		assert.ErrorIs(t, service.SetAvatar(ctx, "nobody", picture), users.ErrNotFound)
	})
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

func TestBansAndMutesLastUntilTheyEnd(t *testing.T) {
	now := time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)
	later := now.Add(time.Hour)

	never := users.Moderation{}
	_, banned := never.BanAt(now)
	assert.False(t, banned)
	_, muted := never.MuteAt(now)
	assert.False(t, muted)

	timed := users.Moderation{BannedAt: &now, BannedUntil: &later, BanReason: "Spam", MutedAt: &now, MutedUntil: &later}
	ban, banned := timed.BanAt(now.Add(59 * time.Minute))
	require.True(t, banned)
	assert.Equal(t, users.Ban{Penalty: users.Penalty{Since: now, Until: &later}, Reason: "Spam"}, ban)
	_, banned = timed.BanAt(later)
	assert.False(t, banned, "a ban is over when it reaches its end")
	_, muted = timed.MuteAt(later)
	assert.False(t, muted)

	forGood := users.Moderation{BannedAt: &now, MutedAt: &now}
	_, banned = forGood.BanAt(now.Add(100 * 365 * 24 * time.Hour))
	assert.True(t, banned)
	mute, muted := forGood.MuteAt(now.Add(100 * 365 * 24 * time.Hour))
	assert.True(t, muted)
	assert.Nil(t, mute.Until)
}

func TestSearchFindsAccountsByNameAndStatus(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		service := newService(db)
		now := db.NowFunc()
		ended := now.Add(-time.Minute)
		lasting := now.Add(time.Hour)

		created := map[string]users.User{}
		for _, name := range []string{"Boss", "Shobu", "Kokujo", "Hakuoh", "Mimi_100", "Zed1"} {
			user, err := service.Create(ctx, name, "hash")
			require.NoError(t, err)
			created[name] = user
		}
		require.NoError(t, users.SetBanIn(db, created["Shobu"].ID, users.Ban{Penalty: users.Penalty{Since: now, Until: &lasting}, Reason: "Spam"}))
		require.NoError(t, users.SetBanIn(db, created["Kokujo"].ID, users.Ban{Penalty: users.Penalty{Since: now.Add(-time.Hour), Until: &ended}, Reason: "Old"}))
		require.NoError(t, users.SetMuteIn(db, created["Hakuoh"].ID, users.Penalty{Since: now}))

		names := func(query users.Query) []string {
			t.Helper()
			page, err := service.Search(ctx, query, now)
			require.NoError(t, err)
			found := make([]string, 0, len(page.Users))
			for _, user := range page.Users {
				found = append(found, user.Username)
			}
			return found
		}

		assert.Equal(t, []string{"Boss", "Hakuoh", "Kokujo", "Mimi_100", "Shobu", "Zed1"}, names(users.Query{}), "by name")
		assert.Equal(t, []string{"Shobu"}, names(users.Query{Status: users.StatusBanned}), "a ban that is over does not count")
		assert.Equal(t, []string{"Hakuoh"}, names(users.Query{Status: users.StatusMuted}))
		assert.Equal(t, []string{"Boss"}, names(users.Query{Status: users.StatusAdmin}))
		assert.Equal(t, []string{"Hakuoh", "Kokujo"}, names(users.Query{Search: "KU"}), "ignoring case")
		assert.Equal(t, []string{"Mimi_100"}, names(users.Query{Search: "_1"}), "wildcards match themselves")

		page, err := service.Search(ctx, users.Query{Request: paging.Request{Page: 2, PageSize: 2}}, now)
		require.NoError(t, err)
		assert.Equal(t, 6, page.Total)
		assert.Equal(t, 2, page.Page)
		require.Len(t, page.Users, 2)
		assert.Equal(t, "Kokujo", page.Users[0].Username)

		for _, query := range []users.Query{
			{Status: "sleeping"},
			{Search: strings.Repeat("a", 21)},
			{Request: paging.Request{PageSize: users.MaxPageSize + 1}},
		} {
			_, err := service.Search(ctx, query, now)
			_, ok := validation.As(err)
			assert.True(t, ok, "%+v: got %v", query, err)
		}
	})
}

func TestBansAndMutesAreStoredAndLifted(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		service := newService(db)
		now := db.NowFunc()
		user, err := service.Create(ctx, "Shobu", "hash")
		require.NoError(t, err)

		require.NoError(t, users.SetBanIn(db, user.ID, users.Ban{Penalty: users.Penalty{Since: now}, Reason: "Cheating"}))
		require.NoError(t, users.SetMuteIn(db, user.ID, users.Penalty{Since: now}))
		reloaded, err := service.ByID(ctx, user.ID)
		require.NoError(t, err)
		ban, banned := reloaded.BanAt(now)
		require.True(t, banned)
		assert.Equal(t, "Cheating", ban.Reason)
		_, muted := reloaded.MuteAt(now)
		assert.True(t, muted)

		require.NoError(t, users.ClearBanIn(db, user.ID))
		require.NoError(t, users.ClearMuteIn(db, user.ID))
		reloaded, err = service.ByID(ctx, user.ID)
		require.NoError(t, err)
		assert.Equal(t, users.Moderation{}, reloaded.Moderation)

		missing := "00000000-0000-0000-0000-000000000000"
		assert.ErrorIs(t, users.SetBanIn(db, missing, users.Ban{}), users.ErrNotFound)
		assert.ErrorIs(t, users.SetMuteIn(db, missing, users.Penalty{}), users.ErrNotFound)
		assert.ErrorIs(t, users.SetRoleIn(db, missing, users.RoleAdmin), users.ErrNotFound)
	})
}

func TestLockInReadsTheAccountsAndTheAdmins(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		service := newService(db)
		boss, err := service.Create(ctx, "Boss", "hash")
		require.NoError(t, err)
		shobu, err := service.Create(ctx, "Shobu", "hash")
		require.NoError(t, err)
		_, err = service.Create(ctx, "Kokujo", "hash")
		require.NoError(t, err)

		err = db.Transaction(func(tx *gorm.DB) error {
			locked, err := users.LockIn(tx, []string{shobu.ID, "00000000-0000-0000-0000-000000000000"}, false)
			require.NoError(t, err)
			assert.Equal(t, []string{shobu.ID}, slices.Collect(maps.Keys(locked)))

			locked, err = users.LockIn(tx, []string{shobu.ID}, true)
			require.NoError(t, err)
			assert.ElementsMatch(t, []string{boss.ID, shobu.ID}, slices.Collect(maps.Keys(locked)))
			return nil
		})
		require.NoError(t, err)
	})
}

func TestComputerAccountsAreNotPeople(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		service := newService(db)

		computer, err := service.CreateComputer(ctx, "Grimwhisper")
		require.NoError(t, err)
		assert.True(t, computer.Computer)
		assert.Empty(t, computer.PasswordHash, "nobody can log in to it")
		assert.Equal(t, users.RolePlayer, computer.Role)

		exist, err := service.AnyExist(ctx)
		require.NoError(t, err)
		assert.False(t, exist, "a computer opponent's account is not a person's")

		first, err := service.Create(ctx, "Shobu", "hash")
		require.NoError(t, err)
		assert.True(t, first.IsAdmin(), "the first person to sign up is still the admin")

		_, err = service.CreateComputer(ctx, "SHOBU")
		assert.ErrorIs(t, err, users.ErrUsernameTaken, "a computer opponent's name is a name like any other")
		_, err = service.Create(ctx, "grimwhisper", "hash")
		assert.ErrorIs(t, err, users.ErrUsernameTaken, "and nobody can sign up as one")

		stored, err := service.ByID(ctx, computer.ID)
		require.NoError(t, err)
		assert.True(t, stored.Computer)
	})
}
