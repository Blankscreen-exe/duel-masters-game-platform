package store_test

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

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

func TestTimesSortAndCompareAsTimes(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		// As RFC 3339 text, without its trailing zeros, the earlier of these
		// two times sorted after the later.
		earlier := time.Date(2026, 10, 26, 12, 0, 0, 123_400_000, time.UTC)
		later := time.Date(2026, 10, 26, 12, 0, 0, 123_450_000, time.UTC)
		for key, at := range map[string]time.Time{"test.later": later, "test.earlier": earlier} {
			require.NoError(t, db.Exec("INSERT INTO settings (key, value, updated_at) VALUES (?, ?, ?)", key, "x", at).Error)
		}

		var ordered []string
		require.NoError(t, db.Raw("SELECT key FROM settings ORDER BY updated_at").Scan(&ordered).Error)
		assert.Equal(t, []string{"test.earlier", "test.later"}, ordered)

		var after []string
		require.NoError(t, db.Raw("SELECT key FROM settings WHERE updated_at > ?", earlier).Scan(&after).Error)
		assert.Equal(t, []string{"test.later"}, after)

		var read time.Time
		require.NoError(t, db.Raw("SELECT updated_at FROM settings WHERE key = ?", "test.later").Scan(&read).Error)
		assert.True(t, later.Equal(read), "read back as written: %s", read)
	})
}

func TestMigrateRewritesSQLiteTimesAsNumbers(t *testing.T) {
	db := storetest.OpenUnmigrated(t, config.SQLite)
	require.NoError(t, store.MigrateTo(db, "0008_add_ratings"))

	// Times as the driver used to write them: RFC 3339 text.
	for key, at := range map[string]string{"test.later": "2026-10-26T12:00:00.12345Z", "test.earlier": "2026-10-26T12:00:00.1234Z"} {
		require.NoError(t, db.Exec("INSERT INTO settings (key, value, updated_at) VALUES (?, ?, ?)", key, "x", at).Error)
	}
	var ordered []string
	require.NoError(t, db.Raw("SELECT key FROM settings ORDER BY updated_at").Scan(&ordered).Error)
	require.Equal(t, []string{"test.later", "test.earlier"}, ordered, "text sorted wrongly")

	require.NoError(t, store.Migrate(db))

	var kinds []string
	require.NoError(t, db.Raw("SELECT DISTINCT typeof(updated_at) FROM settings").Scan(&kinds).Error)
	assert.Equal(t, []string{"integer"}, kinds)
	ordered = nil
	require.NoError(t, db.Raw("SELECT key FROM settings ORDER BY updated_at").Scan(&ordered).Error)
	assert.Equal(t, []string{"test.earlier", "test.later"}, ordered)

	var read time.Time
	require.NoError(t, db.Raw("SELECT updated_at FROM settings WHERE key = ?", "test.earlier").Scan(&read).Error)
	assert.True(t, time.Date(2026, 10, 26, 12, 0, 0, 123_400_000, time.UTC).Equal(read), "the same time: %s", read)

	require.NoError(t, store.RollbackTo(db, "0008_add_ratings"))
	var texts []string
	require.NoError(t, db.Raw("SELECT updated_at FROM settings ORDER BY key").Scan(&texts).Error)
	assert.Equal(t, []string{"2026-10-26T12:00:00.1234Z", "2026-10-26T12:00:00.12345Z"}, texts, "undone, back to text")
}
