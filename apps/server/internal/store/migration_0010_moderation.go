package store

import (
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// user0010 is the users table's moderation columns as migration 0010 adds
// them. See package users.
type user0010 struct {
	// BannedAt is when the player was banned, null for a player never banned
	// or whose ban an admin lifted. BannedUntil is when the ban lifts itself,
	// null for a ban for good; a ban past it is over.
	BannedAt    *time.Time
	BannedUntil *time.Time
	// BanReason is what the player is shown when they try to log in.
	BanReason string `gorm:"type:text;not null;default:''"`
	// MutedAt and MutedUntil are the same for keeping the player out of the
	// lobby chat.
	MutedAt    *time.Time
	MutedUntil *time.Time
}

func (user0010) TableName() string { return "users" }

// moderationColumns0010 are the users columns migration 0010 adds.
var moderationColumns0010 = []string{"BannedAt", "BannedUntil", "BanReason", "MutedAt", "MutedUntil"}

// adminAction0010 is the admin_actions table as migration 0010 creates it:
// the admin log. See package adminlog.
type adminAction0010 struct {
	ID string `gorm:"primaryKey;size:36"`
	// AdminID is the admin who acted, null once their account is gone;
	// AdminName keeps their name as it was.
	AdminID   *string   `gorm:"size:36;index:idx_admin_actions_admin_id"`
	Admin     *user0002 `gorm:"constraint:OnDelete:SET NULL"`
	AdminName string    `gorm:"size:20;not null"`
	// Action is what was done, such as "user_banned".
	Action string `gorm:"size:40;not null;index:idx_admin_actions_action"`
	// Subject is what it was done to, as shown in the log, such as a player's
	// name or a setting's key; Details say how, such as a ban's length.
	Subject   string    `gorm:"type:text;not null"`
	Details   string    `gorm:"type:text;not null"`
	CreatedAt time.Time `gorm:"not null;index:idx_admin_actions_created_at"`
}

func (adminAction0010) TableName() string { return "admin_actions" }

// migration0010Moderation lets admins ban and mute players, and keeps the
// admin log.
func migration0010Moderation() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "0010_add_moderation",
		Migrate: func(tx *gorm.DB) error {
			if err := addColumns(tx, &user0010{}, moderationColumns0010...); err != nil {
				return err
			}
			return tx.Migrator().CreateTable(&adminAction0010{})
		},
		Rollback: func(tx *gorm.DB) error {
			if err := tx.Migrator().DropTable(&adminAction0010{}); err != nil {
				return err
			}
			starting := map[string]any{"banned_at": nil, "banned_until": nil, "ban_reason": "", "muted_at": nil, "muted_until": nil}
			return dropColumns(tx, &user0010{}, starting, moderationColumns0010...)
		},
	}
}
