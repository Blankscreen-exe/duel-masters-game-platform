package decks_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/cards"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/decks"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/formats"
)

var (
	open    = formats.Format{Name: "Open", AllSets: true}
	classic = formats.Format{Name: "DM-01 to DM-12", Sets: []string{"DM-01", "DM-02"}}
)

// with replaces the first entries of a legal deck, keeping it at 40 cards
// when the replacements hold 4 cards each.
func with(replacements ...decks.Entry) []decks.Entry {
	entries := legalDeck()
	copy(entries, replacements)
	return entries
}

func TestALegalDeckHasNoProblems(t *testing.T) {
	cards := known(t)

	assert.Empty(t, decks.Check(legalDeck(), cards, open))
	assert.Empty(t, decks.Check(legalDeck(), cards, classic))
}

func TestADeckNeedsExactlyFortyCards(t *testing.T) {
	cards := known(t)

	short := legalDeck()[:9]
	assert.Equal(t, []decks.Problem{{Kind: decks.ProblemSize, Count: 36, Limit: 40}}, decks.Check(short, cards, open))

	long := append(legalDeck(), decks.Entry{CardID: bolshack, Count: 1})
	assert.Equal(t, []decks.Problem{{Kind: decks.ProblemSize, Count: 41, Limit: 40}}, decks.Check(long, cards, open))

	assert.Equal(t, []decks.Problem{{Kind: decks.ProblemSize, Count: 0, Limit: 40}}, decks.Check(nil, cards, open))
}

func TestCardsTheGameCannotPlayBreakEveryFormat(t *testing.T) {
	cards := known(t)

	for _, id := range []string{future, "gone-from-the-card-database"} {
		problems := decks.Check(with(decks.Entry{CardID: id, Count: 4}), cards, open)
		assert.Equal(t, []decks.Problem{{Kind: decks.ProblemNotPlayable, CardID: id, CardName: cards[id].Name, Count: 4}}, problems, id)
	}
}

func TestAFormatAllowsOnlyCardsFromItsSets(t *testing.T) {
	cards := known(t)

	for _, id := range []string{promo, late} {
		problems := decks.Check(with(decks.Entry{CardID: id, Count: 4}), cards, classic)
		assert.Equal(t, []decks.Problem{{Kind: decks.ProblemNotInFormat, CardID: id, CardName: cards[id].Name, Count: 4}}, problems, id)
		assert.Empty(t, decks.Check(with(decks.Entry{CardID: id, Count: 4}), cards, open), id)
	}

	assert.Empty(t, decks.Check(with(decks.Entry{CardID: reprint, Count: 4}), cards, classic),
		"a card counts as from a set when any of its printings is")
}

func TestAFormatBansAndLimitsCards(t *testing.T) {
	cards := known(t)
	limited := formats.Format{AllSets: true, Limits: map[string]int{bolshack: 0, hulcus: 1, reprint: 2}}

	problems := decks.Check(with(
		decks.Entry{CardID: bolshack, Count: 1},
		decks.Entry{CardID: hulcus, Count: 2},
		decks.Entry{CardID: reprint, Count: 2},
	), cards, limited)
	assert.Equal(t, []decks.Problem{
		{Kind: decks.ProblemSize, Count: 33, Limit: 40},
		{Kind: decks.ProblemBanned, CardID: bolshack, CardName: "Bolshack Dragon", Count: 1},
		{Kind: decks.ProblemOverLimit, CardID: hulcus, CardName: "Aqua Hulcus", Count: 2, Limit: 1},
	}, problems, "a card at its limit is fine")
}

func TestADeckHasAtMostFourCopiesOfACard(t *testing.T) {
	cards := known(t)

	problems := decks.Check(with(decks.Entry{CardID: basic01, Count: 5}, decks.Entry{CardID: basic02, Count: 3}), cards, open)
	assert.Equal(t, []decks.Problem{{Kind: decks.ProblemTooManyCopies, CardID: basic01, CardName: "Basic 01", Count: 5, Limit: 4}}, problems)
}

func TestProblemMessages(t *testing.T) {
	for problem, message := range map[decks.Problem]string{
		{Kind: decks.ProblemSize, Count: 36, Limit: 40}:                                            "The deck has 36 cards; it needs exactly 40.",
		{Kind: decks.ProblemSize, Count: 1, Limit: 40}:                                             "The deck has 1 card; it needs exactly 40.",
		{Kind: decks.ProblemNotPlayable, CardName: "Future Card"}:                                  "Future Card cannot be played yet.",
		{Kind: decks.ProblemNotPlayable, CardID: "gone"}:                                           "A card no longer in the card database cannot be played yet.",
		{Kind: decks.ProblemNotInFormat, CardName: "Promo Hero"}:                                   "Promo Hero is not from a set this format allows.",
		{Kind: decks.ProblemBanned, CardName: "Bolshack Dragon"}:                                   "Bolshack Dragon is banned in this format.",
		{Kind: decks.ProblemOverLimit, CardName: "Aqua Hulcus", Count: 2, Limit: 1}:                "Aqua Hulcus: 2 copies, but at most 1 is allowed.",
		{Kind: decks.ProblemTooManyCopies, CardName: "Basic 01", Count: 5, Limit: decks.MaxCopies}: "Basic 01: 5 copies, but at most 4 are allowed.",
		{Kind: decks.ProblemTooManyCivilizations, Count: 3, Limit: 1}:                              "The deck's cards have 3 civilizations, but at most 1 is allowed.",
		{Kind: decks.ProblemTooManyCivilizations, Count: 3, Limit: 2}:                              "The deck's cards have 3 civilizations, but at most 2 are allowed.",
		{Kind: decks.ProblemCardType, CardName: "Castle of Doom"}:                                  "Castle of Doom is a type of card this deck may not have.",
	} {
		assert.Equal(t, message, problem.Message())
	}
}

func TestCivilizationsAreInTheGamesOrder(t *testing.T) {
	cards := known(t)

	entries := []decks.Entry{{CardID: promo}, {CardID: bolshack}, {CardID: man750}, {CardID: hulcus}, {CardID: reprint}, {CardID: basic01}, {CardID: "gone"}}
	assert.Equal(t, []string{"Light", "Water", "Darkness", "Fire", "Nature"}, decks.Civilizations(entries, cards))
	assert.Empty(t, decks.Civilizations(nil, cards))
}

func TestTheCoverIsTheChosenCardOrTheMostExpensive(t *testing.T) {
	cards := known(t)
	entries := []decks.Entry{{CardID: basic02}, {CardID: man750}, {CardID: revolver}, {CardID: "gone"}}

	assert.Equal(t, basic02, decks.Cover(basic02, entries, cards))
	assert.Equal(t, man750, decks.Cover("", entries, cards), "750 Man and Revolver Dragon both cost 7; the first by name wins")
	assert.Empty(t, decks.Cover("", nil, cards))
	assert.Empty(t, decks.Cover("", []decks.Entry{{CardID: "gone"}}, cards))
}

func TestSortForDisplayOrdersByCostThenName(t *testing.T) {
	cards := known(t)
	entries := []decks.Entry{{CardID: "gone"}, {CardID: revolver}, {CardID: basic01}, {CardID: man750}, {CardID: hulcus}, {CardID: basic03}}

	decks.SortForDisplay(entries, cards)

	ids := make([]string, 0, len(entries))
	for _, entry := range entries {
		ids = append(ids, entry.CardID)
	}
	assert.Equal(t, []string{basic01, hulcus, basic03, man750, revolver, "gone"}, ids)
}

func TestRulesLimitCivilizationsAndCardTypes(t *testing.T) {
	known := map[string]cards.Card{
		"fire":   {ID: "fire", Name: "Fire Creature", Type: "Creature", Civilizations: []string{"Fire"}},
		"dual":   {ID: "dual", Name: "Fire Nature Creature", Type: "Creature", Civilizations: []string{"Fire", "Nature"}},
		"spell":  {ID: "spell", Name: "Water Spell", Type: "Spell", Civilizations: []string{"Water"}},
		"castle": {ID: "castle", Name: "Fire Castle", Type: "Castle", Civilizations: []string{"Fire"}},
	}
	creaturesAndSpells := []string{"Creature", "Evolution Creature", "Spell"}

	assert.Empty(t, decks.CheckRules([]decks.Entry{{CardID: "fire", Count: 4}}, known, decks.Rules{MaxCivilizations: 1, CardTypes: creaturesAndSpells}))
	assert.Empty(t, decks.CheckRules([]decks.Entry{{CardID: "dual", Count: 4}, {CardID: "spell", Count: 4}}, known, decks.Rules{}),
		"no rules, no problems")

	problems := decks.CheckRules([]decks.Entry{{CardID: "dual", Count: 4}, {CardID: "castle", Count: 2}}, known, decks.Rules{MaxCivilizations: 1, CardTypes: creaturesAndSpells})
	assert.Equal(t, []decks.Problem{
		{Kind: decks.ProblemTooManyCivilizations, Count: 2, Limit: 1},
		{Kind: decks.ProblemCardType, CardID: "castle", CardName: "Fire Castle", Count: 2},
	}, problems, "a multicoloured card counts each of its civilizations")

	assert.Empty(t, decks.CheckRules([]decks.Entry{{CardID: "dual", Count: 4}, {CardID: "fire", Count: 4}}, known, decks.Rules{MaxCivilizations: 2}))
}
