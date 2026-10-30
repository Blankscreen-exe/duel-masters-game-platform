package store

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// user0008 is the users table's rating columns as migration 0008 adds them.
// See package ratings.
type user0008 struct {
	// Rating is the player's Elo rating; everyone starts at 1000.
	Rating int `gorm:"not null;default:1000;index:idx_users_rating"`
	// Wins and Losses count the games that counted.
	Wins   int `gorm:"not null;default:0"`
	Losses int `gorm:"not null;default:0"`
	// Streak is the current run of wins, above zero, or of losses, below.
	Streak int `gorm:"not null;default:0"`
}

func (user0008) TableName() string { return "users" }

// game0008 is the games table's columns as migration 0008 adds them: each
// player's deck civilizations and what the game did to their rating.
type game0008 struct {
	// HostCivilizations and GuestCivilizations are the civilizations of the
	// decks as played, as a JSON list, for history to show with the decks.
	HostCivilizations  string `gorm:"type:text;not null;default:'[]'"`
	GuestCivilizations string `gorm:"type:text;not null;default:'[]'"`
	// The rating changes and the ratings after the game are null for a game
	// that did not count.
	HostRatingChange  *int
	HostRating        *int
	GuestRatingChange *int
	GuestRating       *int
}

func (game0008) TableName() string { return "games" }

// The columns migration 0008 adds, by table.
var (
	userColumns0008 = []string{"Rating", "Wins", "Losses", "Streak"}
	gameColumns0008 = []string{"HostCivilizations", "GuestCivilizations", "HostRatingChange", "HostRating", "GuestRatingChange", "GuestRating"}
)

// migration0008Ratings gives every player a rating, wins, losses and a
// streak, and every finished game its decks' civilizations and its rating
// changes. Games finished before it keep no civilizations or rating changes.
func migration0008Ratings() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "0008_add_ratings",
		Migrate: func(tx *gorm.DB) error {
			if err := addUserColumns(tx, &user0008{}, userColumns0008...); err != nil {
				return err
			}
			for _, field := range gameColumns0008 {
				if err := tx.Migrator().AddColumn(&game0008{}, field); err != nil {
					return err
				}
			}
			return tx.Migrator().CreateIndex(&user0008{}, "idx_users_rating")
		},
		Rollback: func(tx *gorm.DB) error {
			if err := tx.Migrator().DropIndex(&user0008{}, "idx_users_rating"); err != nil {
				return err
			}
			// Nothing cascades from a game, so its table may be rebuilt.
			for _, field := range gameColumns0008 {
				if err := tx.Migrator().DropColumn(&game0008{}, field); err != nil {
					return err
				}
			}
			starting := map[string]any{"rating": 1000, "wins": 0, "losses": 0, "streak": 0}
			return dropUserColumns(tx, &user0008{}, starting, userColumns0008...)
		},
	}
}
