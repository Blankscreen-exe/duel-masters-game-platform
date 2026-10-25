package tests

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/server"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/tests/scenario"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Every question the engine asks a player is numbered, and an answer only
// settles the question it names. These tests play Aqua Vehicle, a vanilla
// 2 mana creature, whose mana payment is a question the player may cancel.

const promptTestTimeout = 5 * time.Second

// promptScenario is a match with Aqua Vehicles in the decks, and its current
// player.
func promptScenario(t *testing.T) (*scenario.TestScenario, *match.PlayerReference) {
	t.Helper()

	scn := scenario.New(scenario.WithDeck(scenario.DeckEntry{UID: summoningSicknessCreatureUID, Count: 40}))

	return scn, scn.Match.CurrentPlayer()
}

// manaForOne puts the two mana cards that pay for one Aqua Vehicle in the
// player's mana zone.
func manaForOne(t *testing.T, player *match.PlayerReference) []string {
	t.Helper()

	mana := make([]string, 0, 2)
	for range 2 {
		card, err := player.Player.SpawnCard(summoningSicknessCreatureUID, match.MANAZONE)
		require.NoError(t, err)
		mana = append(mana, card.ID)
	}

	return mana
}

// creatureInHand puts an Aqua Vehicle in the player's hand.
func creatureInHand(t *testing.T, player *match.PlayerReference) *match.Card {
	t.Helper()

	creature, err := player.Player.SpawnCard(summoningSicknessCreatureUID, match.HAND)
	require.NoError(t, err)

	return creature
}

// assertPlayed answers the open mana question with mana and checks the
// creature was played, rather than its play cancelled by an earlier answer.
func assertPlayed(t *testing.T, scn *scenario.TestScenario, player *match.PlayerReference, creature *match.Card, mana []string) {
	t.Helper()

	require.NoError(t, scn.SubmitAction(player, mana...))
	require.NoError(t, scn.WaitForEventLoop())
	assert.Equal(t, match.BATTLEZONE, creature.Zone, "the question was answered by its own answer")
}

func TestQuestionsAreNumbered(t *testing.T) {
	scn, player := promptScenario(t)
	manaForOne(t, player)

	first, err := scn.ActionPlayCardPrompt(player, creatureInHand(t, player).ID)
	require.NoError(t, err)
	require.NoError(t, scn.CancelAction(player))
	require.NoError(t, scn.WaitForEventLoop())

	second, err := scn.ActionPlayCardPrompt(player, creatureInHand(t, player).ID)
	require.NoError(t, err)
	require.NoError(t, scn.CancelAction(player))
	require.NoError(t, scn.WaitForEventLoop())

	assert.Positive(t, first.Prompt.ID)
	assert.Greater(t, second.Prompt.ID, first.Prompt.ID)
}

func TestAnAnswerSentWithNoQuestionOpenIsDropped(t *testing.T) {
	scn, player := promptScenario(t)
	mana := manaForOne(t, player)

	// Before the fix, this cancel waited for the next question and cancelled
	// it, whatever it was. It names numbers the next questions will take, so
	// only being sent while nothing was open keeps it out.
	for prompt := range 3 {
		require.NoError(t, scn.SendAnswer(player, prompt, true, 0))
	}

	creature := creatureInHand(t, player)
	_, err := scn.ActionPlayCardPrompt(player, creature.ID)
	require.NoError(t, err)

	assertPlayed(t, scn, player, creature, mana)
}

func TestAnAnswerToAnEarlierQuestionIsDropped(t *testing.T) {
	scn, player := promptScenario(t)
	mana := manaForOne(t, player)

	earlier, err := scn.ActionPlayCardPrompt(player, creatureInHand(t, player).ID)
	require.NoError(t, err)
	require.NoError(t, scn.CancelAction(player))
	require.NoError(t, scn.WaitForEventLoop())

	creature := creatureInHand(t, player)
	_, err = scn.ActionPlayCardPrompt(player, creature.ID)
	require.NoError(t, err)

	// A late cancel for the question already settled.
	require.NoError(t, scn.SendAnswer(player, earlier.Prompt.ID, true, 0))

	assertPlayed(t, scn, player, creature, mana)
}

func TestASecondClickOnAnAnswerDoesNotCarryOver(t *testing.T) {
	scn, player := promptScenario(t)
	mana := manaForOne(t, player)

	first := creatureInHand(t, player)
	question, err := scn.ActionPlayCardPrompt(player, first.ID)
	require.NoError(t, err)

	// The pay answer, then a second click that sends a cancel for the same
	// question. The pay answer settles it; the cancel answers nothing.
	require.NoError(t, scn.SendAnswer(player, question.Prompt.ID, false, len(mana), mana...))
	require.NoError(t, scn.SendAnswer(player, question.Prompt.ID, true, 0))
	require.NoError(t, scn.WaitForEventLoop())
	assert.Equal(t, match.BATTLEZONE, first.Zone)

	// Fresh mana for the next play, the first having tapped what it used.
	mana = manaForOne(t, player)

	second := creatureInHand(t, player)
	_, err = scn.ActionPlayCardPrompt(player, second.ID)
	require.NoError(t, err)

	assertPlayed(t, scn, player, second, mana)
}

func TestAPlayerComingBackIsAskedTheSameQuestion(t *testing.T) {
	scn, player := promptScenario(t)
	mana := manaForOne(t, player)

	creature := creatureInHand(t, player)
	question, err := scn.ActionPlayCardPrompt(player, creature.ID)
	require.NoError(t, err)

	conn := scenario.NewMockConnection()
	socket := server.NewSocket(conn, scn.Match, player.UID, player.Username)
	join, err := json.Marshal(server.Message{Header: "join_match"})
	require.NoError(t, err)
	scn.Match.Parse(socket, join)

	require.Eventually(t, func() bool { return conn.OpenPrompt() != 0 }, promptTestTimeout, time.Millisecond,
		"the question still open is sent again")
	assert.Equal(t, question.Prompt.ID, conn.OpenPrompt(), "under the same number")

	require.NoError(t, scn.SendAnswer(player, conn.OpenPrompt(), false, len(mana), mana...))
	require.NoError(t, scn.WaitForEventLoop())
	assert.Equal(t, match.BATTLEZONE, creature.Zone)
}
