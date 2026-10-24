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
const turnStepCreatureUID = "9781089f-1aa9-4a75-b106-35e9d431e31d"

func TestStateReportsTheTurnStep(t *testing.T) {
	scn := scenario.New(scenario.WithDeck(scenario.DeckEntry{UID: turnStepCreatureUID, Count: 40}))
	player := scn.Match.CurrentPlayer()
	opponent := scn.Match.PlayerRef(scn.Match.Opponent(player.Player))

	assertStep := func(expected string, message string) {
		t.Helper()
		for _, viewer := range []*match.PlayerReference{player, opponent} {
			state, err := scn.LatestState(viewer, 0)
			require.NoError(t, err)
			assert.Equal(t, expected, state.Step, "%s, as seen by %s", message, viewer.Socket.User.Username)
		}
	}

	assertStep("charge", "a turn waits for the player in the charge step")

	charged, err := player.Player.SpawnCard(turnStepCreatureUID, match.HAND)
	require.NoError(t, err)
	require.NoError(t, scn.ActionChargeMana(player, charged.ID))
	assertStep("charge", "charging mana does not end the charge step")

	_, err = player.Player.SpawnCard(turnStepCreatureUID, match.MANAZONE)
	require.NoError(t, err)
	creature, err := player.Player.SpawnCard(turnStepCreatureUID, match.HAND)
	require.NoError(t, err)
	require.NoError(t, scn.ActionPlayCard(player, creature.ID))
	require.NoError(t, scn.WaitForEventLoop())
	assertStep("main", "playing a card moves the turn to the main step")

	require.NoError(t, scn.ActionEndTurn(player))
	assertStep("charge", "the next turn starts in its charge step")

	require.NoError(t, scn.ActionEndTurn(opponent))
	assertStep("charge", "the turn comes back in its charge step")

	shields, err := opponent.Player.Container(match.SHIELDZONE)
	require.NoError(t, err)
	require.NotEmpty(t, shields)

	_, err = scn.ActionAttackPlayer(player, creature.ID)
	require.NoError(t, err)
	require.NoError(t, scn.ResolveAttack(player, shields[0].ID))
	require.NoError(t, scn.WaitForEventLoop())
	assertStep("attack", "attacking moves the turn to the attack step")
}

// Seamine is a vanilla 6 mana water blocker.
const turnStepBlockerUID = "446eaf96-36c8-4093-b4b2-e77e7afb6e3f"

func TestStateReportsTheAttackStepWhileAnAttackResolves(t *testing.T) {
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

	for _, viewer := range []*match.PlayerReference{attacker, defender} {
		state, err := scn.LatestState(viewer, 0)
		require.NoError(t, err)
		assert.Equal(t, "attack", state.Step, "an attack is in the attack step before it resolves, as seen by %s", viewer.Socket.User.Username)
	}

	// Declining to block lets the attack through to the shield.
	require.NoError(t, scn.CancelAction(defender))
	require.NoError(t, scn.WaitForEventLoop())

	state, err := scn.LatestState(attacker, 0)
	require.NoError(t, err)
	assert.Equal(t, "attack", state.Step, "the attack step outlasts the attack")
}

func TestStateReportsWhetherTheTurnPlayerChargedMana(t *testing.T) {
	scn := scenario.New(scenario.WithDeck(scenario.DeckEntry{UID: turnStepCreatureUID, Count: 40}))
	player := scn.Match.CurrentPlayer()
	opponent := scn.Match.PlayerRef(scn.Match.Opponent(player.Player))

	assertCharged := func(expected bool, message string) {
		t.Helper()
		for _, viewer := range []*match.PlayerReference{player, opponent} {
			state, err := scn.LatestState(viewer, 0)
			require.NoError(t, err)
			assert.Equal(t, expected, state.TurnManaCharged, "%s, as seen by %s", message, viewer.Socket.User.Username)
		}
	}

	assertCharged(false, "a turn starts without mana charged")

	charged, err := player.Player.SpawnCard(turnStepCreatureUID, match.HAND)
	require.NoError(t, err)
	require.NoError(t, scn.ActionChargeMana(player, charged.ID))
	assertCharged(true, "charging mana is seen by both players")

	require.NoError(t, scn.ActionEndTurn(player))
	assertCharged(false, "the next turn starts without mana charged")
}
