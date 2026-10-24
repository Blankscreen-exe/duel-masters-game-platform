package store_test

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/config"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/store"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/store/storetest"
)

func TestOpenSQLiteCreatesTheDatabaseWithSafeSettings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "folder", "platform.db")

	db, err := store.Open(context.Background(), config.Database{Driver: config.SQLite, SQLitePath: path}, storetest.Logger())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.Close(db)) })

	assert.FileExists(t, path, "the database file and its folders are created")

	var foreignKeys int
	require.NoError(t, db.Raw("PRAGMA foreign_keys").Scan(&foreignKeys).Error)
	assert.Equal(t, 1, foreignKeys, "foreign keys are enforced")

	var journalMode string
	require.NoError(t, db.Raw("PRAGMA journal_mode").Scan(&journalMode).Error)
	assert.Equal(t, "wal", journalMode)
}

func TestOpenSQLiteAppliesSettingsToEveryConnection(t *testing.T) {
	db, err := store.Open(context.Background(), config.Database{Driver: config.SQLite, SQLitePath: filepath.Join(t.TempDir(), "platform.db")}, storetest.Logger())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.Close(db)) })

	sqlDB, err := db.DB()
	require.NoError(t, err)

	// Hold several connections at once, so the pragmas are read on different
	// connections from the pool.
	ctx := context.Background()
	for range 4 {
		conn, err := sqlDB.Conn(ctx)
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, conn.Close()) })

		var foreignKeys, busyTimeout int
		require.NoError(t, conn.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&foreignKeys))
		require.NoError(t, conn.QueryRowContext(ctx, "PRAGMA busy_timeout").Scan(&busyTimeout))
		assert.Equal(t, 1, foreignKeys)
		assert.Equal(t, 10_000, busyTimeout)
	}
	assert.GreaterOrEqual(t, sqlDB.Stats().OpenConnections, 4)
}

func TestOpenRefusesAnUnreachableDatabase(t *testing.T) {
	_, err := store.Open(context.Background(), config.Database{Driver: config.Postgres, PostgresURL: "postgres://nobody:wrong@127.0.0.1:1/none?connect_timeout=2"}, storetest.Logger())
	assert.Error(t, err)
}

func TestMigrateCreatesTheSchema(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		assert.True(t, db.Migrator().HasTable("settings"))
		assert.True(t, db.Migrator().HasTable("schema_migrations"))
	})
}

func TestMigrateCanRunAgain(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		require.NoError(t, store.Migrate(db), "an up-to-date database is left as it is")

		var applied int64
		require.NoError(t, db.Table("schema_migrations").Count(&applied).Error)

		require.NoError(t, store.Migrate(db))

		var after int64
		require.NoError(t, db.Table("schema_migrations").Count(&after).Error)
		assert.Equal(t, applied, after)
	})
}

func TestMigrateRefusesADatabaseFromANewerServer(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		require.NoError(t, db.Exec("INSERT INTO schema_migrations (id) VALUES (?)", "9999_from_the_future").Error)

		assert.Error(t, store.Migrate(db))
	})
}

func TestOpenSQLiteConnectionsCanOpenAtTheSameTime(t *testing.T) {
	db, err := store.Open(context.Background(), config.Database{Driver: config.SQLite, SQLitePath: filepath.Join(t.TempDir(), "platform.db")}, storetest.Logger())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.Close(db)) })

	sqlDB, err := db.DB()
	require.NoError(t, err)

	ctx := context.Background()
	var wg sync.WaitGroup
	errs := make([]error, 8)
	for i := range errs {
		wg.Go(func() {
			conn, err := sqlDB.Conn(ctx)
			if err != nil {
				errs[i] = err
				return
			}
			defer conn.Close()

			var mode string
			if err := conn.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&mode); err != nil {
				errs[i] = err
				return
			}
			if mode != "wal" {
				errs[i] = fmt.Errorf("connection %d is in %q mode", i, mode)
			}
		})
	}
	wg.Wait()

	for _, err := range errs {
		assert.NoError(t, err)
	}
}
