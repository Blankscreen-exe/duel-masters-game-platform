package store

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// format0015 is the columns migration 0015 adds to formats. See package formats.
type format0015 struct {
	ID string `gorm:"primaryKey;size:36"`
	// MinCivilizations is the fewest civilizations a deck in the format may
	// have, 0 for no fewest.
	MinCivilizations int `gorm:"not null;default:0"`
	// MaxCivilizations is the most, 0 for no most.
	MaxCivilizations int `gorm:"not null;default:0"`
}

func (format0015) TableName() string { return "formats" }

// migration0015FormatCivilizations lets a format hold decks to a number of
// civilizations. Every format so far allows any number.
func migration0015FormatCivilizations() *gormigrate.Migration {
	fields := []string{"MinCivilizations", "MaxCivilizations"}

	return &gormigrate.Migration{
		ID: "0015_format_civilizations",
		Migrate: func(tx *gorm.DB) error {
			return addColumns(tx, &format0015{}, fields...)
		},
		Rollback: func(tx *gorm.DB) error {
			starting := map[string]any{"min_civilizations": 0, "max_civilizations": 0}
			return dropColumns(tx, &format0015{}, starting, fields...)
		},
	}
}
