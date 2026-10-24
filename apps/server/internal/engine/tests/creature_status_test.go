package tests

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/cnd"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/server"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/tests/scenario"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// Golden Wing Striker: 3 mana nature creature with power attacker +2000.
	creatureStatusAttackerUID = "6663848d-035e-44b6-9d9f-7b236ea5bc43"
	// Seamine: 6 mana water creature with blocker.
	creatureStatusBlockerUID = "446eaf96-36c8-4093-b4b2-e77e7afb6e3f"
)

func summonForStatus(t *testing.T, scn *scenario.TestScenario, player *match.PlayerReference, uid string, mana int) *match.Card {
	t.Helper()

	for range mana {
		// Golden Wing Striker is nature and Seamine water, so give both.
		_, err := player.Player.SpawnCard(creatureStatusAttackerUID, match.MANAZONE)
		require.NoError(t, err)
		_, err = player.Player.SpawnCard(creatureStatusBlockerUID, match.MANAZONE)
		require.NoError(t, err)
	}
	card, err := player.Player.SpawnCard(uid, match.HAND)
	require.NoError(t, err)
	require.NoError(t, scn.ActionPlayCard(player, card.ID))
	require.NoError(t, scn.WaitForEventLoop())

	return card
}

func statusesOf(t *testing.T, scn *scenario.TestScenario, player *match.PlayerReference, cardID string) []server.CreatureStatus {
	t.Helper()

	state, err := scn.LatestState(player, 0)
	require.NoError(t, err)

	return state.Me.Statuses[cardID]
}

func TestStateReportsCreatureStatuses(t *testing.T) {
	scn := scenario.New(scenario.WithDeck(scenario.DeckEntry{UID: creatureStatusAttackerUID, Count: 40}))
	player := scn.Match.CurrentPlayer()
	opponent := scn.Match.PlayerRef(scn.Match.Opponent(player.Player))

	attacker := summonForStatus(t, scn, player, creatureStatusAttackerUID, 3)
	blocker := summonForStatus(t, scn, player, creatureStatusBlockerUID, 6)

	t.Run("a creature summoned this turn has summoning sickness", func(t *testing.T) {
		assert.Contains(t, statusesOf(t, scn, player, blocker.ID), server.CreatureStatus{ID: cnd.SummoningSickness})
	})

	require.NoError(t, scn.ActionEndTurn(player))
	require.NoError(t, scn.ActionEndTurn(opponent))

	t.Run("printed abilities are shown, and not as granted", func(t *testing.T) {
		assert.Equal(t, []server.CreatureStatus{{ID: cnd.PowerAttacker, Value: 2000}}, statusesOf(t, scn, player, attacker.ID))
		assert.Contains(t, statusesOf(t, scn, player, blocker.ID), server.CreatureStatus{ID: cnd.Blocker})
	})

	t.Run("a status another card gave is shown as granted", func(t *testing.T) {
		attacker.AddCondition(cnd.Slayer, nil, blocker.ID)
		attacker.AddCondition(cnd.CantBeBlocked, nil, attacker.ID)

		// Charging mana broadcasts the state.
		charged, err := player.Player.SpawnCard(creatureStatusAttackerUID, match.HAND)
		require.NoError(t, err)
		require.NoError(t, scn.ActionChargeMana(player, charged.ID))

		statuses := statusesOf(t, scn, player, attacker.ID)
		assert.Contains(t, statuses, server.CreatureStatus{ID: cnd.Slayer, Granted: true})
		assert.Contains(t, statuses, server.CreatureStatus{ID: cnd.CantBeBlocked})
	})

	t.Run("statuses hold steady while the turn changes", func(t *testing.T) {
		since, err := scn.MessageCount(player)
		require.NoError(t, err)
		require.NoError(t, scn.ActionEndTurn(player))

		states, err := scn.StatesSince(player, since)
		require.NoError(t, err)
		require.NotEmpty(t, states)
		for _, state := range states {
			assert.Contains(t, state.Me.Statuses[blocker.ID], server.CreatureStatus{ID: cnd.Blocker})
		}
	})
}

func TestStateReportsKeywordStatuses(t *testing.T) {
	scn := scenario.New(scenario.WithDeck(scenario.DeckEntry{UID: creatureStatusAttackerUID, Count: 40}))
	player := scn.Match.CurrentPlayer()
	opponent := scn.Match.PlayerRef(scn.Match.Opponent(player.Player))

	creature := summonForStatus(t, scn, player, creatureStatusAttackerUID, 3)
	require.NoError(t, scn.ActionEndTurn(player))
	require.NoError(t, scn.ActionEndTurn(opponent))

	creature.AddCondition(cnd.Survivor, true, creature.ID)
	creature.AddCondition(cnd.TreatedAsTapped, nil, creature.ID)
	creature.AddCondition(cnd.ShieldBreakModifier, 1, creature.ID)
	creature.AddCondition(cnd.ShieldBreakModifier, 2, "another-card")
	creature.AddCondition(cnd.Stealth, civ.Water, creature.ID)

	// Charging mana broadcasts the state.
	charged, err := player.Player.SpawnCard(creatureStatusAttackerUID, match.HAND)
	require.NoError(t, err)
	require.NoError(t, scn.ActionChargeMana(player, charged.ID))

	statuses := statusesOf(t, scn, player, creature.ID)

	t.Run("survivor and treated as tapped are shown", func(t *testing.T) {
		assert.Contains(t, statuses, server.CreatureStatus{ID: cnd.Survivor})
		assert.Contains(t, statuses, server.CreatureStatus{ID: cnd.TreatedAsTapped})
	})

	t.Run("extra shields to break add up across effects", func(t *testing.T) {
		assert.Contains(t, statuses, server.CreatureStatus{ID: cnd.ShieldBreakModifier, Value: 3, Granted: true})
	})

	t.Run("stealth is not shown while the opponent has no mana of its civilization", func(t *testing.T) {
		for _, status := range statuses {
			assert.NotEqual(t, cnd.Stealth, status.ID)
		}
	})

	t.Run("stealth is shown once it keeps the creature from being blocked", func(t *testing.T) {
		_, err := opponent.Player.SpawnCard(creatureStatusBlockerUID, match.MANAZONE)
		require.NoError(t, err)

		// Summoning broadcasts the state again.
		summonForStatus(t, scn, player, creatureStatusAttackerUID, 3)

		assert.Contains(t, statusesOf(t, scn, player, creature.ID), server.CreatureStatus{ID: cnd.Stealth})
	})
}
