package store

import (
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// The starter decks' tables as migration 0011 creates them: ready-made
// decks every player may play. See package decks.

type starterDeck0011 struct {
	ID   string `gorm:"primaryKey;size:36"`
	Name string `gorm:"size:40;not null;uniqueIndex:idx_starter_decks_name"`
	// CoverCardID is the card shown for the deck, null to let the site pick
	// one of its cards.
	CoverCardID *string   `gorm:"size:16"`
	CoverCard   *card0003 `gorm:"foreignKey:CoverCardID;constraint:OnDelete:RESTRICT"`
	// SortOrder is the deck's place in lists; the first is every player's
	// default.
	SortOrder int `gorm:"not null"`
	// Enabled is whether players are offered the deck.
	Enabled   bool      `gorm:"not null"`
	CreatedAt time.Time `gorm:"not null"`
	UpdatedAt time.Time `gorm:"not null"`
}

func (starterDeck0011) TableName() string { return "starter_decks" }

// starterDeckCard0011 is how many copies of a card a starter deck has.
type starterDeckCard0011 struct {
	StarterDeckID string          `gorm:"primaryKey;size:36"`
	StarterDeck   starterDeck0011 `gorm:"constraint:OnDelete:CASCADE"`
	CardID        string          `gorm:"primaryKey;size:16"`
	Card          card0003        `gorm:"constraint:OnDelete:RESTRICT"`
	Count         int             `gorm:"not null;check:chk_starter_deck_cards_count,count BETWEEN 1 AND 4"`
}

func (starterDeckCard0011) TableName() string { return "starter_deck_cards" }

// migration0011StarterDecks creates the starter decks. The decks themselves
// are added once the card catalogue is copied in (see RunSeeds), as they
// name its cards.
func migration0011StarterDecks() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "0011_create_starter_decks",
		Migrate: func(tx *gorm.DB) error {
			return tx.Migrator().CreateTable(&starterDeck0011{}, &starterDeckCard0011{})
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropTable(&starterDeckCard0011{}, &starterDeck0011{})
		},
	}
}
