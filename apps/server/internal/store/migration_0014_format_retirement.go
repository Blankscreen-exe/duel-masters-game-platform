package store

import (
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// format0014 is the column migration 0014 adds to formats. See package formats.
type format0014 struct {
	ID string `gorm:"primaryKey;size:36"`
	// RetiredAt is when an admin retired the format, null while players can
	// choose it.
	RetiredAt *time.Time
}

func (format0014) TableName() string { return "formats" }

// migration0014FormatRetirement lets a format be retired.
func migration0014FormatRetirement() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "0014_format_retirement",
		Migrate: func(tx *gorm.DB) error {
			return addColumns(tx, &format0014{}, "RetiredAt")
		},
		Rollback: func(tx *gorm.DB) error {
			return dropColumns(tx, &format0014{}, map[string]any{"retired_at": nil}, "RetiredAt")
		},
	}
}
