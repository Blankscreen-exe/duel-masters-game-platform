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

func insertCard(db *gorm.DB, id string) error {
	return db.Exec(
		"INSERT INTO cards (id, name, type, cost, power_text, rules_text, search_text, sort_order, in_database) VALUES (?, ?, 'Creature', 1, '', '', '', 1, ?)",
		id, "Card "+id, true,
	).Error
}

func insertDeck(db *gorm.DB, id, userID string) error {
	now := time.Now().UTC()
	return db.Exec("INSERT INTO decks (id, user_id, name, created_at, updated_at) VALUES (?, ?, 'Deck', ?, ?)", id, userID, now, now).Error
}

func insertDeckCard(db *gorm.DB, deckID, cardID string, count int) error {
	return db.Exec("INSERT INTO deck_cards (deck_id, card_id, count) VALUES (?, ?, ?)", deckID, cardID, count).Error
}

func TestSchemaKeepsDeckCopiesBetweenOneAndFour(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		require.NoError(t, insertUser(db, "u1", "Alice", "player"))
		require.NoError(t, insertDeck(db, "d1", "u1"))
		for _, id := range []string{"c0", "c1", "c4", "c5"} {
			require.NoError(t, insertCard(db, id))
		}

		require.NoError(t, insertDeckCard(db, "d1", "c1", 1))
		require.NoError(t, insertDeckCard(db, "d1", "c4", 4))
		assert.Error(t, insertDeckCard(db, "d1", "c0", 0))
		assert.Error(t, insertDeckCard(db, "d1", "c5", 5))
	})
}

func TestSchemaTiesDeckCardsToExistingCards(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		require.NoError(t, insertUser(db, "u1", "Alice", "player"))
		require.NoError(t, insertDeck(db, "d1", "u1"))

		err := insertDeckCard(db, "d1", "no-such-card", 1)
		assert.True(t, errors.Is(err, gorm.ErrForeignKeyViolated), "got %v", err)
	})
}

func TestSchemaDeletesDecksWithTheirUser(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		require.NoError(t, insertUser(db, "u1", "Alice", "player"))
		require.NoError(t, insertUser(db, "u2", "Bob", "player"))
		require.NoError(t, insertCard(db, "c1"))
		require.NoError(t, insertDeck(db, "d1", "u1"))
		require.NoError(t, insertDeck(db, "d2", "u2"))
		require.NoError(t, insertDeckCard(db, "d1", "c1", 4))
		require.NoError(t, insertDeckCard(db, "d2", "c1", 4))

		require.NoError(t, db.Exec("DELETE FROM users WHERE id = ?", "u1").Error)

		var decks, deckCards []string
		require.NoError(t, db.Raw("SELECT id FROM decks").Scan(&decks).Error)
		require.NoError(t, db.Raw("SELECT deck_id FROM deck_cards").Scan(&deckCards).Error)
		assert.Equal(t, []string{"d2"}, decks)
		assert.Equal(t, []string{"d2"}, deckCards)
	})
}

func TestSchemaLimitsFormatCardsToBannedOrLimited(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		var format string
		require.NoError(t, db.Raw("SELECT id FROM formats WHERE name = 'Open'").Scan(&format).Error)
		for _, id := range []string{"c0", "c2", "c3"} {
			require.NoError(t, insertCard(db, id))
		}

		limit := func(cardID string, copies int) error {
			return db.Exec("INSERT INTO format_cards (format_id, card_id, max_copies) VALUES (?, ?, ?)", format, cardID, copies).Error
		}
		require.NoError(t, limit("c0", 0))
		require.NoError(t, limit("c2", 2))
		assert.Error(t, limit("c3", 3))
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

func TestSchemaStartsPlayersAtTheStartingRating(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		require.NoError(t, insertUser(db, "u1", "Alice", "player"))

		var standing struct{ Rating, Wins, Losses, Streak int }
		require.NoError(t, db.Raw("SELECT rating, wins, losses, streak FROM users WHERE id = ?", "u1").Scan(&standing).Error)
		assert.Equal(t, struct{ Rating, Wins, Losses, Streak int }{Rating: 1000}, standing)
	})
}
