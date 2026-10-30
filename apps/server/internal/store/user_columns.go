package store

import "gorm.io/gorm"

// addUserColumns adds a migration's columns to the users table, leaving those
// a rollback left in place on SQLite (see dropUserColumns).
func addUserColumns(tx *gorm.DB, model any, fields ...string) error {
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

// dropUserColumns undoes addUserColumns. SQLite can drop a column only by
// rebuilding the table, and with foreign keys on, dropping the old users table
// deletes its rows' sessions, decks and everything else that cascades from
// them. So there the columns stay, which nothing reads any more, with every
// row reset to starting, the values given by column name, so that migrating
// again leaves the same accounts as on a database that really dropped them.
func dropUserColumns(tx *gorm.DB, model any, starting map[string]any, fields ...string) error {
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
