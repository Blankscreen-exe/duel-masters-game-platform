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

// ratingColumns0008 are the columns migration 0008 adds, by table.
var ratingColumns0008 = []struct {
	model  any
	fields []string
}{
	{&user0008{}, []string{"Rating", "Wins", "Losses", "Streak"}},
	{&game0008{}, []string{"HostCivilizations", "GuestCivilizations", "HostRatingChange", "HostRating", "GuestRatingChange", "GuestRating"}},
}

// migration0008Ratings gives every player a rating, wins, losses and a
// streak, and every finished game its decks' civilizations and its rating
// changes. Games finished before it keep no civilizations or rating changes.
func migration0008Ratings() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "0008_add_ratings",
		Migrate: func(tx *gorm.DB) error {
			for _, table := range ratingColumns0008 {
				for _, field := range table.fields {
					if err := tx.Migrator().AddColumn(table.model, field); err != nil {
						return err
					}
				}
			}
			return tx.Migrator().CreateIndex(&user0008{}, "idx_users_rating")
		},
		Rollback: func(tx *gorm.DB) error {
			if err := tx.Migrator().DropIndex(&user0008{}, "idx_users_rating"); err != nil {
				return err
			}
			for _, table := range ratingColumns0008 {
				for _, field := range table.fields {
					if err := tx.Migrator().DropColumn(table.model, field); err != nil {
						return err
					}
				}
			}
			return nil
		},
	}
}
