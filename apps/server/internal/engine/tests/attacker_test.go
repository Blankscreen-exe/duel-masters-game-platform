package tests

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/tests/scenario"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Deklowaz, the Terminator has a tap ability that needs no choices.
const attackerTestTapAbilityUID = "a808b98c-2de7-412b-970c-a3b925bf43c2"

func TestStateReportsTheAttackingCreature(t *testing.T) {
	scn := scenario.New(scenario.WithDeck(scenario.DeckEntry{UID: turnStepCreatureUID, Count: 40}))
	attacker := scn.Match.CurrentPlayer()
	defender := scn.Match.PlayerRef(scn.Match.Opponent(attacker.Player))

	for range 2 {
		_, err := attacker.Player.SpawnCard(turnStepCreatureUID, match.MANAZONE)
		require.NoError(t, err)
	}
	creature, err := attacker.Player.SpawnCard(turnStepCreatureUID, match.HAND)
	require.NoError(t, err)
	require.NoError(t, scn.ActionPlayCard(attacker, creature.ID))
	require.NoError(t, scn.WaitForEventLoop())

	t.Run("no creature is attacking before an attack", func(t *testing.T) {
		state, err := scn.LatestState(attacker, 0)
		require.NoError(t, err)
		assert.Empty(t, state.Attacker)
	})

	require.NoError(t, scn.ActionEndTurn(attacker))

	for range 6 {
		_, err := defender.Player.SpawnCard(turnStepCreatureUID, match.MANAZONE)
		require.NoError(t, err)
	}
	blocker, err := defender.Player.SpawnCard(turnStepBlockerUID, match.HAND)
	require.NoError(t, err)
	require.NoError(t, scn.ActionPlayCard(defender, blocker.ID))
	require.NoError(t, scn.WaitForEventLoop())
	require.NoError(t, scn.ActionEndTurn(defender))

	shields, err := defender.Player.Container(match.SHIELDZONE)
	require.NoError(t, err)
	require.NotEmpty(t, shields)

	blockStart, err := scn.MessageCount(defender)
	require.NoError(t, err)
	_, err = scn.ActionAttackPlayer(attacker, creature.ID)
	require.NoError(t, err)
	require.NoError(t, scn.ResolveAttack(attacker, shields[0].ID))

	// The attack now waits for the defender to decide whether to block.
	_, err = scn.WaitForAction(defender, blockStart)
	require.NoError(t, err, "the blocker should have been offered")

	t.Run("both players see the creature attacking", func(t *testing.T) {
		for _, viewer := range []*match.PlayerReference{attacker, defender} {
			state, err := scn.LatestState(viewer, 0)
			require.NoError(t, err)
			assert.Equal(t, creature.ID, state.Attacker, "as seen by %s", viewer.Socket.User.Username)
		}
	})

	t.Run("a spectator joining mid attack sees it too", func(t *testing.T) {
		spectator, err := scn.Spectate("spectator", "Spectator")
		require.NoError(t, err)

		states, err := scenario.StatesOn(spectator, 0)
		require.NoError(t, err)
		state := states[len(states)-1]
		assert.Equal(t, creature.ID, state.Attacker)
		assert.Equal(t, "attack", state.Step)
	})

	// Declining to block lets the attack through to the shield.
	require.NoError(t, scn.CancelAction(defender))
	require.NoError(t, scn.WaitForEventLoop())

	t.Run("no creature is attacking once the attack has resolved", func(t *testing.T) {
		state, err := scn.LatestState(attacker, 0)
		require.NoError(t, err)
		assert.Empty(t, state.Attacker)
	})
}

func TestStateDoesNotReportATapAbilityAsAnAttack(t *testing.T) {
	scn := scenario.New(scenario.WithDeck(scenario.DeckEntry{UID: turnStepCreatureUID, Count: 40}))
	player := scn.Match.CurrentPlayer()
	opponent := scn.Match.PlayerRef(scn.Match.Opponent(player.Player))

	// A tap ability needs a creature that was in play since the turn began.
	inHand, err := player.Player.SpawnCard(attackerTestTapAbilityUID, match.HAND)
	require.NoError(t, err)
	creature, err := player.Player.MoveCard(inHand.ID, match.HAND, match.BATTLEZONE, inHand.ID)
	require.NoError(t, err)
	require.Equal(t, match.BATTLEZONE, creature.Zone)
	require.NoError(t, scn.ActionEndTurn(player))
	require.NoError(t, scn.ActionEndTurn(opponent))

	since, err := scn.MessageCount(player)
	require.NoError(t, err)
	require.NoError(t, scn.ActionUseTapAbility(player, creature.ID))
	require.NoError(t, scn.WaitForEventLoop())
	require.True(t, creature.Tapped, "the tap ability should have been used")

	states, err := scn.StatesSince(player, since)
	require.NoError(t, err)
	require.NotEmpty(t, states)
	for _, state := range states {
		assert.Empty(t, state.Attacker)
	}
}
