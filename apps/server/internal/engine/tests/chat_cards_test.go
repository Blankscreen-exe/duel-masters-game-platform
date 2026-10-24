package tests

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/server"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/tests/scenario"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSummonLogNamesTheSummonedCard(t *testing.T) {
	scn := scenario.New(scenario.WithDeck(scenario.DeckEntry{UID: turnStepCreatureUID, Count: 40}))
	player := scn.Match.CurrentPlayer()
	opponent := scn.Match.PlayerRef(scn.Match.Opponent(player.Player))

	for range 2 {
		_, err := player.Player.SpawnCard(turnStepCreatureUID, match.MANAZONE)
		require.NoError(t, err)
	}
	// Both players hold a copy, so the log has to tell which one was summoned.
	_, err := opponent.Player.SpawnCard(turnStepCreatureUID, match.HAND)
	require.NoError(t, err)
	creature, err := player.Player.SpawnCard(turnStepCreatureUID, match.HAND)
	require.NoError(t, err)

	since, err := scn.MessageCount(opponent)
	require.NoError(t, err)
	require.NoError(t, scn.ActionPlayCard(player, creature.ID))
	require.NoError(t, scn.WaitForEventLoop())

	log, err := scn.ChatLog(opponent, since)
	require.NoError(t, err)

	var summon *server.ChatMessage
	for i := range log {
		if log[i].Message == player.Socket.User.Username+" summoned "+creature.Name+" to the battle zone" {
			summon = &log[i]
		}
	}
	require.NotNil(t, summon, "the summon should have been logged, got %v", log)

	assert.Equal(t, []server.ChatCard{{
		Name:    creature.Name,
		ImageID: turnStepCreatureUID,
		Owner:   player.Socket.User.Username,
		CardID:  creature.ID,
	}}, summon.Cards)
}
