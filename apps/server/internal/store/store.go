// Package store opens the server's database, SQLite or PostgreSQL, through
// GORM and brings its schema up to date with versioned migrations.
package store

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/ncruces/go-sqlite3/gormlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/config"
)

const (
	// connectTimeout bounds the first round trip to the database at startup.
	connectTimeout = 10 * time.Second
	// slowQueryThreshold is how long a query may take before it is logged as
	// slow.
	slowQueryThreshold = 200 * time.Millisecond
	// sqliteBusyTimeoutMs is how long SQLite waits for a lock held by another
	// connection before giving up.
	sqliteBusyTimeoutMs = 10_000
)

// Open connects to the configured database and checks that it answers. The
// caller closes it with Close.
func Open(ctx context.Context, cfg config.Database, logger *slog.Logger) (*gorm.DB, error) {
	dialector, err := dialectorFor(cfg)
	if err != nil {
		return nil, err
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: gormlogger.NewSlogLogger(logger, gormlogger.Config{
			SlowThreshold:             slowQueryThreshold,
			LogLevel:                  gormlogger.Warn,
			IgnoreRecordNotFoundError: true,
			ParameterizedQueries:      true,
		}),
		// Both databases report a broken unique constraint as the same error,
		// gorm.ErrDuplicatedKey, and a broken foreign key as
		// gorm.ErrForeignKeyViolated.
		TranslateError: true,
		// Timestamps are stored in UTC on both databases.
		NowFunc: func() time.Time { return time.Now().UTC() },
	})
	if err != nil {
		return nil, fmt.Errorf("opening the %s database: %w", cfg.Driver, err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()

	sqlDB, err := db.DB()
	if err == nil {
		err = sqlDB.PingContext(pingCtx)
	}
	if err != nil {
		return nil, errors.Join(fmt.Errorf("connecting to the %s database: %w", cfg.Driver, err), Close(db))
	}

	if cfg.Driver == config.SQLite {
		if err := useWriteAheadLog(pingCtx, db); err != nil {
			return nil, errors.Join(err, Close(db))
		}
	}

	return db, nil
}

// useWriteAheadLog switches SQLite to write-ahead logging, so reads do not
// wait for writes. The mode is stored in the database file, so it is set once
// here, while the pool holds a single connection: switching takes an exclusive
// lock that SQLite does not wait for, so connections switching at the same
// time could fail. Every later connection opens the file in this mode.
func useWriteAheadLog(ctx context.Context, db *gorm.DB) error {
	var mode string
	if err := db.WithContext(ctx).Raw("PRAGMA journal_mode = WAL").Scan(&mode).Error; err != nil {
		return fmt.Errorf("switching SQLite to write-ahead logging: %w", err)
	}
	if mode != "wal" {
		return fmt.Errorf("switching SQLite to write-ahead logging: the database stayed in %q mode", mode)
	}

	return nil
}

// Close closes the database's connections.
func Close(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("closing the database: %w", err)
	}

	return sqlDB.Close()
}

func dialectorFor(cfg config.Database) (gorm.Dialector, error) {
	switch cfg.Driver {
	case config.SQLite:
		dsn, err := sqliteDSN(cfg.SQLitePath)
		if err != nil {
			return nil, err
		}
		return gormlite.Open(dsn), nil
	case config.Postgres:
		return postgres.Open(cfg.PostgresURL), nil
	default:
		return nil, fmt.Errorf("unknown database driver %q", cfg.Driver)
	}
}

// sqliteDSN creates the folder for the database file and returns the address
// that opens each connection with:
//   - a busy timeout, so the connection waits for another's lock;
//   - foreign keys enforced, which SQLite leaves off unless asked;
//   - normal syncing, which is safe with write-ahead logging (see
//     useWriteAheadLog);
//   - immediate transactions, which take the write lock when they begin and so
//     cannot fail half way when two connections write at once. SQLite allows
//     one writer at a time, so transactions run one after another.
func sqliteDSN(path string) (string, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return "", fmt.Errorf("creating the folder for the SQLite database: %w", err)
	}

	query := url.Values{}
	query.Add("_pragma", fmt.Sprintf("busy_timeout(%d)", sqliteBusyTimeoutMs))
	query.Add("_pragma", "foreign_keys(1)")
	query.Add("_pragma", "synchronous(normal)")
	query.Set("_txlock", "immediate")

	return (&url.URL{Scheme: "file", Opaque: filepath.ToSlash(path), RawQuery: query.Encode()}).String(), nil
}
