package store

import (
	"fmt"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// seedsTable records which data seeds a database has had.
const seedsTable = "data_seeds"

// RunSeeds adds, once each, the data a site begins with that refers to the
// card catalogue, such as the starter decks: the schema's migrations run
// before the catalogue is copied in, so they cannot. Each seed runs in its
// own transaction and is recorded, so a seed an admin has since changed or
// emptied is never added again. A seed, like a migration, never changes once
// released; a later one is added after it.
func RunSeeds(db *gorm.DB, seeds ...*gormigrate.Migration) error {
	if len(seeds) == 0 {
		return nil
	}

	m := gormigrate.New(db, &gormigrate.Options{
		TableName:      seedsTable,
		IDColumnName:   "id",
		IDColumnSize:   255,
		UseTransaction: true,
	}, seeds)
	if err := m.Migrate(); err != nil {
		return fmt.Errorf("adding the starting data: %w", err)
	}

	return nil
}
