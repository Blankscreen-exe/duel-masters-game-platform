package store

import (
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// The card catalogue's tables as migration 0003 creates them. The catalogue
// is copied from the card database's files at startup; see package cards.

type cardSet0003 struct {
	Code string `gorm:"primaryKey;size:16"`
	// SortOrder is the set's place in release order.
	SortOrder int    `gorm:"not null"`
	IconPath  string `gorm:"type:text;not null"`
}

func (cardSet0003) TableName() string { return "card_sets" }

type card0003 struct {
	// ID is the card's first printing's id in the card database.
	ID        string `gorm:"primaryKey;size:16"`
	Name      string `gorm:"size:200;not null;uniqueIndex:idx_cards_name"`
	Type      string `gorm:"size:40;not null;index:idx_cards_type"`
	Cost      int    `gorm:"not null;index:idx_cards_cost"`
	Power     *int
	PowerText string `gorm:"size:20;not null"`
	RulesText string `gorm:"type:text;not null"`
	// SearchText is the name and rules text in lowercase without accents, so
	// searching behaves the same on every database.
	SearchText string `gorm:"type:text;not null"`
	// EngineID is the rules engine's id, null while the engine cannot play the
	// card.
	EngineID *string `gorm:"size:36;uniqueIndex:idx_cards_engine_id"`
	// SortOrder is the card's place in name order, worked out once so every
	// database lists cards alike.
	SortOrder int `gorm:"not null;index:idx_cards_sort_order"`
	// InDatabase is false for a card no longer in the card database. Its row
	// stays, so nothing referring to it breaks.
	InDatabase bool `gorm:"not null"`
}

func (card0003) TableName() string { return "cards" }

type cardPrinting0003 struct {
	ID              string      `gorm:"primaryKey;size:16"`
	CardID          string      `gorm:"size:16;not null;index:idx_card_printings_card_id"`
	Card            card0003    `gorm:"constraint:OnDelete:CASCADE"`
	SetCode         string      `gorm:"size:16;not null;index:idx_card_printings_set_code"`
	Set             cardSet0003 `gorm:"foreignKey:SetCode;references:Code"`
	SortOrder       int         `gorm:"not null"`
	CollectorNumber string      `gorm:"size:20;not null"`
	Rarity          string      `gorm:"size:40;not null"`
	RarityCode      string      `gorm:"size:10;not null"`
	// ImagePath is the picture inside the card database's data folder, empty
	// when it is missing.
	ImagePath string `gorm:"type:text;not null"`
}

func (cardPrinting0003) TableName() string { return "card_printings" }

type cardCivilization0003 struct {
	CardID       string   `gorm:"primaryKey;size:16"`
	Card         card0003 `gorm:"constraint:OnDelete:CASCADE"`
	Civilization string   `gorm:"primaryKey;size:20;index:idx_card_civilizations_civilization"`
}

func (cardCivilization0003) TableName() string { return "card_civilizations" }

type cardRace0003 struct {
	CardID    string   `gorm:"primaryKey;size:16"`
	Card      card0003 `gorm:"constraint:OnDelete:CASCADE"`
	Race      string   `gorm:"primaryKey;size:100;index:idx_card_races_race"`
	SortOrder int      `gorm:"not null"`
}

func (cardRace0003) TableName() string { return "card_races" }

type cardKeyword0003 struct {
	CardID  string   `gorm:"primaryKey;size:16"`
	Card    card0003 `gorm:"constraint:OnDelete:CASCADE"`
	Keyword string   `gorm:"primaryKey;size:60;index:idx_card_keywords_keyword"`
}

func (cardKeyword0003) TableName() string { return "card_keywords" }

type cardCategory0003 struct {
	CardID   string   `gorm:"primaryKey;size:16"`
	Card     card0003 `gorm:"constraint:OnDelete:CASCADE"`
	Category string   `gorm:"primaryKey;size:60;index:idx_card_categories_category"`
}

func (cardCategory0003) TableName() string { return "card_categories" }

type cardAbility0003 struct {
	CardID    string   `gorm:"primaryKey;size:16"`
	Card      card0003 `gorm:"constraint:OnDelete:CASCADE"`
	SortOrder int      `gorm:"primaryKey"`
	Text      string   `gorm:"type:text;not null"`
	Reminder  string   `gorm:"type:text;not null"`
}

func (cardAbility0003) TableName() string { return "card_abilities" }

// cardCatalogueState0003 records which version of the card database and the
// engine's cards the catalogue was last copied from. It has one row.
type cardCatalogueState0003 struct {
	ID          int       `gorm:"primaryKey;autoIncrement:false"`
	Fingerprint string    `gorm:"size:64;not null"`
	SyncedAt    time.Time `gorm:"not null"`
}

func (cardCatalogueState0003) TableName() string { return "card_catalogue_state" }

// migration0003CardCatalogue creates the card catalogue's tables.
func migration0003CardCatalogue() *gormigrate.Migration {
	tables := []any{
		&cardSet0003{},
		&card0003{},
		&cardPrinting0003{},
		&cardCivilization0003{},
		&cardRace0003{},
		&cardKeyword0003{},
		&cardCategory0003{},
		&cardAbility0003{},
		&cardCatalogueState0003{},
	}

	return &gormigrate.Migration{
		ID: "0003_create_card_catalogue",
		Migrate: func(tx *gorm.DB) error {
			return tx.Migrator().CreateTable(tables...)
		},
		Rollback: func(tx *gorm.DB) error {
			reversed := make([]any, 0, len(tables))
			for i := len(tables) - 1; i >= 0; i-- {
				reversed = append(reversed, tables[i])
			}
			return tx.Migrator().DropTable(reversed...)
		},
	}
}
