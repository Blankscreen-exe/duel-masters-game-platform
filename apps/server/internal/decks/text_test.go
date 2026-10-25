package decks_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/decks"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/validation"
)

func TestParseTextReadsCountsAndNames(t *testing.T) {
	lines, err := decks.ParseText(strings.Join([]string{
		"4 Bolshack Dragon",
		"  3x Aqua Hulcus  ",
		"2X Solar Ray",
		"1×Promo Hero",
		"Revolver Dragon",
		"",
		"# a comment",
		"// another",
		"4 X Revolver Dragon",
		"750 Man",
	}, "\r\n"))
	require.NoError(t, err)

	assert.Equal(t, []decks.Line{
		{Number: 1, Count: 4, Name: "Bolshack Dragon", Text: "4 Bolshack Dragon"},
		{Number: 2, Count: 3, Name: "Aqua Hulcus", Text: "3x Aqua Hulcus"},
		{Number: 3, Count: 2, Name: "Solar Ray", Text: "2X Solar Ray"},
		{Number: 4, Count: 1, Name: "1×Promo Hero", Text: "1×Promo Hero"},
		{Number: 5, Count: 1, Name: "Revolver Dragon", Text: "Revolver Dragon"},
		{Number: 9, Count: 4, Name: "X Revolver Dragon", Text: "4 X Revolver Dragon"},
		{Number: 10, Count: 750, Name: "Man", Text: "750 Man"},
	}, lines, "an x must touch the count; a name starting with X stays whole")
}

func TestParseTextRefusesTextBeyondTheLimits(t *testing.T) {
	for name, text := range map[string]string{
		"too long":       strings.Repeat("a", decks.TextMaxLength+1),
		"too many lines": strings.Repeat("\n", decks.TextMaxLines),
		"not text":       "4 Bolshack \xff",
	} {
		_, err := decks.ParseText(text)
		invalid, ok := validation.As(err)
		require.True(t, ok, "%s: got %v", name, err)
		assert.Equal(t, "text", invalid.Field, name)
	}

	_, err := decks.ParseText(strings.Repeat("\n", decks.TextMaxLines-1))
	assert.NoError(t, err, "exactly the most lines")
}

func TestTextWritesADeckListThatReadsBack(t *testing.T) {
	cards := known(t)

	text := decks.Text([]decks.Entry{{CardID: man750, Count: 2}, {CardID: basic10, Count: 1}, {CardID: hulcus, Count: 4}, {CardID: "gone", Count: 3}}, cards)
	assert.Equal(t, "4 Aqua Hulcus\n2 750 Man\n1 Basic 10\n", text, "by cost, then name, leaving out cards no longer in the card database")

	lines, err := decks.ParseText(text)
	require.NoError(t, err)
	assert.Equal(t, decks.Line{Number: 2, Count: 2, Name: "750 Man", Text: "2 750 Man"}, lines[1])

	assert.Empty(t, decks.Text(nil, cards))
}
