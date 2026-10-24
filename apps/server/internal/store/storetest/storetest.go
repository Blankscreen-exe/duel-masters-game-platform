// Package storetest runs data-layer tests against every database the server
// supports, so code using the database is proven to behave the same on each.
package storetest

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/config"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/store"
)

// PostgresURLVariable names the environment variable holding the address of a
// PostgreSQL server the tests may create schemas in. Without it the PostgreSQL
// runs are skipped, except on CI, where they are required.
const PostgresURLVariable = "TEST_POSTGRES_URL"

// Each runs test once per database, each time as a subtest named after the
// database, with a fresh, fully migrated database of its own: a new SQLite
// file, or a new PostgreSQL schema dropped afterwards.
func Each(t *testing.T, test func(t *testing.T, db *gorm.DB)) {
	t.Helper()

	for _, driver := range []config.DatabaseDriver{config.SQLite, config.Postgres} {
		t.Run(string(driver), func(t *testing.T) {
			test(t, Open(t, driver))
		})
	}
}

// Open returns a fresh, fully migrated database of the given kind, closed when
// the test ends.
func Open(t *testing.T, driver config.DatabaseDriver) *gorm.DB {
	t.Helper()

	db := OpenUnmigrated(t, driver)
	require.NoError(t, store.Migrate(db))

	return db
}

// OpenUnmigrated returns a fresh, empty database of the given kind, closed when
// the test ends.
func OpenUnmigrated(t *testing.T, driver config.DatabaseDriver) *gorm.DB {
	t.Helper()

	var cfg config.Database
	switch driver {
	case config.SQLite:
		cfg = config.Database{Driver: config.SQLite, SQLitePath: filepath.Join(t.TempDir(), "test.db")}
	case config.Postgres:
		cfg = config.Database{Driver: config.Postgres, PostgresURL: postgresSchema(t)}
	default:
		t.Fatalf("unknown database driver %q", driver)
	}

	db, err := store.Open(context.Background(), cfg, Logger())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.Close(db)) })

	return db
}

// Logger discards everything, for tests that need a logger.
func Logger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// postgresSchema creates a schema for this test alone, dropped when it ends,
// and returns an address whose connections use it.
func postgresSchema(t *testing.T) string {
	t.Helper()

	address := os.Getenv(PostgresURLVariable)
	if address == "" {
		if os.Getenv("CI") != "" {
			t.Fatalf("%s must be set on CI, so the PostgreSQL tests run", PostgresURLVariable)
		}
		t.Skipf("set %s to run the PostgreSQL tests", PostgresURLVariable)
	}

	suffix := make([]byte, 8)
	_, err := rand.Read(suffix)
	require.NoError(t, err)
	schema := "test_" + hex.EncodeToString(suffix)

	admin, err := store.Open(context.Background(), config.Database{Driver: config.Postgres, PostgresURL: address}, Logger())
	require.NoError(t, err)
	require.NoError(t, admin.Exec(fmt.Sprintf("CREATE SCHEMA %s", schema)).Error)

	t.Cleanup(func() {
		require.NoError(t, admin.Exec(fmt.Sprintf("DROP SCHEMA %s CASCADE", schema)).Error)
		require.NoError(t, store.Close(admin))
	})

	parsed, err := url.Parse(address)
	require.NoError(t, err, "%s must be a URL such as postgres://user:password@host:5432/database", PostgresURLVariable)
	query := parsed.Query()
	query.Set("search_path", schema)
	parsed.RawQuery = query.Encode()

	return parsed.String()
}
