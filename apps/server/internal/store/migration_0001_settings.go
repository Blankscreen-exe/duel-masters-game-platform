package store

import (
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// setting0001 is the settings table as migration 0001 creates it.
type setting0001 struct {
	Key       string    `gorm:"primaryKey;size:100"`
	Value     string    `gorm:"type:text;not null"`
	UpdatedAt time.Time `gorm:"not null"`
}

func (setting0001) TableName() string { return "settings" }

// migration0001Settings creates the key-value table for the site-wide settings
// admins change in the admin panel.
func migration0001Settings() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "0001_create_settings",
		Migrate: func(tx *gorm.DB) error {
			return tx.Migrator().CreateTable(&setting0001{})
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropTable(&setting0001{})
		},
	}
}
