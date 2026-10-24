package tests

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/tests/scenario"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// Aqua Vehicle is a vanilla 2 mana water creature with 1000 power.
	activatedCreatureUID = "9781089f-1aa9-4a75-b106-35e9d431e31d"
	// Seamine is a vanilla 6 mana water blocker.
	activatedBlockerUID = "446eaf96-36c8-4093-b4b2-e77e7afb6e3f"
	// Saucer-Head Shark is a 5 mana water creature that, when summoned, returns
	// every creature with 2000 power or less to its owner's hand.
	activatedBounceUID = "70e6cc2c-c63d-4dd9-9b6e-0713fed174bb"
)

// activationsSince counts how often each card was reported as acting in the
// states sent to the player since position since.
func activationsSince(t *testing.T, scn *scenario.TestScenario, player *match.PlayerReference, since int) map[string]int {
	t.Helper()

	states, err := scn.StatesSince(player, since)
	require.NoError(t, err)

	counts := make(map[string]int)
	for _, state := range states {
		require.NotNil(t, state.ActivatedCards, "an empty list rather than null")
		for _, id := range state.ActivatedCards {
			counts[id]++
		}
	}

	return counts
}

func summonForTest(t *testing.T, scn *scenario.TestScenario, player *match.PlayerReference, uid string, mana int) *match.Card {
	t.Helper()

	for range mana {
		_, err := player.Player.SpawnCard(activatedCreatureUID, match.MANAZONE)
		require.NoError(t, err)
	}
	card, err := player.Player.SpawnCard(uid, match.HAND)
	require.NoError(t, err)
	require.NoError(t, scn.ActionPlayCard(player, card.ID))
	require.NoError(t, scn.WaitForEventLoop())

	return card
}

func TestStateReportsActingCards(t *testing.T) {
	t.Run("a summoned creature acts, while charging mana and drawing do not", func(t *testing.T) {
		scn := scenario.New(scenario.WithDeck(scenario.DeckEntry{UID: activatedCreatureUID, Count: 40}))
		player := scn.Match.CurrentPlayer()
		opponent := scn.Match.PlayerRef(scn.Match.Opponent(player.Player))

		chargeSince, err := scn.MessageCount(player)
		require.NoError(t, err)
		charged, err := player.Player.SpawnCard(activatedCreatureUID, match.HAND)
		require.NoError(t, err)
		require.NoError(t, scn.ActionChargeMana(player, charged.ID))
		assert.Empty(t, activationsSince(t, scn, player, chargeSince), "charging mana is not an effect")

		summonSince, err := scn.MessageCount(player)
		require.NoError(t, err)
		creature := summonForTest(t, scn, player, activatedCreatureUID, 2)
		assert.Equal(t, map[string]int{creature.ID: 1}, activationsSince(t, scn, player, summonSince))

		opponentSince, err := scn.MessageCount(opponent)
		require.NoError(t, err)
		require.NoError(t, scn.ActionEndTurn(player))
		assert.Empty(t, activationsSince(t, scn, opponent, opponentSince), "drawing is not an effect")
	})

	t.Run("an attacker acts once, and so does a blocker", func(t *testing.T) {
		scn := scenario.New(scenario.WithDeck(scenario.DeckEntry{UID: activatedCreatureUID, Count: 40}))
		attacker := scn.Match.CurrentPlayer()
		defender := scn.Match.PlayerRef(scn.Match.Opponent(attacker.Player))

		creature := summonForTest(t, scn, attacker, activatedCreatureUID, 2)
		require.NoError(t, scn.ActionEndTurn(attacker))
		blocker := summonForTest(t, scn, defender, activatedBlockerUID, 6)
		require.NoError(t, scn.ActionEndTurn(defender))

		shields, err := defender.Player.Container(match.SHIELDZONE)
		require.NoError(t, err)

		since, err := scn.MessageCount(attacker)
		require.NoError(t, err)
		blockStart, err := scn.MessageCount(defender)
		require.NoError(t, err)
		_, err = scn.ActionAttackPlayer(attacker, creature.ID)
		require.NoError(t, err)
		require.NoError(t, scn.ResolveAttack(attacker, shields[0].ID))
		_, err = scn.WaitForAction(defender, blockStart)
		require.NoError(t, err, "the blocker should have been offered")
		require.NoError(t, scn.SubmitAction(defender, blocker.ID))
		require.NoError(t, scn.WaitForEventLoop())

		activations := activationsSince(t, scn, attacker, since)
		assert.Equal(t, 1, activations[creature.ID], "the attacker is reported once")
		assert.Equal(t, 1, activations[blocker.ID], "the blocker is reported")
	})

	t.Run("a creature whose effect moves other creatures acts once", func(t *testing.T) {
		scn := scenario.New(scenario.WithDeck(scenario.DeckEntry{UID: activatedCreatureUID, Count: 40}))
		player := scn.Match.CurrentPlayer()

		small := summonForTest(t, scn, player, activatedCreatureUID, 2)
		require.Equal(t, match.BATTLEZONE, small.Zone)

		since, err := scn.MessageCount(player)
		require.NoError(t, err)
		shark := summonForTest(t, scn, player, activatedBounceUID, 5)

		require.Equal(t, match.HAND, small.Zone, "the shark returns the small creature")
		assert.Equal(t, map[string]int{shark.ID: 1}, activationsSince(t, scn, player, since), "summoned and bouncing, but reported once")
	})

	t.Run("only a source in a zone both players see is reported", func(t *testing.T) {
		scn := scenario.New(scenario.WithDeck(scenario.DeckEntry{UID: activatedCreatureUID, Count: 40}))
		player := scn.Match.CurrentPlayer()
		opponent := scn.Match.PlayerRef(scn.Match.Opponent(player.Player))

		inPlay := summonForTest(t, scn, player, activatedCreatureUID, 2)
		inHand, err := player.Player.SpawnCard(activatedCreatureUID, match.HAND)
		require.NoError(t, err)
		target, err := opponent.Player.SpawnCard(activatedCreatureUID, match.MANAZONE)
		require.NoError(t, err)
		other, err := opponent.Player.SpawnCard(activatedCreatureUID, match.MANAZONE)
		require.NoError(t, err)

		// Moves made the way card effects make them, naming their cause.
		_, err = opponent.Player.MoveCard(target.ID, match.MANAZONE, match.GRAVEYARD, inHand.ID)
		require.NoError(t, err)
		_, err = opponent.Player.MoveCard(other.ID, match.MANAZONE, match.GRAVEYARD, other.ID)
		require.NoError(t, err)
		_, err = opponent.Player.MoveCard(other.ID, match.GRAVEYARD, match.MANAZONE, "test-effect")
		require.NoError(t, err)
		_, err = opponent.Player.MoveCard(other.ID, match.MANAZONE, match.GRAVEYARD, inPlay.ID)
		require.NoError(t, err)

		since, err := scn.MessageCount(player)
		require.NoError(t, err)
		require.NoError(t, scn.ActionEndTurn(player))

		var reported []string
		states, err := scn.StatesSince(player, since)
		require.NoError(t, err)
		for _, state := range states {
			reported = append(reported, state.ActivatedCards...)
		}

		assert.Equal(t, []string{inPlay.ID}, reported, "a hidden card, the moved card itself and a cause that is no card are not reported")
	})
}
