package admin

import (
	"context"

	"gorm.io/gorm"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/adminlog"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/auth"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/mail"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/users"
)

// LetIn lets a player use the site without confirming their email address, for
// one waiting while the site cannot send email. Letting in a player already in
// changes nothing and records nothing. It returns the errors changeAccount does.
func (s *Service) LetIn(ctx context.Context, actor users.User, playerID string) (users.User, error) {
	player, changed, err := s.changeAccount(ctx, actor, playerID, false, func(tx *gorm.DB, admin, player users.User, _ map[string]users.User) (bool, error) {
		if player.IsVerified() {
			return false, nil
		}
		if err := auth.ForgetCodeIn(tx, player.ID); err != nil {
			return false, err
		}
		if err := users.ConfirmEmailIn(tx, player.ID, s.timestamp()); err != nil {
			return false, err
		}
		return true, adminlog.RecordIn(tx, admin, adminlog.PlayerLetIn, player.Username, "")
	})
	if err != nil {
		return users.User{}, err
	}

	if changed {
		s.logger.InfoContext(ctx, "let a player in", "admin", actor.Username, "player", player.Username)
	}
	return player, nil
}

// SetMailSettings stores the mail server's settings. It returns
// mail.ErrNoSecretKey without a key to encrypt them with, a *validation.Error
// for settings breaking the rules, and ErrNotAdmin for an actor no longer an
// admin. Nothing of the settings goes in the log.
func (s *Service) SetMailSettings(ctx context.Context, actor users.User, input mail.Input) error {
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		admin, err := users.LockAdminIn(tx, actor.ID, s.timestamp())
		if err != nil {
			return err
		}
		if err := s.mail.SaveIn(ctx, tx, input, s.timestamp()); err != nil {
			return err
		}
		return adminlog.RecordIn(tx, admin, adminlog.MailSettingsChanged, "Mail server", "")
	})
	if err != nil {
		return err
	}

	s.logger.InfoContext(ctx, "changed the mail settings", "admin", actor.Username)
	return nil
}

// ClearMailSettings forgets the mail server's settings. Forgetting none records
// nothing. It returns ErrNotAdmin for an actor no longer an admin.
func (s *Service) ClearMailSettings(ctx context.Context, actor users.User) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		admin, err := users.LockAdminIn(tx, actor.ID, s.timestamp())
		if err != nil {
			return err
		}
		had, err := s.mail.ClearIn(ctx, tx)
		if err != nil || !had {
			return err
		}
		return adminlog.RecordIn(tx, admin, adminlog.MailSettingsCleared, "Mail server", "")
	})
}
