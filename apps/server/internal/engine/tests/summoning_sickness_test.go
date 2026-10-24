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

// Aqua Vehicle is a vanilla 2 mana water creature.
const summoningSicknessCreatureUID = "9781089f-1aa9-4a75-b106-35e9d431e31d"

func summonSick(t *testing.T, scn *scenario.TestScenario, player *match.PlayerReference) *match.Card {
	t.Helper()

	for range 2 {
		_, err := player.Player.SpawnCard(summoningSicknessCreatureUID, match.MANAZONE)
		require.NoError(t, err)
	}
	creature, err := player.Player.SpawnCard(summoningSicknessCreatureUID, match.HAND)
	require.NoError(t, err)
	require.NoError(t, scn.ActionPlayCard(player, creature.ID))
	require.NoError(t, scn.WaitForEventLoop())

	return creature
}

// summonSickIn reports whether a card is flagged as summoning sick in a zone.
func summonSickIn(t *testing.T, cards []server.CardState, cardID string) bool {
	t.Helper()

	for _, card := range cards {
		if card.CardID == cardID {
			return match.CardFlags(card.Flags)&match.SummonSickFlag != 0
		}
	}

	t.Fatalf("card %s is not in the zone", cardID)
	return false
}

func TestStateFlagsSummoningSickness(t *testing.T) {
	scn := scenario.New(scenario.WithDeck(scenario.DeckEntry{UID: summoningSicknessCreatureUID, Count: 40}))
	player := scn.Match.CurrentPlayer()
	opponent := scn.Match.PlayerRef(scn.Match.Opponent(player.Player))

	fast := summonSick(t, scn, player)

	t.Run("a creature summoned this turn is summoning sick, for both players", func(t *testing.T) {
		mine, err := scn.LatestState(player, 0)
		require.NoError(t, err)
		theirs, err := scn.LatestState(opponent, 0)
		require.NoError(t, err)

		assert.True(t, summonSickIn(t, mine.Me.Battlezone, fast.ID))
		assert.True(t, summonSickIn(t, theirs.Opponent.Battlezone, fast.ID))
	})

	t.Run("a speed attacker is not summoning sick", func(t *testing.T) {
		fast.AddCondition(cnd.SpeedAttacker, nil, "test-effect")
		slow := summonSick(t, scn, player)

		state, err := scn.LatestState(player, 0)
		require.NoError(t, err)
		assert.False(t, summonSickIn(t, state.Me.Battlezone, fast.ID))
		assert.True(t, summonSickIn(t, state.Me.Battlezone, slow.ID))
	})

	t.Run("the sickness is gone once the turn ends", func(t *testing.T) {
		require.NoError(t, scn.ActionEndTurn(player))

		state, err := scn.LatestState(player, 0)
		require.NoError(t, err)
		for _, card := range state.Me.Battlezone {
			assert.False(t, summonSickIn(t, state.Me.Battlezone, card.CardID), "%s is still sick", card.Name)
		}
	})
}
