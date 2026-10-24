package store_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/store/storetest"
)

// The data-layer code checks these rules too; the schema enforces them so no
// path into the database can break them.

func insertUser(db *gorm.DB, id, username, role string) error {
	now := time.Now().UTC()
	return db.Exec(
		"INSERT INTO users (id, username, username_key, password_hash, role, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		id, username, lower(username), "hash", role, now, now,
	).Error
}

func insertSession(db *gorm.DB, id, userID string) error {
	now := time.Now().UTC()
	return db.Exec(
		"INSERT INTO sessions (id, user_id, created_at, expires_at) VALUES (?, ?, ?, ?)",
		id, userID, now, now.Add(time.Hour),
	).Error
}

func lower(s string) string {
	out := []rune(s)
	for i, r := range out {
		if r >= 'A' && r <= 'Z' {
			out[i] = r + ('a' - 'A')
		}
	}
	return string(out)
}

func TestSchemaAllowsOnlyKnownRoles(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		require.NoError(t, insertUser(db, "u1", "Alice", "player"))
		require.NoError(t, insertUser(db, "u2", "Bob", "admin"))

		assert.Error(t, insertUser(db, "u3", "Carol", "superuser"))
	})
}

func TestSchemaKeepsUsernamesUniqueRegardlessOfCase(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		require.NoError(t, insertUser(db, "u1", "Shobu", "player"))

		err := insertUser(db, "u2", "SHOBU", "player")
		assert.True(t, errors.Is(err, gorm.ErrDuplicatedKey), "got %v", err)
	})
}

func TestSchemaTiesSessionsToExistingUsers(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		err := insertSession(db, "s1", "no-such-user")
		assert.True(t, errors.Is(err, gorm.ErrForeignKeyViolated), "got %v", err)
	})
}

func TestSchemaDeletesSessionsWithTheirUser(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		require.NoError(t, insertUser(db, "u1", "Alice", "player"))
		require.NoError(t, insertUser(db, "u2", "Bob", "player"))
		require.NoError(t, insertSession(db, "s1", "u1"))
		require.NoError(t, insertSession(db, "s2", "u1"))
		require.NoError(t, insertSession(db, "s3", "u2"))

		require.NoError(t, db.Exec("DELETE FROM users WHERE id = ?", "u1").Error)

		var left []string
		require.NoError(t, db.Raw("SELECT id FROM sessions ORDER BY id").Scan(&left).Error)
		assert.Equal(t, []string{"s3"}, left)
	})
}
