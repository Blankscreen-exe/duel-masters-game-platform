package computer

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/cards"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/cnd"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/server"
)

// testCard is a card for the test library, by engine id.
func testCard(id, cardType string, cost, power int, civilizations []string, keywords, categories []string) cards.Card {
	return cards.Card{
		ID: id, EngineID: id, Name: id, Type: cardType, Cost: cost, Power: &power,
		Civilizations: civilizations, Keywords: keywords, Categories: categories,
	}
}

// testLibrary is a library of a few made-up cards.
func testLibrary() *Library {
	return NewLibrary([]cards.Card{
		testCard("fire-small", typeCreature, 2, 2000, []string{"Fire"}, nil, nil),
		testCard("fire-big", typeCreature, 7, 9000, []string{"Fire"}, []string{"double_break"}, nil),
		testCard("fire-rusher", typeCreature, 3, 3000, []string{"Fire"}, nil, []string{"must_attack"}),
		testCard("water-blocker", typeCreature, 3, 3000, []string{"Water"}, []string{"blocker"}, nil),
		testCard("water-wall", typeCreature, 6, 6000, []string{"Water"}, []string{"blocker"}, nil),
		testCard("nature-fire", typeCreature, 5, 5000, []string{"Fire", "Nature"}, nil, nil),
		testCard("nature-mana", typeCreature, 2, 1000, []string{"Nature"}, nil, nil),
		testCard("discard-own", typeSpell, 2, 0, []string{"Darkness"}, nil, []string{"discard_own"}),
		testCard("removal", typeSpell, 4, 0, []string{"Darkness"}, nil, []string{"removal"}),
		testCard("discard-opp", typeSpell, 4, 0, []string{"Darkness"}, nil, []string{"discard_opp"}),
	})
}

// testPlayer is a computer player with a library of a few made-up cards.
func testPlayer() *Player {
	return testPlayerWith(Balanced, Normal)
}

func testPlayerWith(style Style, strength Strength) *Player {
	return New(Options{Library: testLibrary(), Style: style, Strength: strength, Seed: 7})
}

// in is a card as the engine sends it: a virtualId and the engine's id.
func in(virtualID, engineID string, flags ...match.CardFlags) server.CardState {
	var all uint8
	for _, flag := range flags {
		all |= uint8(flag)
	}
	return server.CardState{CardID: virtualID, ImageID: engineID, Flags: all}
}

// creatures puts creatures in a side's battle zone, with their power.
func creatures(side *server.PlayerState, list ...struct {
	card     server.CardState
	power    int
	statuses []string
}) {
	side.Powers = map[string]server.CreaturePower{}
	side.Statuses = map[string][]server.CreatureStatus{}
	for _, c := range list {
		side.Battlezone = append(side.Battlezone, c.card)
		side.Powers[c.card.CardID] = server.CreaturePower{Current: c.power, Attacking: c.power}
		for _, status := range c.statuses {
			side.Statuses[c.card.CardID] = append(side.Statuses[c.card.CardID], server.CreatureStatus{ID: status})
		}
	}
}

type creature = struct {
	card     server.CardState
	power    int
	statuses []string
}

func TestManaPaysEachCivilizationWithItsOwnCard(t *testing.T) {
	pl := testPlayer()
	// The two-civilization card can only pay for Nature if the Fire card
	// pays for Fire, whichever order they come in.
	q := &question{max: 5, about: server.PromptAbout{Kind: string(match.PromptPayMana), Source: &server.PromptSource{ImageID: "nature-fire"}}}
	offered := []server.CardState{
		in("dual", "nature-fire"), in("fire-1", "fire-small"), in("fire-2", "fire-small"),
		in("fire-3", "fire-small"), in("fire-4", "fire-small"), in("water", "water-blocker"),
	}

	paid := pl.payMana(q, offered, &server.MatchState{})

	assert.Len(t, paid, 5)
	assert.Contains(t, paid, "dual", "the only card with Nature pays for it")
	assert.True(t, slices.ContainsFunc(paid, func(id string) bool { return id != "dual" && id != "water" }), "a Fire card pays for Fire")
}

func TestManaKeepsTheCivilizationsTheHandNeeds(t *testing.T) {
	pl := testPlayer()
	q := &question{max: 2, about: server.PromptAbout{Kind: string(match.PromptPayMana), Source: &server.PromptSource{ImageID: "fire-small"}}}
	offered := []server.CardState{in("fire", "fire-small"), in("water", "water-blocker"), in("nature", "nature-mana")}
	state := &server.MatchState{Me: server.PlayerState{Hand: []server.CardState{in("next", "water-wall")}}}

	paid := pl.payMana(q, offered, state)

	assert.ElementsMatch(t, []string{"fire", "nature"}, paid, "the Water the hand needs stays untapped")
}

func TestBlockingWinsTheBattleWhenItCan(t *testing.T) {
	pl := testPlayer()
	state := &server.MatchState{Me: server.PlayerState{Shieldzone: make([]server.ShieldState, 5)}}
	creatures(&state.Me, creature{card: in("small", "water-blocker"), power: 3000}, creature{card: in("wall", "water-wall"), power: 6000})
	creatures(&state.Opponent, creature{card: in("attacker", "nature-fire"), power: 5000})
	q := &question{max: 1, cancellable: true, about: server.PromptAbout{Kind: string(match.PromptBlock), Source: &server.PromptSource{CardID: "attacker", ImageID: "nature-fire"}}}
	q.cards = state.Me.Battlezone

	cards, cancel := pl.chooseCards(q, state)

	assert.False(t, cancel)
	assert.Equal(t, []string{"wall"}, cards, "the blocker that beats the attacker")
}

func TestBlockingSavesTheLastShieldOnly(t *testing.T) {
	pl := testPlayer()
	state := &server.MatchState{Me: server.PlayerState{Shieldzone: make([]server.ShieldState, 4)}}
	creatures(&state.Me, creature{card: in("small", "water-blocker"), power: 3000})
	creatures(&state.Opponent, creature{card: in("attacker", "nature-fire"), power: 5000})
	q := &question{max: 1, cancellable: true, about: server.PromptAbout{Kind: string(match.PromptBlock), Source: &server.PromptSource{CardID: "attacker", ImageID: "nature-fire"}}}
	q.cards = state.Me.Battlezone

	_, cancel := pl.chooseCards(q, state)
	assert.True(t, cancel, "with shields to spare, it does not throw its blocker away")

	state.Me.Shieldzone = nil
	cards, cancel := pl.chooseCards(q, state)
	assert.False(t, cancel)
	assert.Equal(t, []string{"small"}, cards, "with no shields left, the attack would lose the game")
}

func TestAnEffectHitsTheOpponentsBestCardAndCostsItsOwnLeast(t *testing.T) {
	pl := testPlayer()
	state := &server.MatchState{}
	creatures(&state.Opponent, creature{card: in("their-small", "fire-small"), power: 2000}, creature{card: in("their-big", "fire-big"), power: 9000})

	harm := &question{min: 1, max: 1, cards: state.Opponent.Battlezone, about: server.PromptAbout{
		Kind: string(match.PromptEffect), OpponentCards: []string{"their-small", "their-big"},
	}}
	cards, _ := pl.chooseCards(harm, state)
	assert.Equal(t, []string{"their-big"}, cards)

	state.Me.Hand = []server.CardState{in("keep", "fire-big"), in("lose", "nature-mana")}
	cost := &question{min: 1, max: 2, cards: state.Me.Hand, about: server.PromptAbout{
		Kind: string(match.PromptEffect), Source: &server.PromptSource{ImageID: "discard-own"},
	}}
	cards, _ = pl.chooseCards(cost, state)
	assert.Equal(t, []string{"lose"}, cards, "only as many as it must, the least worth")
}

func TestARefusedAnswerIsNotRepeated(t *testing.T) {
	pl := testPlayer()
	offered := []server.CardState{in("a", "fire-small"), in("b", "fire-small"), in("c", "fire-small")}
	refuse := func(q *question) {
		pl.view.question = q
		pl.view.take([]byte(`{"header":"action_error","message":"Your selection of cards does not fulfill the requirements"}`))
	}

	optional := &question{min: 1, max: 1, cancellable: true, cards: offered, about: server.PromptAbout{Kind: string(match.PromptEffect)}}
	first, ok := pl.answerTo(optional, &server.MatchState{})
	require.True(t, ok)
	assert.False(t, first.Cancel)
	again, ok := pl.answerTo(optional, &server.MatchState{})
	require.True(t, ok)
	assert.Equal(t, first, again, "asked again with no refusal, as when the engine is slow, it answers the same")
	refuse(optional)
	refused, ok := pl.answerTo(optional, &server.MatchState{})
	require.True(t, ok)
	assert.True(t, refused.Cancel, "a refused answer to a question it may cancel is cancelled")

	mandatory := &question{min: 2, max: 2, cards: offered, about: server.PromptAbout{Kind: string(match.PromptEffect)}}
	for refusal := 1; refusal <= maxRefusals; refusal++ {
		refuse(mandatory)
		reply, ok := pl.answerTo(mandatory, &server.MatchState{})
		require.True(t, ok, "after refusal %d", refusal)
		assert.False(t, reply.Cancel)
		assert.Len(t, reply.Cards, 2, "after refusal %d it keeps to what the question allows", refusal)
	}
	refuse(mandatory)
	_, ok = pl.answerTo(mandatory, &server.MatchState{})
	assert.False(t, ok, "it stops trying after the last refusal")
}

func TestARefusedEndOfTurnMakesEveryCreatureThatCanAttack(t *testing.T) {
	pl := testPlayer()
	state := &server.MatchState{MyTurn: true, Turn: 3, TurnManaCharged: true, Opponent: server.PlayerState{Shieldzone: make([]server.ShieldState, 5)}}
	creatures(&state.Me, creature{card: in("small", "fire-small"), power: 2000})
	creatures(&state.Opponent, creature{card: in("wall", "water-wall"), power: 6000, statuses: []string{cnd.Blocker}})

	assert.Equal(t, headerOnly{Header: "end_turn"}, pl.nextMove(state), "the attack would lose, so it ends the turn")
	assert.Equal(t, headerOnly{Header: "end_turn"}, pl.nextMove(state), "asked again with no refusal, it ends the turn again")

	pl.view.take([]byte(`{"header":"warn","message":"Small must attack before you can end your turn"}`))
	assert.Equal(t, cardMove{Header: "attack_player", ID: "small"}, pl.nextMove(state), "refused, it attacks")
}

func TestAMustAttackCreatureAttacksWhateverTheRisk(t *testing.T) {
	pl := testPlayer()
	state := &server.MatchState{MyTurn: true, Turn: 3, Opponent: server.PlayerState{Shieldzone: make([]server.ShieldState, 5)}}
	creatures(&state.Me, creature{card: in("rusher", "fire-rusher"), power: 3000})
	creatures(&state.Opponent, creature{card: in("wall", "water-wall"), power: 6000, statuses: []string{cnd.Blocker}})
	pl.plan.startTurn(state.Turn)

	move, ok := pl.attack(state, board{side: &state.Me}, board{side: &state.Opponent}, false)

	require.True(t, ok)
	assert.Equal(t, cardMove{Header: "attack_player", ID: "rusher"}, move, "into a blocker that beats it, because it must")
}

func TestAnAttackIntoABlockerThatWinsIsHeldBack(t *testing.T) {
	pl := testPlayer()
	state := &server.MatchState{MyTurn: true, Turn: 3, Opponent: server.PlayerState{Shieldzone: make([]server.ShieldState, 5)}}
	creatures(&state.Me, creature{card: in("small", "fire-small"), power: 2000})
	creatures(&state.Opponent, creature{card: in("wall", "water-wall"), power: 6000, statuses: []string{cnd.Blocker}})
	pl.plan.startTurn(state.Turn)

	_, ok := pl.attack(state, board{side: &state.Me}, board{side: &state.Opponent}, false)
	assert.False(t, ok)

	_, ok = pl.attack(state, board{side: &state.Me}, board{side: &state.Opponent}, true)
	assert.True(t, ok, "unless the engine will not end the turn without it")
}

func TestChargingManaFavoursACivilizationItLacks(t *testing.T) {
	pl := testPlayer()
	state := &server.MatchState{Me: server.PlayerState{
		Manazone: []server.CardState{in("m1", "fire-small"), in("m2", "fire-small")},
		Hand:     []server.CardState{in("fire", "fire-small"), in("water", "water-blocker")},
	}}

	card, ok := pl.manaToCharge(state)

	require.True(t, ok)
	assert.Equal(t, "water", card)
}

func TestTheViewReadsQuestionsInParts(t *testing.T) {
	var v view
	v.take([]byte(`{"header":"action","prompt":4,"kind":"effect","cards":{"Your hand":[{"virtualId":"a","uid":"x"}],"Your mana":[{"virtualId":"b","uid":"y"}]},"minSelections":1,"maxSelections":1,"cancellable":true}`))

	require.NotNil(t, v.question)
	assert.Equal(t, 4, v.question.id)
	assert.ElementsMatch(t, []string{"a", "b"}, ids(v.question.cards))

	v.take([]byte(`{"header":"wait","message":"Waiting"}`))
	assert.True(t, v.waiting)
	v.take([]byte(`{"header":"close_action"}`))
	v.take([]byte(`{"header":"end_wait"}`))
	assert.Nil(t, v.question)
	assert.False(t, v.waiting)
}

func TestAStylePlaysWhatItFavoursFirst(t *testing.T) {
	state := &server.MatchState{
		MyTurn:   true,
		Me:       server.PlayerState{Hand: []server.CardState{in("blocker", "water-blocker", match.PlayableFlag), in("discard", "discard-opp", match.PlayableFlag)}},
		Opponent: server.PlayerState{HandCount: 5},
	}

	balanced, _ := testPlayerWith(Balanced, Normal).cardToPlay(state)
	assert.Equal(t, "blocker", balanced)

	discard, ok := StyleNamed("discard")
	require.True(t, ok)
	favoured, _ := testPlayerWith(discard, Normal).cardToPlay(state)
	assert.Equal(t, "discard", favoured, "a discard style empties the opponent's hand first")

	state.Opponent.HandCount = 0
	idle, _ := testPlayerWith(discard, Normal).cardToPlay(state)
	assert.Equal(t, "blocker", idle, "but not when there is no hand to empty")
}

func TestOnlyAnEasyPlayerMakesMistakes(t *testing.T) {
	normal, easy := testPlayerWith(Balanced, Normal), testPlayerWith(Balanced, Easy)
	mistakes := 0
	for range 1000 {
		assert.False(t, normal.errs())
		if easy.errs() {
			mistakes++
		}
	}
	assert.InDelta(t, mistakeChance*1000, mistakes, 60, "about %v of the time", mistakeChance)
}

func TestEveryStyleHasAName(t *testing.T) {
	names := StyleNames()
	assert.Equal(t, []string{"balanced", "bounce", "discard", "giant", "rush", "tapper", "wall"}, names)
	for _, name := range names {
		_, ok := StyleNamed(name)
		assert.True(t, ok, name)
	}
	_, ok := StyleNamed("chaos")
	assert.False(t, ok)
}

func TestACreatureThatHasToAttackTriesEveryAttack(t *testing.T) {
	pl := testPlayer()
	state := &server.MatchState{MyTurn: true, Turn: 3, TurnManaCharged: true, Opponent: server.PlayerState{Shieldzone: make([]server.ShieldState, 5)}}
	creatures(&state.Me, creature{card: in("rusher", "fire-rusher"), power: 3000})
	creatures(&state.Opponent, creature{card: in("giant", "fire-big", match.TappedFlag), power: 9000})

	// It must attack, and cannot beat the giant, so it attacks the shields;
	// an effect may insist it attacks the giant instead, so refused, it
	// attacks the giant, taking any target.
	assert.Equal(t, cardMove{Header: "attack_player", ID: "rusher"}, pl.nextMove(state))
	assert.Equal(t, cardMove{Header: "attack_creature", ID: "rusher"}, pl.nextMove(state))
	assert.True(t, pl.plan.anyTarget["rusher"])

	target := &question{min: 1, max: 1, cancellable: true, cards: state.Opponent.Battlezone, about: server.PromptAbout{
		Kind: string(match.PromptAttackTarget), Source: &server.PromptSource{CardID: "rusher", ImageID: "fire-rusher"},
	}}
	pl.view.question = target
	pl.view.take([]byte(`{"header":"action_error","message":"Your selection of cards does not fulfill the requirements"}`))
	reply, ok := pl.answerTo(target, state)
	require.True(t, ok)
	assert.False(t, reply.Cancel, "a target refused, it tries another rather than give up its attack")
	assert.Equal(t, []string{"giant"}, reply.Cards)

	assert.Equal(t, headerOnly{Header: "end_turn"}, pl.nextMove(state), "with every attack tried, it ends the turn")
}
