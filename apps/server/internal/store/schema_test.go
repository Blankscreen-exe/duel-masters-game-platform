package store_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/store"
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

func TestSchemaStartsPlayersNeitherBannedNorMuted(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		require.NoError(t, insertUser(db, "u1", "Alice", "player"))

		var moderation struct {
			BannedAt, BannedUntil, MutedAt, MutedUntil *time.Time
			BanReason                                  string
		}
		require.NoError(t, db.Raw("SELECT banned_at, banned_until, ban_reason, muted_at, muted_until FROM users WHERE id = ?", "u1").Scan(&moderation).Error)
		assert.Nil(t, moderation.BannedAt)
		assert.Nil(t, moderation.BannedUntil)
		assert.Empty(t, moderation.BanReason)
		assert.Nil(t, moderation.MutedAt)
		assert.Nil(t, moderation.MutedUntil)
	})
}

func TestSchemaKeepsTheAdminLogOnceAnAdminIsGone(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		require.NoError(t, insertUser(db, "u1", "Alice", "admin"))
		require.NoError(t, db.Exec(
			"INSERT INTO admin_actions (id, admin_id, admin_name, action, subject, details, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
			"a1", "u1", "Alice", "user_banned", "Bob", "7 days", time.Now().UTC(),
		).Error)

		require.NoError(t, db.Exec("DELETE FROM users WHERE id = ?", "u1").Error)

		var kept struct {
			AdminID   *string
			AdminName string
		}
		require.NoError(t, db.Raw("SELECT admin_id, admin_name FROM admin_actions WHERE id = ?", "a1").Scan(&kept).Error)
		assert.Nil(t, kept.AdminID)
		assert.Equal(t, "Alice", kept.AdminName)
	})
}

func TestSchemaStartsAccountsAsPeople(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		require.NoError(t, insertUser(db, "u1", "Shobu", "player"))

		var computer bool
		require.NoError(t, db.Raw("SELECT computer FROM users WHERE id = ?", "u1").Scan(&computer).Error)
		assert.False(t, computer)
	})
}

func TestUndoingComputerAccountsKeepsEveryonesSessions(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		require.NoError(t, insertUser(db, "u1", "Shobu", "player"))
		require.NoError(t, insertSession(db, "s1", "u1"))

		require.NoError(t, store.RollbackTo(db, "0011_create_starter_decks"))
		var sessions int64
		require.NoError(t, db.Table("sessions").Count(&sessions).Error)
		assert.Equal(t, int64(1), sessions, "undoing the migration takes nobody's sessions")

		require.NoError(t, store.Migrate(db), "and it can run again")
		var computer bool
		require.NoError(t, db.Raw("SELECT computer FROM users WHERE id = ?", "u1").Scan(&computer).Error)
		assert.False(t, computer)
	})
}

func TestUndoingUserAvatarsKeepsEveryAccount(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		require.NoError(t, insertUser(db, "u1", "Shobu", "player"))
		require.NoError(t, insertSession(db, "s1", "u1"))
		require.NoError(t, db.Table("users").Where("id = ?", "u1").Update("avatar_id", "dm01-spells-01").Error)

		require.NoError(t, store.RollbackTo(db, "0015_format_civilizations"))
		var sessions int64
		require.NoError(t, db.Table("sessions").Count(&sessions).Error)
		assert.Equal(t, int64(1), sessions, "undoing the migration takes nobody's sessions")

		require.NoError(t, store.Migrate(db), "and it can run again")
		var avatar string
		require.NoError(t, db.Raw("SELECT avatar_id FROM users WHERE id = ?", "u1").Scan(&avatar).Error)
		assert.Empty(t, avatar, "as on a database that really dropped the column")
	})
}

func TestUndoingFormatCivilizationsKeepsEveryFormatsRules(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		require.NoError(t, store.Migrate(db))
		var sets int64
		require.NoError(t, db.Table("format_sets").Count(&sets).Error)
		require.Positive(t, sets)
		require.NoError(t, db.Table("formats").Where("name = ?", "Open").Updates(map[string]any{"min_civilizations": 2, "max_civilizations": 2}).Error)

		require.NoError(t, store.RollbackTo(db, "0014_format_retirement"))
		var left int64
		require.NoError(t, db.Table("format_sets").Count(&left).Error)
		assert.Equal(t, sets, left, "undoing the migration takes no format's sets")

		require.NoError(t, store.Migrate(db), "and it can run again")
		var ruled int64
		require.NoError(t, db.Table("formats").Where("min_civilizations <> 0 OR max_civilizations <> 0").Count(&ruled).Error)
		assert.Zero(t, ruled, "as on a database that really dropped the columns")
	})
}

func TestUndoingFormatRetirementKeepsEveryFormatsRules(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		count := func(table string) int64 {
			var rows int64
			require.NoError(t, db.Table(table).Count(&rows).Error)
			return rows
		}
		sets := count("format_sets")
		require.Positive(t, sets)
		require.NoError(t, db.Table("formats").Where("name = ?", "Open").Update("retired_at", time.Now()).Error)

		require.NoError(t, store.RollbackTo(db, "0013_create_computer_opponents"))
		assert.Equal(t, sets, count("format_sets"), "undoing the migration takes no format's sets")
		assert.Equal(t, int64(2), count("formats"))

		require.NoError(t, store.Migrate(db), "and it can run again")
		var retired int64
		require.NoError(t, db.Table("formats").Where("retired_at IS NOT NULL").Count(&retired).Error)
		assert.Zero(t, retired, "as on a database that really dropped the column")
	})
}

func TestUndoingRatingsKeepsEveryonesSessions(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		require.NoError(t, insertUser(db, "u1", "Shobu", "player"))
		require.NoError(t, insertSession(db, "s1", "u1"))
		require.NoError(t, db.Table("users").Where("id = ?", "u1").Updates(map[string]any{
			"rating": 1500, "wins": 9, "losses": 2, "streak": 3,
		}).Error)

		require.NoError(t, store.RollbackTo(db, "0007_create_games"))
		var sessions int64
		require.NoError(t, db.Table("sessions").Count(&sessions).Error)
		assert.Equal(t, int64(1), sessions, "undoing the migration takes nobody's sessions")

		require.NoError(t, store.Migrate(db), "and it can run again")

		var standing struct{ Rating, Wins, Losses, Streak int }
		require.NoError(t, db.Table("users").Where("id = ?", "u1").Take(&standing).Error)
		assert.Equal(t, 1000, standing.Rating, "everyone starts again")
		assert.Zero(t, standing.Wins+standing.Losses+standing.Streak)
		assert.True(t, db.Migrator().HasIndex("users", "idx_users_rating"))
	})
}

func TestUndoingModerationKeepsEveryonesSessions(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		require.NoError(t, insertUser(db, "u1", "Shobu", "player"))
		require.NoError(t, insertSession(db, "s1", "u1"))
		require.NoError(t, db.Table("users").Where("id = ?", "u1").Updates(map[string]any{
			"banned_at": time.Now(), "banned_until": time.Now(), "ban_reason": "Cheating", "muted_at": time.Now(), "muted_until": time.Now(),
		}).Error)

		require.NoError(t, store.RollbackTo(db, "0009_store_sqlite_times_as_numbers"))
		var sessions int64
		require.NoError(t, db.Table("sessions").Count(&sessions).Error)
		assert.Equal(t, int64(1), sessions, "undoing the migration takes nobody's sessions")
		assert.False(t, db.Migrator().HasTable("admin_actions"))

		require.NoError(t, store.Migrate(db), "and it can run again")

		var moderation struct {
			BannedAt, BannedUntil, MutedAt, MutedUntil *time.Time
			BanReason                                  string
		}
		require.NoError(t, db.Table("users").Where("id = ?", "u1").Take(&moderation).Error)
		assert.Nil(t, moderation.BannedAt, "nobody comes back banned")
		assert.Nil(t, moderation.BannedUntil)
		assert.Nil(t, moderation.MutedAt, "or muted")
		assert.Nil(t, moderation.MutedUntil)
		assert.Empty(t, moderation.BanReason)
		assert.True(t, db.Migrator().HasTable("admin_actions"))
	})
}
