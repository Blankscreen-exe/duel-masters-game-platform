package store

import "gorm.io/gorm"

// MigrateTo runs the migrations up to and including id, so a test can put
// data in the shape a later migration changes.
func MigrateTo(db *gorm.DB, id string) error {
	return migrator(db).MigrateTo(id)
}

// RollbackTo undoes the migrations after id.
func RollbackTo(db *gorm.DB, id string) error {
	return migrator(db).RollbackTo(id)
}
