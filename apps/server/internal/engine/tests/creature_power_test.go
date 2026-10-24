package tests

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/cnd"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/server"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/tests/scenario"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Golden Wing Striker is a 3 mana nature creature with 2000 power and power
// attacker +2000. A deck of nothing else has no shield triggers or blockers.
const creaturePowerUID = "6663848d-035e-44b6-9d9f-7b236ea5bc43"

func TestStateReportsCreaturePower(t *testing.T) {
	scn := scenario.New(scenario.WithDeck(scenario.DeckEntry{UID: creaturePowerUID, Count: 40}))
	player := scn.Match.CurrentPlayer()
	opponent := scn.Match.PlayerRef(scn.Match.Opponent(player.Player))

	for range 3 {
		_, err := player.Player.SpawnCard(creaturePowerUID, match.MANAZONE)
		require.NoError(t, err)
	}
	creature, err := player.Player.SpawnCard(creaturePowerUID, match.HAND)
	require.NoError(t, err)
	require.NoError(t, scn.ActionPlayCard(player, creature.ID))
	require.NoError(t, scn.WaitForEventLoop())

	latestPower := func(viewer *match.PlayerReference, owner func(server.MatchState) server.PlayerState) server.CreaturePower {
		t.Helper()
		state, err := scn.LatestState(viewer, 0)
		require.NoError(t, err)
		power, ok := owner(*state).Powers[creature.ID]
		require.True(t, ok, "the creature's power should be in the state")
		return power
	}
	mine := func(state server.MatchState) server.PlayerState { return state.Me }
	theirs := func(state server.MatchState) server.PlayerState { return state.Opponent }

	t.Run("a creature just summoned has its printed power", func(t *testing.T) {
		assert.Equal(t, server.CreaturePower{Printed: 2000, Current: 2000, Attacking: 2000}, latestPower(player, mine))
		assert.Equal(t, server.CreaturePower{Printed: 2000, Current: 2000, Attacking: 2000}, latestPower(opponent, theirs), "the opponent sees the same power")
	})

	t.Run("power attacker raises only the attacking power", func(t *testing.T) {
		require.NoError(t, scn.ActionEndTurn(player))
		require.NoError(t, scn.ActionEndTurn(opponent))

		assert.Equal(t, server.CreaturePower{Printed: 2000, Current: 2000, Attacking: 4000}, latestPower(player, mine))
	})

	t.Run("an effect raising power is reported", func(t *testing.T) {
		creature.AddCondition(cnd.PowerAmplifier, 3000, "test-effect")

		charged, err := player.Player.SpawnCard(creaturePowerUID, match.HAND)
		require.NoError(t, err)
		require.NoError(t, scn.ActionChargeMana(player, charged.ID))

		assert.Equal(t, server.CreaturePower{Printed: 2000, Current: 5000, Attacking: 7000}, latestPower(player, mine))
	})

	t.Run("cards outside the battle zone have no power reported", func(t *testing.T) {
		state, err := scn.LatestState(player, 0)
		require.NoError(t, err)

		hand, err := player.Player.Container(match.HAND)
		require.NoError(t, err)
		for _, card := range hand {
			assert.NotContains(t, state.Me.Powers, card.ID)
		}
	})
}

// Smash Horn Q is a 3 mana nature survivor with 2000 power. Every survivor
// gives every survivor +1000, itself included, while it has the survivor
// condition, which is cleared at the end of each turn and rebuilt at untap.
const creaturePowerSurvivorUID = "5ff3c63c-30df-4f5c-acce-796f5b6c2dac"

func TestCreaturePowerHoldsSteadyAcrossTheTurnChange(t *testing.T) {
	scn := scenario.New(scenario.WithDeck(scenario.DeckEntry{UID: creaturePowerUID, Count: 40}))
	player := scn.Match.CurrentPlayer()
	opponent := scn.Match.PlayerRef(scn.Match.Opponent(player.Player))

	for range 3 {
		_, err := player.Player.SpawnCard(creaturePowerUID, match.MANAZONE)
		require.NoError(t, err)
	}
	survivor, err := player.Player.SpawnCard(creaturePowerSurvivorUID, match.HAND)
	require.NoError(t, err)
	require.NoError(t, scn.ActionPlayCard(player, survivor.ID))
	require.NoError(t, scn.WaitForEventLoop())

	// The opponent's untap step gives it the survivor condition.
	require.NoError(t, scn.ActionEndTurn(player))
	state, err := scn.LatestState(player, 0)
	require.NoError(t, err)
	require.Equal(t, 3000, state.Me.Powers[survivor.ID].Current)

	since, err := scn.MessageCount(player)
	require.NoError(t, err)
	require.NoError(t, scn.ActionEndTurn(opponent))

	for _, current := range currentPowersSince(t, scn, player, since, survivor.ID) {
		assert.Equal(t, 3000, current, "the power must not dip while the turn changes")
	}
}

// currentPowersSince lists the current power of a creature in every state sent
// to the player since position since.
func currentPowersSince(t *testing.T, scn *scenario.TestScenario, player *match.PlayerReference, since int, cardID string) []int {
	t.Helper()

	states, err := scn.StatesSince(player, since)
	require.NoError(t, err)

	powers := make([]int, 0, len(states))
	for _, state := range states {
		if power, ok := state.Me.Powers[cardID]; ok {
			powers = append(powers, power.Current)
		}
	}

	require.NotEmpty(t, powers)
	return powers
}
