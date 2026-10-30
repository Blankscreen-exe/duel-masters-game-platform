package store

import (
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// The computer opponents' tables as migration 0013 creates them: the
// personalities players can play against. See package opponents.

type computerOpponent0013 struct {
	ID string `gorm:"primaryKey;size:36"`
	// UserID is the opponent's account, which its games are recorded under
	// and whose name is its name.
	UserID      string   `gorm:"size:36;not null;uniqueIndex:idx_computer_opponents_user_id"`
	User        user0002 `gorm:"foreignKey:UserID;constraint:OnDelete:RESTRICT"`
	Description string   `gorm:"size:200;not null"`
	// Style is how it plays, one of the computer player's styles.
	Style string `gorm:"size:20;not null"`
	// Strength is how well it plays.
	Strength string `gorm:"size:10;not null;check:chk_computer_opponents_strength,strength IN ('easy','normal')"`
	// FormatID is the format its deck keeps to, and a player's must.
	FormatID string     `gorm:"size:36;not null;index:idx_computer_opponents_format_id"`
	Format   format0004 `gorm:"foreignKey:FormatID;constraint:OnDelete:RESTRICT"`
	// MaxCivilizations is how many civilizations its deck may have, 0 for any
	// number.
	MaxCivilizations int `gorm:"not null;check:chk_computer_opponents_max_civilizations,max_civilizations BETWEEN 0 AND 5"`
	// CardTypes are the card types its deck may have, as a JSON list of the
	// card database's names for them.
	CardTypes string `gorm:"type:text;not null"`
	// Enabled is whether players are offered it; an opponent retired is not.
	Enabled   bool      `gorm:"not null"`
	SortOrder int       `gorm:"not null"`
	CreatedAt time.Time `gorm:"not null"`
	UpdatedAt time.Time `gorm:"not null"`
}

func (computerOpponent0013) TableName() string { return "computer_opponents" }

// computerOpponentCard0013 is how many copies of a card an opponent's deck
// has.
type computerOpponentCard0013 struct {
	ComputerOpponentID string               `gorm:"primaryKey;size:36"`
	ComputerOpponent   computerOpponent0013 `gorm:"constraint:OnDelete:CASCADE"`
	CardID             string               `gorm:"primaryKey;size:16"`
	Card               card0003             `gorm:"constraint:OnDelete:RESTRICT"`
	Count              int                  `gorm:"not null;check:chk_computer_opponent_cards_count,count BETWEEN 1 AND 4"`
}

func (computerOpponentCard0013) TableName() string { return "computer_opponent_cards" }

// migration0013ComputerOpponents creates the computer opponents. The
// opponents themselves are added once the card catalogue is copied in (see
// RunSeeds), as their decks name its cards.
func migration0013ComputerOpponents() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "0013_create_computer_opponents",
		Migrate: func(tx *gorm.DB) error {
			return tx.Migrator().CreateTable(&computerOpponent0013{}, &computerOpponentCard0013{})
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropTable(&computerOpponentCard0013{}, &computerOpponent0013{})
		},
	}
}
