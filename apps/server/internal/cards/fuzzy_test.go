package cards

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFuzzyWordMatch(t *testing.T) {
	for _, test := range []struct {
		name         string
		typed, word  string
		wantedResult bool
	}{
		{"the word itself", "draw", "draw", true},
		{"the start of a word", "dest", "destroy", true},
		{"a single letter starts a word", "d", "draw", true},
		{"two neighbouring letters swapped", "crad", "card", true},
		{"a letter missed", "creatre", "creature", true},
		{"a letter added", "creaature", "creature", true},
		{"a letter changed", "shiald", "shield", true},
		{"two mistakes in a long word", "opponant's", "opponents", true},
		{"two mistakes in a medium word", "shaald", "shield", false},
		{"three mistakes in a long word", "oppanantz", "opponents", false},
		{"a mistake in a short word", "tap", "top", false},
		{"a mistake in the first letter", "braw", "draw", false},
		{"a very different length", "drawing", "dr", false},
		{"another word altogether", "blocker", "destroy", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.wantedResult, fuzzyWordMatch(test.typed, test.word))
		})
	}
}

func TestFuzzyMatch(t *testing.T) {
	words := fuzzyWords("When you put this creature into the battle zone, you may draw a card.")

	assert.True(t, fuzzyMatch(fuzzyWords("draw a crad"), words), "a typing mistake")
	assert.True(t, fuzzyMatch(fuzzyWords("card draw"), words), "in any order")
	assert.True(t, fuzzyMatch(fuzzyWords("DRAW, card!"), words), "ignoring case and punctuation")
	assert.False(t, fuzzyMatch(fuzzyWords("draw shield"), words), "every word must be found")
	assert.False(t, fuzzyMatch(nil, words), "no words match nothing")
	assert.False(t, fuzzyMatch(fuzzyWords("draw"), nil), "a card without text matches nothing")
}

func TestFuzzyWordsFoldsAccentsAndPunctuation(t *testing.T) {
	assert.Equal(t, []string{"uberdragon", "s", "2", "shields"}, fuzzyWords("Überdragon's 2 SHIELDS."))
	assert.Empty(t, fuzzyWords(" ... "))
}
