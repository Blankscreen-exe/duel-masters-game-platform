package store

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// user0012 is the users table's computer column as migration 0012 adds it.
// See package users.
type user0012 struct {
	// Computer marks the account of a computer opponent, which nobody logs in
	// to and admins cannot change.
	Computer bool `gorm:"not null;default:false"`
}

func (user0012) TableName() string { return "users" }

// migration0012ComputerAccounts lets accounts belong to computer opponents.
// It adds a column rather than a role: changing the role's check would mean
// rebuilding the users table on SQLite, whose implicit delete would take
// every account's sessions and decks with it.
func migration0012ComputerAccounts() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "0012_add_computer_accounts",
		Migrate: func(tx *gorm.DB) error {
			return addUserColumns(tx, &user0012{}, "Computer")
		},
		Rollback: func(tx *gorm.DB) error {
			return dropUserColumns(tx, &user0012{}, map[string]any{"computer": false}, "Computer")
		},
	}
}
