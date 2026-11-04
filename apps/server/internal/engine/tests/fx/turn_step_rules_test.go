package fx

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/cnd"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/tests/scenario"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// Immortal Baron, Vorg is a vanilla 2000 creature and the whole test deck, so
	// attacks resolve without blockers or shield triggers interfering.
	attackStepAttackerUID = "af3bc221-1cc2-4f58-83ea-2673ac2c66c5"
	// Popple, Flowerpetal Dancer's tap ability puts the top card of its
	// controller's deck into their manazone, which resolves without a prompt of
	// its own. It does not count as charging mana.
	attackStepTapAbilityUID = "bb3a50dd-049d-488d-b89c-779dcf29b82e"
	attackStepSetupSrc      = "attack_step_test_setup"
)

// A turn waits in each of the three steps a player acts in, and moves on only
// when the player says so: mana is charged in the charge step, cards are played
// in the main step, and attacks and tap abilities are made in the attack step.
// Each step refuses the others' actions, and the turn never goes back.
func TestTurnStepsMoveOnOnlyWhenThePlayerSays(t *testing.T) {

	t.Run("a turn waits in the charge step", func(t *testing.T) {
		scn, player, opponent := attackStepScenario(t)
		passTurn(t, scn, player, opponent)

		assert.IsType(t, &match.ChargeStep{}, scn.Match.Step)

		spare := firstHandCard(t, player)
		require.NoError(t, scn.ActionChargeMana(player, spare.ID))
		assert.Equal(t, match.MANAZONE, spare.Zone)
		assert.IsType(t, &match.ChargeStep{}, scn.Match.Step, "charging mana does not move the turn on")
	})

	t.Run("nothing is played in the charge step", func(t *testing.T) {
		scn, player, opponent := attackStepScenario(t)
		passTurn(t, scn, player, opponent)
		creature := spawnInHand(t, player)

		warnings := sendRaw(t, scn, player, fmt.Sprintf(`{"header":"add_to_playzone","virtualId":%q}`, creature.ID))

		assert.Equal(t, match.HAND, creature.Zone)
		assert.True(t, containsWarning(warnings, "summon phase"), "got %v", warnings)
		assert.IsType(t, &match.ChargeStep{}, scn.Match.Step)
	})

	t.Run("nothing attacks before the attack step", func(t *testing.T) {
		scn, player, opponent := attackStepScenario(t)
		attacker := putInBattleZone(t, scn, player, attackStepAttackerUID)
		passTurn(t, scn, player, opponent)

		for _, step := range []string{"", match.GoToMainStep} {
			if step != "" {
				require.NoError(t, scn.GoToStep(player, step))
			}

			warnings := sendRaw(t, scn, player, fmt.Sprintf(`{"header":"attack_player","virtualId":%q}`, attacker.ID))

			assert.False(t, attacker.Tapped)
			assert.True(t, containsWarning(warnings, "battle phase"), "got %v", warnings)
			assert.False(t, inAttackStep(scn))
		}
	})

	t.Run("no tap ability is used before the attack step", func(t *testing.T) {
		scn, player, opponent := attackStepScenario(t)
		popple := putInBattleZone(t, scn, player, attackStepTapAbilityUID)
		passTurn(t, scn, player, opponent)
		require.True(t, popple.HasCondition(cnd.TapAbility))

		warnings := sendRaw(t, scn, player, fmt.Sprintf(`{"header":"tap_ability","virtualId":%q}`, popple.ID))

		assert.False(t, popple.Tapped)
		assert.True(t, containsWarning(warnings, "battle phase"), "got %v", warnings)
	})

	t.Run("the main step ends charging mana", func(t *testing.T) {
		scn, player, opponent := attackStepScenario(t)
		passTurn(t, scn, player, opponent)

		require.NoError(t, scn.GoToStep(player, match.GoToMainStep))
		assert.IsType(t, &match.MainStep{}, scn.Match.Step)
		assert.False(t, player.Player.CanChargeMana)

		spare := firstHandCard(t, player)
		warnings := sendRaw(t, scn, player, fmt.Sprintf(`{"header":"add_to_manazone","virtualId":%q}`, spare.ID))
		assert.Equal(t, match.HAND, spare.Zone)
		assert.True(t, containsWarning(warnings, "mana phase"), "got %v", warnings)
	})

	t.Run("cards are played in the main step", func(t *testing.T) {
		scn, player, opponent := attackStepScenario(t)
		passTurn(t, scn, player, opponent)
		creature := spawnInHand(t, player)

		require.NoError(t, scn.ActionPlayCard(player, creature.ID))
		assert.Equal(t, match.BATTLEZONE, creature.Zone)
		assert.IsType(t, &match.MainStep{}, scn.Match.Step, "playing a card does not move the turn on")
	})

	t.Run("the attack step ends playing cards", func(t *testing.T) {
		scn, player, opponent := attackStepScenario(t)
		passTurn(t, scn, player, opponent)
		creature := spawnInHand(t, player)

		require.NoError(t, scn.GoToStep(player, match.GoToAttackStep))
		assert.True(t, inAttackStep(scn))

		warnings := sendRaw(t, scn, player, fmt.Sprintf(`{"header":"add_to_playzone","virtualId":%q}`, creature.ID))
		assert.Equal(t, match.HAND, creature.Zone)
		assert.True(t, containsWarning(warnings, "can't summon"), "got %v", warnings)
	})

	t.Run("the attack step can be reached straight from the charge step", func(t *testing.T) {
		scn, player, opponent := attackStepScenario(t)
		passTurn(t, scn, player, opponent)

		require.NoError(t, scn.GoToStep(player, match.GoToAttackStep))
		assert.True(t, inAttackStep(scn))
		assert.False(t, player.Player.CanChargeMana)
	})

	t.Run("the turn never goes back", func(t *testing.T) {
		scn, player, opponent := attackStepScenario(t)
		passTurn(t, scn, player, opponent)
		require.NoError(t, scn.GoToStep(player, match.GoToAttackStep))

		start, err := scn.MessageCount(player)
		require.NoError(t, err)
		require.NoError(t, scn.GoToStep(player, match.GoToMainStep))

		assert.True(t, inAttackStep(scn))
		warnings, err := scn.Warnings(player, start)
		require.NoError(t, err)
		assert.True(t, containsWarning(warnings, "battle phase"), "got %v", warnings)
	})

	t.Run("asking for the step the turn is in changes nothing", func(t *testing.T) {
		scn, player, opponent := attackStepScenario(t)
		passTurn(t, scn, player, opponent)
		require.NoError(t, scn.GoToStep(player, match.GoToMainStep))

		start, err := scn.MessageCount(player)
		require.NoError(t, err)
		require.NoError(t, scn.GoToStep(player, match.GoToMainStep))

		assert.IsType(t, &match.MainStep{}, scn.Match.Step)
		headers, err := scn.MessageHeaders(player, start)
		require.NoError(t, err)
		assert.Empty(t, headers, "nothing changed, so nothing is sent")
	})

	t.Run("only the player whose turn it is moves it on", func(t *testing.T) {
		scn, player, opponent := attackStepScenario(t)
		passTurn(t, scn, player, opponent)

		require.NoError(t, scn.GoToStep(opponent, match.GoToAttackStep))
		assert.IsType(t, &match.ChargeStep{}, scn.Match.Step)
	})

	t.Run("a turn can end from the charge step", func(t *testing.T) {
		scn, player, opponent := attackStepScenario(t)
		passTurn(t, scn, player, opponent)

		require.NoError(t, scn.ActionEndTurn(player))
		assert.Equal(t, opponent.Player.Turn, scn.Match.Turn)
		assert.IsType(t, &match.ChargeStep{}, scn.Match.Step, "the next turn starts in the charge step")
	})
}

// Attacks only tap their creature and stay in the attack step once confirmed,
// and every step on the way to a confirmed attack is cancellable: shield
// selection and attack target selection. Backing out of either has to leave the
// creature able to attack.
//
// Shield selection is the case worth guarding hardest, because it is cancelled
// on a nested context of its own. Cancellation does not propagate to a parent
// context, so the outer AttackPlayer context still looks perfectly healthy after
// the player has walked away, and anything that reads it to decide whether the
// attack happened gets the answer wrong.
func TestAttacksAreOnlyMadeOnceConfirmed(t *testing.T) {

	t.Run("cancelling shield selection leaves the creature able to attack again", func(t *testing.T) {
		scn, player, opponent := attackStepScenario(t)
		attacker := putInBattleZone(t, scn, player, attackStepAttackerUID)
		passTurn(t, scn, player, opponent)

		_, err := scn.ActionAttackPlayer(player, attacker.ID)
		require.NoError(t, err)
		require.NoError(t, scn.CancelAction(player))
		require.NoError(t, scn.WaitForEventLoop())

		assert.False(t, attacker.Tapped, "an abandoned attack does not tap the attacker")

		shields, err := opponent.Player.Container(match.SHIELDZONE)
		require.NoError(t, err)
		shieldCount := len(shields)

		_, err = scn.ActionAttackPlayer(player, attacker.ID)
		require.NoError(t, err, "the same creature should be able to declare a second attack")
		require.NoError(t, scn.ResolveAttack(player, shields[0].ID))
		require.NoError(t, scn.WaitForEventLoop())

		shieldsAfter, err := opponent.Player.Container(match.SHIELDZONE)
		require.NoError(t, err)
		assert.Len(t, shieldsAfter, shieldCount-1, "the second attack should break a shield")
		assert.True(t, attacker.Tapped)
		assert.True(t, inAttackStep(scn))
	})

	t.Run("cancelling attack target selection leaves the turn untouched", func(t *testing.T) {
		scn, player, opponent := attackStepScenario(t)
		attacker := putInBattleZone(t, scn, player, attackStepAttackerUID)
		defender := putInBattleZone(t, scn, opponent, attackStepAttackerUID)
		passTurn(t, scn, player, opponent)

		// Only a tapped creature can be attacked, and this one never acted.
		defender.Tapped = true

		_, err := scn.ActionAttackCreaturePrompt(player, attacker.ID)
		require.NoError(t, err)
		require.NoError(t, scn.CancelAction(player))
		require.NoError(t, scn.WaitForEventLoop())

		assert.False(t, attacker.Tapped)
		assert.Equal(t, match.BATTLEZONE, defender.Zone, "no battle should have happened")
		assert.True(t, inAttackStep(scn))
	})

	t.Run("confirming an attack on a creature fights it", func(t *testing.T) {
		scn, player, opponent := attackStepScenario(t)
		attacker := putInBattleZone(t, scn, player, attackStepAttackerUID)
		defender := putInBattleZone(t, scn, opponent, attackStepAttackerUID)
		passTurn(t, scn, player, opponent)

		defender.Tapped = true

		require.NoError(t, scn.ActionAttackCreature(player, attacker.ID, defender.ID))
		require.NoError(t, scn.WaitForEventLoop())

		assert.True(t, inAttackStep(scn))
		assert.NotEqual(t, match.BATTLEZONE, defender.Zone, "two 2000 creatures destroy each other")
	})

	t.Run("using a tap ability resolves it", func(t *testing.T) {
		scn, player, opponent := attackStepScenario(t)
		popple := putInBattleZone(t, scn, player, attackStepTapAbilityUID)

		// A creature conjured straight into play has no conditions until an untap
		// step runs, so the tap ability itself only exists after passing a turn.
		passTurn(t, scn, player, opponent)
		require.True(t, popple.HasCondition(cnd.TapAbility))

		require.NoError(t, scn.ActionUseTapAbility(player, popple.ID))
		require.NoError(t, scn.WaitForEventLoop())

		assert.True(t, popple.Tapped)
		assert.True(t, inAttackStep(scn))
	})

	t.Run("a rejected attack leaves the creature untapped", func(t *testing.T) {
		scn, player, opponent := attackStepScenario(t)
		attacker := putInBattleZone(t, scn, player, attackStepAttackerUID)
		passTurn(t, scn, player, opponent)

		attacker.Tapped = true

		_, err := scn.ActionAttackPlayer(player, attacker.ID)
		require.Error(t, err, "a tapped creature cannot attack")
	})

	t.Run("the flags are cleared again on the next turn", func(t *testing.T) {
		scn, player, opponent := attackStepScenario(t)
		attacker := putInBattleZone(t, scn, player, attackStepAttackerUID)
		passTurn(t, scn, player, opponent)

		shields, err := opponent.Player.Container(match.SHIELDZONE)
		require.NoError(t, err)

		_, err = scn.ActionAttackPlayer(player, attacker.ID)
		require.NoError(t, err)
		require.NoError(t, scn.ResolveAttack(player, shields[0].ID))
		require.NoError(t, scn.WaitForEventLoop())
		require.True(t, inAttackStep(scn))

		passTurn(t, scn, player, opponent)

		assert.False(t, inAttackStep(scn), "a new turn starts outside the attack step")
		assert.True(t, player.Player.CanChargeMana)
		assert.False(t, player.Player.HasChargedMana)

		spare := firstHandCard(t, player)
		require.NoError(t, scn.ActionChargeMana(player, spare.ID))
		assert.Equal(t, match.MANAZONE, spare.Zone)
	})
}

func attackStepScenario(t *testing.T) (*scenario.TestScenario, *match.PlayerReference, *match.PlayerReference) {
	t.Helper()

	scn := scenario.New()
	player := scn.Match.CurrentPlayer()
	opponent := scn.Match.PlayerRef(scn.Match.Opponent(player.Player))

	return scn, player, opponent
}

// inAttackStep reads the step the same way Match.BroadcastState does, so the
// assertions do not depend on which non-attack step the match happens to be in.
func inAttackStep(scn *scenario.TestScenario) bool {
	_, ok := scn.Match.Step.(*match.AttackStep)
	return ok
}

func firstHandCard(t *testing.T, player *match.PlayerReference) *match.Card {
	t.Helper()

	hand, err := player.Player.Container(match.HAND)
	require.NoError(t, err)
	require.NotEmpty(t, hand, "the player needs a card in hand to charge mana with")

	return hand[0]
}

// spawnInHand puts a creature the player can afford into their hand, with the
// mana to pay for it.
func spawnInHand(t *testing.T, player *match.PlayerReference) *match.Card {
	t.Helper()

	for range 2 {
		_, err := player.Player.SpawnCard(attackStepAttackerUID, match.MANAZONE)
		require.NoError(t, err)
	}
	creature, err := player.Player.SpawnCard(attackStepAttackerUID, match.HAND)
	require.NoError(t, err)

	return creature
}

// sendRaw sends a message as the duel board would, bypassing the scenario's
// helpers, which move the turn to the step their action needs, and returns the
// warnings the player was sent once the match has dealt with it.
func sendRaw(t *testing.T, scn *scenario.TestScenario, player *match.PlayerReference, message string) []string {
	t.Helper()

	start, err := scn.MessageCount(player)
	require.NoError(t, err)

	scn.Match.Parse(player.Socket(), []byte(message))
	require.NoError(t, scn.WaitForEventLoop())

	warnings, err := scn.Warnings(player, start)
	require.NoError(t, err)

	return warnings
}

func containsWarning(warnings []string, substring string) bool {
	for _, warning := range warnings {
		if strings.Contains(warning, substring) {
			return true
		}
	}

	return false
}
