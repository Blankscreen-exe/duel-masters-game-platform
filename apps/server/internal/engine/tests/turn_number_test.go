package tests

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/tests/scenario"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Balesk Baj, the Timeburner gives its owner an extra turn after an unblocked
// attack.
const turnNumberTestExtraTurnUID = "4dbc114f-b53e-45f5-baa5-5ca33a5a3337"

// assertTurnNumber checks the turn number both players were last sent.
func assertTurnNumber(t *testing.T, scn *scenario.TestScenario, want int) {
	t.Helper()

	for _, viewer := range []*match.PlayerReference{scn.Match.Player1, scn.Match.Player2} {
		state, err := scn.LatestState(viewer, 0)
		require.NoError(t, err)
		assert.Equal(t, want, state.Turn, "as seen by %s", viewer.Username)
	}
}

func TestStateReportsTheTurnNumber(t *testing.T) {
	scn := scenario.New()
	player := scn.Match.CurrentPlayer()
	opponent := scn.Match.PlayerRef(scn.Match.Opponent(player.Player))

	t.Run("the first turn is turn one", func(t *testing.T) {
		assertTurnNumber(t, scn, 1)
	})

	t.Run("each player's turn counts", func(t *testing.T) {
		require.NoError(t, scn.ActionEndTurn(player))
		assertTurnNumber(t, scn, 2)

		require.NoError(t, scn.ActionEndTurn(opponent))
		assertTurnNumber(t, scn, 3)
	})

	t.Run("a spectator joining sees the turn under way", func(t *testing.T) {
		spectator, err := scn.Spectate("spectator", "Spectator")
		require.NoError(t, err)

		states, err := scenario.StatesOn(spectator, 0)
		require.NoError(t, err)
		require.NotEmpty(t, states)
		assert.Equal(t, 3, states[len(states)-1].Turn)
	})
}

func TestStateCountsAnExtraTurn(t *testing.T) {
	scn := scenario.New()
	player := scn.Match.CurrentPlayer()

	inHand, err := player.Player.SpawnCard(turnNumberTestExtraTurnUID, match.HAND)
	require.NoError(t, err)
	attacker, err := player.Player.MoveCard(inHand.ID, match.HAND, match.BATTLEZONE, inHand.ID)
	require.NoError(t, err)
	require.Equal(t, match.BATTLEZONE, attacker.Zone)

	action, err := scn.ActionAttackPlayer(player, attacker.ID)
	require.NoError(t, err)
	require.NoError(t, scn.ResolveAttack(player, action.Cards[0].CardID))
	require.NoError(t, scn.ActionEndTurn(player))
	require.True(t, scn.Match.IsPlayerTurn(player.Player), "the extra turn should have been taken")

	assertTurnNumber(t, scn, 2)
}
