package tests

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/server"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/tests/scenario"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Terror Pit is a 6 mana darkness spell and shield trigger: destroy one of
// your opponent's creatures.
const promptAboutTerrorPitUID = "5883180e-d88c-4f24-b17c-f5a837420147"

// Seamine is a vanilla 6 mana water blocker.
const promptAboutBlockerUID = "446eaf96-36c8-4093-b4b2-e77e7afb6e3f"

// assertAbout checks what a question says it is about: its kind, the card
// whose effect asks (nil for none told), and which offered cards are the
// opponent's.
func assertAbout(t *testing.T, about server.PromptAbout, kind match.PromptKind, source *match.Card, opponentCards []string, what string) {
	t.Helper()

	assert.Equal(t, string(kind), about.Kind, "%s: kind", what)
	if source == nil {
		assert.Nil(t, about.Source, "%s: no source told", what)
	} else if assert.NotNil(t, about.Source, "%s: source", what) {
		assert.Equal(t, server.PromptSource{CardID: source.ID, ImageID: source.ImageID, Name: source.Name}, *about.Source, "%s: source", what)
	}
	assert.ElementsMatch(t, opponentCards, about.OpponentCards, "%s: the opponent's cards", what)
}

func cardIDs(cards []server.CardState) []string {
	ids := make([]string, 0, len(cards))
	for _, card := range cards {
		ids = append(ids, card.CardID)
	}
	return ids
}

// summon plays a creature from the player's hand, paying with every card
// offered, and checks the question asked about the mana.
func summonCheckingMana(t *testing.T, scn *scenario.TestScenario, player *match.PlayerReference, creature *match.Card) {
	t.Helper()

	payment, err := scn.ActionPlayCardPrompt(player, creature.ID)
	require.NoError(t, err)
	assertAbout(t, payment.PromptAbout, match.PromptPayMana, creature, nil, "paying for a creature")
	require.NoError(t, scn.SubmitAction(player, cardIDs(payment.Cards)[:payment.MaxSelections]...))
	require.NoError(t, scn.WaitForEventLoop())
	require.Equal(t, match.BATTLEZONE, creature.Zone)
}

func TestQuestionsSayWhatTheyAreAbout(t *testing.T) {
	scn := scenario.New(scenario.WithDeck(scenario.DeckEntry{UID: brokenShieldsCreatureUID, Count: 40}))
	attacker := scn.Match.CurrentPlayer()
	defender := scn.Match.PlayerRef(scn.Match.Opponent(attacker.Player))

	// Two creatures, so Terror Pit has a choice to ask about.
	for range 4 {
		_, err := attacker.Player.SpawnCard(brokenShieldsCreatureUID, match.MANAZONE)
		require.NoError(t, err)
	}
	creature, err := attacker.Player.SpawnCard(brokenShieldsCreatureUID, match.HAND)
	require.NoError(t, err)
	summonCheckingMana(t, scn, attacker, creature)
	other, err := attacker.Player.SpawnCard(brokenShieldsCreatureUID, match.HAND)
	require.NoError(t, err)
	require.NoError(t, scn.ActionPlayCard(attacker, other.ID))
	require.NoError(t, scn.WaitForEventLoop())
	require.NoError(t, scn.ActionEndTurn(attacker))

	// The defender has a water blocker, paid with one water mana and five
	// darkness, and a Terror Pit in its shields for the shield trigger.
	_, err = defender.Player.SpawnCard(brokenShieldsCreatureUID, match.MANAZONE)
	require.NoError(t, err)
	for range 5 {
		_, err := defender.Player.SpawnCard(promptAboutTerrorPitUID, match.MANAZONE)
		require.NoError(t, err)
	}
	blocker, err := defender.Player.SpawnCard(promptAboutBlockerUID, match.HAND)
	require.NoError(t, err)
	require.NoError(t, scn.ActionPlayCard(defender, blocker.ID))
	require.NoError(t, scn.WaitForEventLoop())
	// Cards are given their shield trigger as each turn begins.
	trigger, err := defender.Player.SpawnCard(promptAboutTerrorPitUID, match.SHIELDZONE)
	require.NoError(t, err)
	require.NoError(t, scn.ActionEndTurn(defender))

	shields, err := defender.Player.Container(match.SHIELDZONE)
	require.NoError(t, err)
	shieldIDs := make([]string, 0, len(shields))
	for _, shield := range shields {
		shieldIDs = append(shieldIDs, shield.ID)
	}

	breaking, err := scn.ActionAttackPlayer(attacker, creature.ID)
	require.NoError(t, err)
	assertAbout(t, breaking.PromptAbout, match.PromptBreakShields, creature, shieldIDs, "choosing shields to break")

	blockAsked, err := scn.MessageCount(defender)
	require.NoError(t, err)
	require.NoError(t, scn.ResolveAttack(attacker, trigger.ID))
	block, err := scn.WaitForAction(defender, blockAsked)
	require.NoError(t, err)
	assertAbout(t, block.PromptAbout, match.PromptBlock, creature, nil, "blocking")
	assert.Equal(t, &server.PromptAttack{
		Attacker: server.PromptFighter{Name: creature.Name, Power: scn.Match.GetPower(creature, true)},
	}, block.Attack, "blocking an attack on the player names the attacker alone")

	triggerAsked, err := scn.MessageCount(defender)
	require.NoError(t, err)
	require.NoError(t, scn.CancelAction(defender))
	triggered, err := scn.WaitForAction(defender, triggerAsked)
	require.NoError(t, err)
	assertAbout(t, triggered.PromptAbout, match.PromptShieldTrigger, creature, nil, "a shield trigger")

	effectAsked, err := scn.MessageCount(defender)
	require.NoError(t, err)
	require.NoError(t, scn.SubmitAction(defender, trigger.ID))
	effect, err := scn.WaitForAction(defender, effectAsked)
	require.NoError(t, err)
	assertAbout(t, effect.PromptAbout, match.PromptEffect, trigger, []string{creature.ID, other.ID}, "a card's effect")

	require.NoError(t, scn.SubmitAction(defender, creature.ID))
	require.NoError(t, scn.WaitForEventLoop())
	assert.Equal(t, match.GRAVEYARD, creature.Zone, "Terror Pit destroyed the attacker")
}

func TestChoosingWhatToAttackSaysWhatItIsAbout(t *testing.T) {
	scn := scenario.New(scenario.WithDeck(scenario.DeckEntry{UID: brokenShieldsCreatureUID, Count: 40}))
	attacker := scn.Match.CurrentPlayer()
	defender := scn.Match.PlayerRef(scn.Match.Opponent(attacker.Player))

	for range 2 {
		_, err := attacker.Player.SpawnCard(brokenShieldsCreatureUID, match.MANAZONE)
		require.NoError(t, err)
	}
	creature, err := attacker.Player.SpawnCard(brokenShieldsCreatureUID, match.HAND)
	require.NoError(t, err)
	require.NoError(t, scn.ActionPlayCard(attacker, creature.ID))
	require.NoError(t, scn.WaitForEventLoop())
	require.NoError(t, scn.ActionEndTurn(attacker))

	for range 2 {
		_, err := defender.Player.SpawnCard(brokenShieldsCreatureUID, match.MANAZONE)
		require.NoError(t, err)
	}
	target, err := defender.Player.SpawnCard(brokenShieldsCreatureUID, match.HAND)
	require.NoError(t, err)
	require.NoError(t, scn.ActionPlayCard(defender, target.ID))
	require.NoError(t, scn.WaitForEventLoop())
	require.NoError(t, scn.ActionEndTurn(defender))
	defender.Player.TapCard(target)

	choosing, err := scn.ActionAttackCreaturePrompt(attacker, creature.ID)
	require.NoError(t, err)
	assertAbout(t, choosing.PromptAbout, match.PromptAttackTarget, creature, []string{target.ID}, "choosing what to attack")
	require.NoError(t, scn.CancelAction(attacker))
	require.NoError(t, scn.WaitForEventLoop())
}

func TestBlockQuestionNamesTheCreatureAttacked(t *testing.T) {
	scn := scenario.New(scenario.WithDeck(scenario.DeckEntry{UID: brokenShieldsCreatureUID, Count: 40}))
	attacker := scn.Match.CurrentPlayer()
	defender := scn.Match.PlayerRef(scn.Match.Opponent(attacker.Player))

	for range 4 {
		_, err := attacker.Player.SpawnCard(brokenShieldsCreatureUID, match.MANAZONE)
		require.NoError(t, err)
	}
	creature, err := attacker.Player.SpawnCard(brokenShieldsCreatureUID, match.HAND)
	require.NoError(t, err)
	require.NoError(t, scn.ActionPlayCard(attacker, creature.ID))
	require.NoError(t, scn.WaitForEventLoop())
	require.NoError(t, scn.ActionEndTurn(attacker))

	_, err = defender.Player.SpawnCard(brokenShieldsCreatureUID, match.MANAZONE)
	require.NoError(t, err)
	for range 5 {
		_, err := defender.Player.SpawnCard(promptAboutTerrorPitUID, match.MANAZONE)
		require.NoError(t, err)
	}
	blocker, err := defender.Player.SpawnCard(promptAboutBlockerUID, match.HAND)
	require.NoError(t, err)
	require.NoError(t, scn.ActionPlayCard(defender, blocker.ID))
	require.NoError(t, scn.WaitForEventLoop())
	target, err := defender.Player.SpawnCard(brokenShieldsCreatureUID, match.HAND)
	require.NoError(t, err)
	_, err = defender.Player.MoveCard(target.ID, match.HAND, match.BATTLEZONE, target.ID)
	require.NoError(t, err)
	require.NoError(t, scn.ActionEndTurn(defender))
	defender.Player.TapCard(target)

	_, err = scn.ActionAttackCreaturePrompt(attacker, creature.ID)
	require.NoError(t, err)
	blockAsked, err := scn.MessageCount(defender)
	require.NoError(t, err)
	require.NoError(t, scn.SubmitAction(attacker, target.ID))
	block, err := scn.WaitForAction(defender, blockAsked)
	require.NoError(t, err)

	assert.Equal(t, string(match.PromptBlock), block.Kind)
	assert.Equal(t, &server.PromptAttack{
		Attacker: server.PromptFighter{Name: creature.Name, Power: scn.Match.GetPower(creature, true)},
		Target:   &server.PromptFighter{Name: target.Name, Power: scn.Match.GetPower(target, false)},
	}, block.Attack)

	require.NoError(t, scn.CancelAction(defender))
	require.NoError(t, scn.WaitForEventLoop())
}
