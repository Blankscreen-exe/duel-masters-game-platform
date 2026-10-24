package tests

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/server"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/tests/scenario"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	cardStateCreatureUID = "9781089f-1aa9-4a75-b106-35e9d431e31d" // Aqua Vehicle
	cardStateSpellUID    = "7b22cc2c-3a4a-4f50-9e61-fb9646a762cd" // Aura Blast
)

func TestStateNamesTheTypeOfEachCard(t *testing.T) {
	scn := scenario.New()
	player := scn.Match.CurrentPlayer()

	creature, err := player.Player.SpawnCard(cardStateCreatureUID, match.HAND)
	require.NoError(t, err)
	spell, err := player.Player.SpawnCard(cardStateSpellUID, match.HAND)
	require.NoError(t, err)
	charged, err := player.Player.SpawnCard(cardStateCreatureUID, match.HAND)
	require.NoError(t, err)

	// Charging mana broadcasts the state.
	since, err := scn.MessageCount(player)
	require.NoError(t, err)
	require.NoError(t, scn.ActionChargeMana(player, charged.ID))

	state, err := scn.LatestState(player, since)
	require.NoError(t, err)

	assert.Equal(t, "creature", cardType(t, state.Me.Hand, creature.ID))
	assert.Equal(t, "spell", cardType(t, state.Me.Hand, spell.ID))
	assert.Equal(t, "creature", cardType(t, state.Me.Manazone, charged.ID))
}

func cardType(t *testing.T, cards []server.CardState, cardID string) string {
	t.Helper()

	for _, card := range cards {
		if card.CardID == cardID {
			return card.Type
		}
	}

	t.Fatalf("card %s is not in the state", cardID)
	return ""
}
