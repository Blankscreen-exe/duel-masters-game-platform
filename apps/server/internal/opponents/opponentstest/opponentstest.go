// Package opponentstest helps tests play the computer opponents: one added to
// the database, and games against the computer in a lobby's world.
package opponentstest

import (
	"context"
	"maps"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/computer"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/decks"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/lobby/lobbytest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/opponents"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/store/storetest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/users"
)

// Added is a computer opponent added to the database.
type Added struct {
	ID      string
	Account users.User
}

// Add adds an enabled computer opponent, playing in the rush style with the
// deck, to the database.
func Add(t *testing.T, db *gorm.DB, name, formatID string, deck []decks.Entry) Added {
	t.Helper()

	account, err := users.NewService(db, storetest.Logger()).CreateComputer(context.Background(), name)
	require.NoError(t, err)

	id := uuid.NewString()
	now := time.Now()
	require.NoError(t, db.Table("computer_opponents").Create(map[string]any{
		"id": id, "user_id": account.ID, "description": "A test opponent.", "style": "rush", "strength": string(computer.Normal),
		"format_id": formatID, "max_civilizations": 0, "card_types": `["Creature","Evolution Creature","Spell"]`,
		"enabled": true, "sort_order": 1, "created_at": now, "updated_at": now,
	}).Error)
	for _, entry := range deck {
		require.NoError(t, db.Table("computer_opponent_cards").Create(map[string]any{
			"computer_opponent_id": id, "card_id": entry.CardID, "count": entry.Count,
		}).Error)
	}

	return Added{ID: id, Account: account}
}

// Retire stops players being offered a computer opponent.
func Retire(t *testing.T, db *gorm.DB, id string) {
	t.Helper()
	require.NoError(t, db.Table("computer_opponents").Where("id = ?", id).Update("enabled", false).Error)
}

// NewChallenges is games against the computer in the world, which must have
// been given opponents.Settings, with the computer moving as soon as it can.
func NewChallenges(t *testing.T, w *lobbytest.World) *opponents.Challenges {
	t.Helper()
	ctx := context.Background()

	_, err := w.Settings.Set(ctx, opponents.Pause.Key, "0")
	require.NoError(t, err)

	known, err := w.Catalogue.Cards(ctx, w.CardIDs)
	require.NoError(t, err)

	return opponents.NewChallenges(opponents.ChallengeDependencies{
		Opponents: opponents.NewService(w.DB, w.Catalogue, w.Formats),
		Formats:   w.Formats,
		Lobby:     w.Lobby,
		Games:     w.Games,
		Library:   computer.NewLibrary(slices.Collect(maps.Values(known))),
		Settings:  w.Settings,
		Logger:    storetest.Logger(),
	})
}
