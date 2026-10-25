package store

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// sqliteDialect is the name GORM gives the SQLite dialect.
const sqliteDialect = "sqlite"

// storedTime is a time column's value in one row, as SQLite holds it.
type storedTime struct {
	RowID int64
	Value string
}

// migration0009SQLiteTimesAsNumbers rewrites the times a SQLite database
// holds as RFC 3339 text, as the driver wrote them until now, as the
// microseconds since 1970 it writes now (see sqliteDSN), so they sort and
// compare as times. PostgreSQL keeps its own time type, and is left alone.
func migration0009SQLiteTimesAsNumbers() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "0009_store_sqlite_times_as_numbers",
		Migrate: func(tx *gorm.DB) error {
			return rewriteSQLiteTimes(tx, "text", func(value string) (any, error) {
				return time.Parse(time.RFC3339Nano, value)
			})
		},
		Rollback: func(tx *gorm.DB) error {
			return rewriteSQLiteTimes(tx, "integer", func(value string) (any, error) {
				var micros int64
				if _, err := fmt.Sscan(value, &micros); err != nil {
					return nil, err
				}
				return time.UnixMicro(micros).UTC().Format(time.RFC3339Nano), nil
			})
		},
	}
}

// rewriteSQLiteTimes rewrites every value of SQLite type kind in the time
// columns of every table as convert makes it, reading each as text. It does
// nothing on another database.
func rewriteSQLiteTimes(tx *gorm.DB, kind string, convert func(value string) (any, error)) error {
	if tx.Dialector.Name() != sqliteDialect {
		return nil
	}

	tables, err := tx.Migrator().GetTables()
	if err != nil {
		return fmt.Errorf("listing the tables: %w", err)
	}

	for _, table := range tables {
		columns, err := tx.Migrator().ColumnTypes(table)
		if err != nil {
			return fmt.Errorf("listing the columns of %s: %w", table, err)
		}

		for _, column := range columns {
			if !strings.EqualFold(column.DatabaseTypeName(), "datetime") {
				continue
			}

			name := clause.Column{Name: column.Name()}
			var values []storedTime
			err := tx.Table(table).
				Select("rowid AS row_id, CAST(? AS TEXT) AS value", name).
				Where("typeof(?) = ?", name, kind).
				Scan(&values).Error
			if err != nil {
				return fmt.Errorf("reading %s.%s: %w", table, column.Name(), err)
			}

			for _, stored := range values {
				converted, err := convert(stored.Value)
				if err != nil {
					return fmt.Errorf("%s.%s holds %q, which is not a time: %w", table, column.Name(), stored.Value, err)
				}
				if err := tx.Table(table).Where("rowid = ?", stored.RowID).Update(column.Name(), converted).Error; err != nil {
					return fmt.Errorf("rewriting %s.%s: %w", table, column.Name(), err)
				}
			}
		}
	}

	return nil
}
