package store

import "gorm.io/gorm"

// addColumns adds a migration's columns to a table others depend on, leaving
// those a rollback left in place on SQLite (see dropColumns).
func addColumns(tx *gorm.DB, model any, fields ...string) error {
	for _, field := range fields {
		if tx.Migrator().HasColumn(model, field) {
			continue
		}
		if err := tx.Migrator().AddColumn(model, field); err != nil {
			return err
		}
	}
	return nil
}

// dropColumns undoes addColumns. SQLite can drop a column only by rebuilding the
// table, and with foreign keys on, dropping the old table deletes everything that
// cascades from its rows: a user's sessions and decks, a format's sets and
// limits. So there the columns stay, which nothing reads any more, with every row
// reset to starting, the values given by column name, so that migrating again
// leaves the same rows as on a database that really dropped them.
func dropColumns(tx *gorm.DB, model any, starting map[string]any, fields ...string) error {
	if tx.Dialector.Name() == sqliteDialect {
		return tx.Session(&gorm.Session{AllowGlobalUpdate: true}).Model(model).Updates(starting).Error
	}

	for _, field := range fields {
		if err := tx.Migrator().DropColumn(model, field); err != nil {
			return err
		}
	}
	return nil
}
