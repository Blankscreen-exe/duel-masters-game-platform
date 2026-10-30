package admin

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/adminlog"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/settings"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/users"
)

// SetSetting changes a setting, which applies at once. It returns ErrNotAdmin
// for an account that is no longer an admin, settings.ErrUnknownSetting for a
// key no feature declares, and a *settings.InvalidValueError for a value the
// setting does not accept. Setting it to the value it has records nothing.
func (s *Service) SetSetting(ctx context.Context, actor users.User, key, value string) (settings.Entry, error) {
	return s.settings.SetRecorded(ctx, key, value, func(tx *gorm.DB, before, after string) error {
		admin, err := users.LockAdminIn(tx, actor.ID, s.timestamp())
		if err != nil || before == after {
			return err
		}
		return adminlog.RecordIn(tx, admin, adminlog.SettingChanged, key, fmt.Sprintf("From %s to %s", before, after))
	})
}

// ResetSetting puts a setting back to its default. It returns the errors
// SetSetting does, but for the value. A setting at its default already records
// nothing.
func (s *Service) ResetSetting(ctx context.Context, actor users.User, key string) (settings.Entry, error) {
	return s.settings.ResetRecorded(ctx, key, func(tx *gorm.DB, before, after string) error {
		admin, err := users.LockAdminIn(tx, actor.ID, s.timestamp())
		if err != nil || before == after {
			return err
		}
		return adminlog.RecordIn(tx, admin, adminlog.SettingReset, key, fmt.Sprintf("From %s back to %s", before, after))
	})
}
