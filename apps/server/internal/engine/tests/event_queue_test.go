package tests

import (
	"encoding/json"
	"testing"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A move a player sends while the engine is busy, as it is while it waits on
// a question, waits its turn instead of being dropped.
func TestAMoveSentWhileTheEngineIsBusyWaitsItsTurn(t *testing.T) {
	scn, player := promptScenario(t)
	mana := manaForOne(t, player)

	creature := creatureInHand(t, player)
	_, err := scn.ActionPlayCardPrompt(player, creature.ID)
	require.NoError(t, err)

	// Ending the turn while the mana question is still open.
	endTurn, err := json.Marshal(map[string]any{"header": "end_turn"})
	require.NoError(t, err)
	scn.Match.Parse(player.Socket(), endTurn)
	require.True(t, scn.Match.HasPendingEvents())

	require.NoError(t, scn.SubmitAction(player, mana...))
	require.NoError(t, scn.WaitForEventLoop())

	assert.Equal(t, match.BATTLEZONE, creature.Zone, "the creature is played first")
	assert.NotEqual(t, player, scn.Match.CurrentPlayer(), "then the turn ends")
}
