package store

import (
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// The decks' tables as migration 0005 creates them. See package decks.

type deck0005 struct {
	ID     string   `gorm:"primaryKey;size:36"`
	UserID string   `gorm:"size:36;not null;index:idx_decks_user_id"`
	User   user0002 `gorm:"constraint:OnDelete:CASCADE"`
	Name   string   `gorm:"size:40;not null"`
	// CoverCardID is the card shown for the deck, null to let the site pick
	// one of its cards.
	CoverCardID *string   `gorm:"size:16"`
	CoverCard   *card0003 `gorm:"foreignKey:CoverCardID;constraint:OnDelete:RESTRICT"`
	CreatedAt   time.Time `gorm:"not null"`
	UpdatedAt   time.Time `gorm:"not null"`
}

func (deck0005) TableName() string { return "decks" }

// deckCard0005 is how many copies of a card a deck has.
type deckCard0005 struct {
	DeckID string   `gorm:"primaryKey;size:36"`
	Deck   deck0005 `gorm:"constraint:OnDelete:CASCADE"`
	CardID string   `gorm:"primaryKey;size:16"`
	Card   card0003 `gorm:"constraint:OnDelete:RESTRICT"`
	Count  int      `gorm:"not null;check:chk_deck_cards_count,count BETWEEN 1 AND 4"`
}

func (deckCard0005) TableName() string { return "deck_cards" }

// migration0005Decks creates the players' decks. A deleted account takes its
// decks with it.
func migration0005Decks() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "0005_create_decks",
		Migrate: func(tx *gorm.DB) error {
			return tx.Migrator().CreateTable(&deck0005{}, &deckCard0005{})
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropTable(&deckCard0005{}, &deck0005{})
		},
	}
}
