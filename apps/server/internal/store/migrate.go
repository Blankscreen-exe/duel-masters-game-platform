package store

import (
	"fmt"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// migrationsTable records which migrations a database has run.
const migrationsTable = "schema_migrations"

// migrations are every schema change, in the order they run. A migration is
// never changed once released: a later change gets a new migration appended
// here. Each migration declares the tables as they were at that point, rather
// than using the features' current models, so it keeps doing the same thing as
// the models change.
func migrations() []*gormigrate.Migration {
	return []*gormigrate.Migration{
		migration0001Settings(),
		migration0002Accounts(),
		migration0003CardCatalogue(),
		migration0004Formats(),
		migration0005Decks(),
		migration0006LobbyChat(),
		migration0007Games(),
		migration0008Ratings(),
		migration0009SQLiteTimesAsNumbers(),
		migration0010Moderation(),
		migration0011StarterDecks(),
		migration0012ComputerAccounts(),
	}
}

// Migrate brings the schema up to date, running in order, each in its own
// transaction, the migrations the database has not run yet. It refuses a
// database that has run migrations this server does not know, such as one
// last used by a newer version.
func Migrate(db *gorm.DB) error {
	if err := migrator(db).Migrate(); err != nil {
		return fmt.Errorf("migrating the database: %w", err)
	}

	return nil
}

func migrator(db *gorm.DB) *gormigrate.Gormigrate {
	return gormigrate.New(db, &gormigrate.Options{
		TableName:                 migrationsTable,
		IDColumnName:              "id",
		IDColumnSize:              255,
		UseTransaction:            true,
		ValidateUnknownMigrations: true,
	}, migrations())
}
