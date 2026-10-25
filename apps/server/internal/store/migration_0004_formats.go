package store

import (
	"fmt"
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// The formats' tables as migration 0004 creates them. See package formats.

type format0004 struct {
	ID          string `gorm:"primaryKey;size:36"`
	Name        string `gorm:"size:40;not null;uniqueIndex:idx_formats_name"`
	Description string `gorm:"type:text;not null"`
	// AllSets allows cards from every set, including sets added to the card
	// database later; otherwise only format_sets are allowed.
	AllSets bool `gorm:"not null"`
	// SortOrder is the format's place in lists.
	SortOrder int       `gorm:"not null"`
	CreatedAt time.Time `gorm:"not null"`
	UpdatedAt time.Time `gorm:"not null"`
}

func (format0004) TableName() string { return "formats" }

// formatSet0004 is a set a format allows. The code is not a foreign key: a
// format may name a set before the card database has it.
type formatSet0004 struct {
	FormatID string     `gorm:"primaryKey;size:36"`
	Format   format0004 `gorm:"constraint:OnDelete:CASCADE"`
	SetCode  string     `gorm:"primaryKey;size:16"`
}

func (formatSet0004) TableName() string { return "format_sets" }

// formatCard0004 limits a card in a format: 0 copies is banned, 1 or 2
// limited.
type formatCard0004 struct {
	FormatID  string     `gorm:"primaryKey;size:36"`
	Format    format0004 `gorm:"constraint:OnDelete:CASCADE"`
	CardID    string     `gorm:"primaryKey;size:16"`
	Card      card0003   `gorm:"constraint:OnDelete:RESTRICT"`
	MaxCopies int        `gorm:"not null;check:chk_format_cards_max_copies,max_copies BETWEEN 0 AND 2"`
}

func (formatCard0004) TableName() string { return "format_cards" }

// startingFormats are the formats a new site begins with (design/SPEC.md,
// section 5).
var startingFormats = []struct {
	name        string
	description string
	sets        []string
}{
	{name: "Open", description: "Every card the game can play."},
	{
		name:        "DM-01 to DM-12",
		description: "Cards from the first twelve sets, as the game was first played.",
		sets:        []string{"DM-01", "DM-02", "DM-03", "DM-04", "DM-05", "DM-06", "DM-07", "DM-08", "DM-09", "DM-10", "DM-11", "DM-12"},
	},
}

// migration0004Formats creates the formats and adds the starting ones.
func migration0004Formats() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "0004_create_formats",
		Migrate: func(tx *gorm.DB) error {
			if err := tx.Migrator().CreateTable(&format0004{}, &formatSet0004{}, &formatCard0004{}); err != nil {
				return err
			}

			now := tx.NowFunc()
			for i, starting := range startingFormats {
				id, err := uuid.NewV7()
				if err != nil {
					return fmt.Errorf("making a format id: %w", err)
				}

				format := format0004{
					ID:          id.String(),
					Name:        starting.name,
					Description: starting.description,
					AllSets:     len(starting.sets) == 0,
					SortOrder:   i + 1,
					CreatedAt:   now,
					UpdatedAt:   now,
				}
				if err := tx.Create(&format).Error; err != nil {
					return fmt.Errorf("adding format %q: %w", starting.name, err)
				}

				for _, code := range starting.sets {
					if err := tx.Create(&formatSet0004{FormatID: format.ID, SetCode: code}).Error; err != nil {
						return fmt.Errorf("adding set %s to format %q: %w", code, starting.name, err)
					}
				}
			}

			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropTable(&formatCard0004{}, &formatSet0004{}, &format0004{})
		},
	}
}
