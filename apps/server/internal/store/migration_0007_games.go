package store

import (
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// game0007 is the games table as migration 0007 creates it: every game that
// started and finished, for match history. See package games.
type game0007 struct {
	// ID is the engine's match id.
	ID   string `gorm:"primaryKey;size:36"`
	Name string `gorm:"size:50;not null"`
	// FormatID is the format played; null once the format is deleted.
	// FormatName keeps its name as it was.
	FormatID   *string     `gorm:"size:36;index:idx_games_format_id"`
	Format     *format0004 `gorm:"foreignKey:FormatID;constraint:OnDelete:SET NULL"`
	FormatName string      `gorm:"size:40;not null"`

	HostID       string   `gorm:"size:36;not null;index:idx_games_host_id"`
	Host         user0002 `gorm:"foreignKey:HostID;constraint:OnDelete:RESTRICT"`
	HostDeckName string   `gorm:"size:40;not null"`
	// HostDeck is the deck as played, as JSON: a list of card ids and
	// counts, so later changes to the deck do not change the history.
	HostDeck string `gorm:"type:text;not null"`

	GuestID       string   `gorm:"size:36;not null;index:idx_games_guest_id"`
	Guest         user0002 `gorm:"foreignKey:GuestID;constraint:OnDelete:RESTRICT"`
	GuestDeckName string   `gorm:"size:40;not null"`
	GuestDeck     string   `gorm:"type:text;not null"`

	// WinnerID is null for a game without a winner.
	WinnerID *string   `gorm:"size:36"`
	Winner   *user0002 `gorm:"foreignKey:WinnerID;constraint:OnDelete:RESTRICT"`
	Ending   string    `gorm:"size:20;not null;check:chk_games_ending,ending IN ('play','surrender','disconnect','aborted')"`

	Turns     int       `gorm:"not null"`
	StartedAt time.Time `gorm:"not null"`
	EndedAt   time.Time `gorm:"not null;index:idx_games_ended_at"`
	// Counted is false for a game too short to count, or aborted.
	Counted bool `gorm:"not null"`
	// Visible and Matchmaking are the options the game was played with.
	Visible     bool `gorm:"not null"`
	Matchmaking bool `gorm:"not null"`
}

func (game0007) TableName() string { return "games" }

// migration0007Games creates the history of finished games.
func migration0007Games() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "0007_create_games",
		Migrate: func(tx *gorm.DB) error {
			return tx.Migrator().CreateTable(&game0007{})
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropTable(&game0007{})
		},
	}
}
