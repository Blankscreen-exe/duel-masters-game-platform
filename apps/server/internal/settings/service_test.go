package settings_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/settings"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/store/storetest"
)

var (
	signupsOpen = settings.Bool{
		Key:         "accounts.signups_open",
		Group:       "Accounts",
		Description: "Whether new players can sign up.",
		Default:     true,
	}
	maxDecks = settings.Number{
		Key:         "decks.max_per_player",
		Group:       "Decks",
		Description: "How many decks a player can keep.",
		Default:     50,
		Min:         1,
		Max:         500,
	}
	motd = settings.Text{
		Key:         "lobby.message_of_the_day",
		Group:       "Lobby",
		Description: "Shown at the top of the lobby.",
		Default:     "Welcome!",
		MaxLength:   20,
	}
)

func newService(t *testing.T, db *gorm.DB) *settings.Service {
	t.Helper()

	service, err := settings.NewService(context.Background(), db, storetest.Logger(), signupsOpen, maxDecks, motd)
	require.NoError(t, err)

	return service
}

func TestSettingsStartAtTheirDefaults(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		service := newService(t, db)

		assert.True(t, service.Bool(signupsOpen))
		assert.Equal(t, int64(50), service.Number(maxDecks))
		assert.Equal(t, "Welcome!", service.Text(motd))

		for _, entry := range service.List() {
			assert.True(t, entry.IsDefault, entry.Key)
			assert.True(t, entry.UpdatedAt.IsZero(), entry.Key)
		}
	})
}

func TestSetAppliesAtOnceAndPersists(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		service := newService(t, db)

		before := time.Now().UTC().Add(-time.Second)
		entry, err := service.Set(ctx, signupsOpen.Key, "false")
		require.NoError(t, err)
		assert.Equal(t, "false", entry.Value)
		assert.False(t, entry.IsDefault)
		assert.True(t, entry.UpdatedAt.After(before), "the change is timestamped")

		_, err = service.Set(ctx, maxDecks.Key, "120")
		require.NoError(t, err)
		_, err = service.Set(ctx, motd.Key, "Tournament at 8")
		require.NoError(t, err)

		assert.False(t, service.Bool(signupsOpen), "applies without a restart")
		assert.Equal(t, int64(120), service.Number(maxDecks))
		assert.Equal(t, "Tournament at 8", service.Text(motd))

		restarted := newService(t, db)
		assert.False(t, restarted.Bool(signupsOpen), "survives a restart")
		assert.Equal(t, int64(120), restarted.Number(maxDecks))
		assert.Equal(t, "Tournament at 8", restarted.Text(motd))

		stored := restarted.List()[0]
		assert.Equal(t, entry.UpdatedAt, stored.UpdatedAt.UTC(), "the time in memory equals the stored one")
	})
}

func TestSetStoresTheCanonicalValue(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		service := newService(t, db)

		entry, err := service.Set(ctx, maxDecks.Key, " 007 ")
		require.NoError(t, err)
		assert.Equal(t, "7", entry.Value)

		entry, err = service.Set(ctx, signupsOpen.Key, " true ")
		require.NoError(t, err)
		assert.Equal(t, "true", entry.Value)

		entry, err = service.Set(ctx, motd.Key, "  spaced  ")
		require.NoError(t, err)
		assert.Equal(t, "  spaced  ", entry.Value, "text is kept exactly as given")
	})
}

func TestSetChangingAValueTwiceKeepsTheLatest(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		service := newService(t, db)

		_, err := service.Set(ctx, maxDecks.Key, "10")
		require.NoError(t, err)
		_, err = service.Set(ctx, maxDecks.Key, "20")
		require.NoError(t, err)

		assert.Equal(t, int64(20), newService(t, db).Number(maxDecks))
	})
}

func TestSetRefusesInvalidValues(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		service := newService(t, db)

		for _, invalid := range []struct{ key, value, reason string }{
			{signupsOpen.Key, "yes", `"true" or "false"`},
			{signupsOpen.Key, "", `"true" or "false"`},
			{maxDecks.Key, "ten", "whole number"},
			{maxDecks.Key, "1.5", "whole number"},
			{maxDecks.Key, "0", "between 1 and 500"},
			{maxDecks.Key, "501", "between 1 and 500"},
			{motd.Key, "this message is far too long", "at most 20 characters"},
			{motd.Key, "\xff", "valid text"},
		} {
			_, err := service.Set(ctx, invalid.key, invalid.value)

			var invalidValue *settings.InvalidValueError
			require.ErrorAs(t, err, &invalidValue, "%s = %q", invalid.key, invalid.value)
			assert.Equal(t, invalid.key, invalidValue.Key)
			assert.Contains(t, invalidValue.Reason, invalid.reason)
		}

		assert.Equal(t, int64(50), service.Number(maxDecks), "nothing changed")
		for _, entry := range newService(t, db).List() {
			assert.True(t, entry.IsDefault, "%s was not stored", entry.Key)
		}
	})
}

func TestTextLimitCountsCharactersNotBytes(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		_, err := newService(t, db).Set(context.Background(), motd.Key, "デュエル・マスターズ・プレイス！")
		assert.NoError(t, err, "16 characters fit, though they take more than 20 bytes")
	})
}

func TestSetAndResetRefuseUnknownSettings(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		service := newService(t, db)

		_, err := service.Set(ctx, "accounts.unknown", "true")
		assert.ErrorIs(t, err, settings.ErrUnknownSetting)

		_, err = service.Reset(ctx, "accounts.unknown")
		assert.ErrorIs(t, err, settings.ErrUnknownSetting)
	})
}

func TestResetGoesBackToTheDefault(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		service := newService(t, db)

		_, err := service.Set(ctx, maxDecks.Key, "9")
		require.NoError(t, err)

		entry, err := service.Reset(ctx, maxDecks.Key)
		require.NoError(t, err)
		assert.Equal(t, "50", entry.Value)
		assert.True(t, entry.IsDefault)
		assert.Equal(t, int64(50), service.Number(maxDecks))
		assert.Equal(t, int64(50), newService(t, db).Number(maxDecks), "the stored value is gone")

		_, err = service.Reset(ctx, maxDecks.Key)
		assert.NoError(t, err, "resetting a setting at its default is harmless")
	})
}

func TestListShowsEverySettingInDeclarationOrder(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		service := newService(t, db)
		_, err := service.Set(context.Background(), maxDecks.Key, "75")
		require.NoError(t, err)

		entries := service.List()
		require.Len(t, entries, 3)

		assert.Equal(t, signupsOpen.Key, entries[0].Key)
		assert.Equal(t, settings.KindBoolean, entries[0].Kind)
		assert.Equal(t, "true", entries[0].Default)

		assert.Equal(t, maxDecks.Key, entries[1].Key)
		assert.Equal(t, settings.KindNumber, entries[1].Kind)
		assert.Equal(t, "75", entries[1].Value)
		assert.Equal(t, "50", entries[1].Default)
		assert.Equal(t, int64(1), entries[1].Min)
		assert.Equal(t, int64(500), entries[1].Max)
		assert.False(t, entries[1].IsDefault)

		assert.Equal(t, motd.Key, entries[2].Key)
		assert.Equal(t, settings.KindText, entries[2].Kind)
		assert.Equal(t, 20, entries[2].MaxLength)
		assert.Equal(t, "Lobby", entries[2].Group)
		assert.Equal(t, "Shown at the top of the lobby.", entries[2].Description)
	})
}

func TestStoredValuesThatNoLongerFitAreIgnored(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		now := time.Now().UTC()
		require.NoError(t, db.Exec("INSERT INTO settings (key, value, updated_at) VALUES (?, ?, ?), (?, ?, ?)",
			maxDecks.Key, "9999", now,
			"removed.feature_flag", "true", now,
		).Error)

		service := newService(t, db)
		assert.Equal(t, int64(50), service.Number(maxDecks), "an out-of-range value falls back to the default")
		assert.Len(t, service.List(), 3, "an undeclared stored setting is not listed")

		var kept int64
		require.NoError(t, db.Table("settings").Where("key = ?", "removed.feature_flag").Count(&kept).Error)
		assert.Equal(t, int64(1), kept, "it is not deleted, in case a later version declares it again")
	})
}

func TestNewServiceRefusesBrokenDefinitions(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		for _, broken := range []struct {
			name       string
			definition settings.Definition
			problem    string
		}{
			{"key without a feature", settings.Bool{Key: "signups", Group: "G", Description: "D"}, "dot-separated"},
			{"key with capitals", settings.Bool{Key: "Accounts.Open", Group: "G", Description: "D"}, "dot-separated"},
			{"missing group", settings.Bool{Key: "a.b", Description: "D"}, "group"},
			{"missing description", settings.Bool{Key: "a.b", Group: "G"}, "description"},
			{"default below min", settings.Number{Key: "a.b", Group: "G", Description: "D", Default: 0, Min: 1, Max: 5}, "between 1 and 5"},
			{"min above max", settings.Number{Key: "a.b", Group: "G", Description: "D", Default: 3, Min: 5, Max: 1}, "min 5 is above max 1"},
			{"text without a limit", settings.Text{Key: "a.b", Group: "G", Description: "D"}, "max length"},
			{"default too long", settings.Text{Key: "a.b", Group: "G", Description: "D", Default: "toolong", MaxLength: 3}, "at most 3"},
		} {
			_, err := settings.NewService(context.Background(), db, storetest.Logger(), broken.definition)
			require.Error(t, err, broken.name)
			assert.Contains(t, err.Error(), broken.problem, broken.name)
		}

		_, err := settings.NewService(context.Background(), db, storetest.Logger(), signupsOpen, signupsOpen)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "declared twice")
	})
}

func TestGettersRefuseUndeclaredSettings(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		service := newService(t, db)

		assert.Panics(t, func() {
			service.Bool(settings.Bool{Key: "accounts.other", Group: "G", Description: "D"})
		})

		changed := maxDecks
		changed.Default = 10
		assert.Panics(t, func() { service.Number(changed) }, "a definition differing from the declared one is refused")
	})
}

func TestSettingsAreSafeToUseConcurrently(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		service := newService(t, db)

		var wg sync.WaitGroup
		errs := make(chan error, 40)
		for i := range 20 {
			wg.Go(func() {
				if _, err := service.Set(ctx, maxDecks.Key, fmt.Sprint(i+1)); err != nil {
					errs <- err
				}
			})
			wg.Go(func() {
				if n := service.Number(maxDecks); n < 1 || n > 500 {
					errs <- errors.New("read an impossible value")
				}
				service.List()
			})
		}
		wg.Wait()
		close(errs)
		for err := range errs {
			assert.NoError(t, err)
		}

		assert.Equal(t, service.Number(maxDecks), newService(t, db).Number(maxDecks), "memory and database agree on the last change")
	})
}
