package tests

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/tests/scenario"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Aqua Vehicle is a vanilla 2 mana water creature, so a deck of nothing else
// has no shield triggers or blockers to interrupt an attack.
const brokenShieldsCreatureUID = "9781089f-1aa9-4a75-b106-35e9d431e31d"

func TestStateReportsBrokenShields(t *testing.T) {
	scn := scenario.New(scenario.WithDeck(scenario.DeckEntry{UID: brokenShieldsCreatureUID, Count: 40}))
	player := scn.Match.CurrentPlayer()
	opponent := scn.Match.PlayerRef(scn.Match.Opponent(player.Player))

	for range 2 {
		_, err := player.Player.SpawnCard(brokenShieldsCreatureUID, match.MANAZONE)
		require.NoError(t, err)
	}
	creature, err := player.Player.SpawnCard(brokenShieldsCreatureUID, match.HAND)
	require.NoError(t, err)
	require.NoError(t, scn.ActionPlayCard(player, creature.ID))
	require.NoError(t, scn.WaitForEventLoop())
	require.NoError(t, scn.ActionEndTurn(player))
	require.NoError(t, scn.ActionEndTurn(opponent))

	shields, err := opponent.Player.Container(match.SHIELDZONE)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(shields), 2)
	broken := shields[0]

	t.Run("a shield broken by an attack is reported once, to both players", func(t *testing.T) {
		playerSince, err := scn.MessageCount(player)
		require.NoError(t, err)
		opponentSince, err := scn.MessageCount(opponent)
		require.NoError(t, err)

		_, err = scn.ActionAttackPlayer(player, creature.ID)
		require.NoError(t, err)
		require.NoError(t, scn.ResolveAttack(player, broken.ID))
		require.NoError(t, scn.WaitForEventLoop())
		require.Equal(t, match.HAND, broken.Zone)

		for viewer, since := range map[*match.PlayerReference]int{player: playerSince, opponent: opponentSince} {
			states, err := scn.StatesSince(viewer, since)
			require.NoError(t, err)

			reported := 0
			for _, state := range states {
				for _, id := range state.BrokenShields {
					if id == broken.ID {
						reported++
					}
				}
			}
			assert.Equal(t, 1, reported, "the broken shield is reported in exactly one state for %s", viewer.Socket.User.Username)
		}
	})

	t.Run("a shield put into the hand without breaking it is not reported", func(t *testing.T) {
		moved := shields[1]
		_, err := opponent.Player.MoveCard(moved.ID, match.SHIELDZONE, match.HAND, "test-effect")
		require.NoError(t, err)

		// Ending the turn broadcasts the state several times over.
		since, err := scn.MessageCount(player)
		require.NoError(t, err)
		require.NoError(t, scn.ActionEndTurn(player))

		states, err := scn.StatesSince(player, since)
		require.NoError(t, err)
		require.NotEmpty(t, states)
		for _, state := range states {
			assert.NotNil(t, state.BrokenShields, "an empty list rather than null")
			assert.NotContains(t, state.BrokenShields, moved.ID)
		}
	})
}
