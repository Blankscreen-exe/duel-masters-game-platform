package store

import (
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// announcement0018 is what an admin tells every player. See package
// announcements.
type announcement0018 struct {
	ID        string `gorm:"primaryKey"`
	Title     string `gorm:"not null"`
	Body      string `gorm:"not null"`
	Important bool   `gorm:"not null;default:false"`
	// AnnouncedAt is when it last went out to players: when it was posted, made
	// important, or an admin asked to show it again.
	AnnouncedAt time.Time `gorm:"not null;index"`
	CreatedAt   time.Time `gorm:"not null"`
	UpdatedAt   time.Time `gorm:"not null"`
}

func (announcement0018) TableName() string { return "announcements" }

// announcementDismissal0018 records a player closing an important announcement's
// popup, so it does not come back.
type announcementDismissal0018 struct {
	AnnouncementID string           `gorm:"primaryKey"`
	Announcement   announcement0018 `gorm:"constraint:OnDelete:CASCADE"`
	UserID         string           `gorm:"primaryKey;index"`
	User           user0002         `gorm:"constraint:OnDelete:CASCADE"`
	DismissedAt    time.Time        `gorm:"not null"`
}

func (announcementDismissal0018) TableName() string { return "announcement_dismissals" }

// user0018 is the users table's column for when the player last looked at the
// announcements. See package users.
type user0018 struct {
	AnnouncementsSeenAt *time.Time
}

func (user0018) TableName() string { return "users" }

// migration0018Announcements lets admins tell every player something.
func migration0018Announcements() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "0018_add_announcements",
		Migrate: func(tx *gorm.DB) error {
			if err := addColumns(tx, &user0018{}, "AnnouncementsSeenAt"); err != nil {
				return err
			}
			return tx.Migrator().CreateTable(&announcement0018{}, &announcementDismissal0018{})
		},
		Rollback: func(tx *gorm.DB) error {
			if err := tx.Migrator().DropTable(&announcementDismissal0018{}, &announcement0018{}); err != nil {
				return err
			}
			return dropColumns(tx, &user0018{}, map[string]any{"announcements_seen_at": nil}, "AnnouncementsSeenAt")
		},
	}
}
