package decks

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/cards"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/validation"
)

// Limits on a deck list given as text.
const (
	TextMaxLength = 10000
	TextMaxLines  = 200
)

// Line is a line of a deck list: how many copies of which card, by name.
type Line struct {
	// Number counts from 1.
	Number int
	// Count and Name are the line read as a count and a name. Without a count,
	// Count is 1 and Name the whole line.
	Count int
	Name  string
	// Text is the whole line, which is also tried as a name, for cards whose
	// names start with a number, such as "750 Man".
	Text string
}

// countedLine is a count, optionally followed by x, then the name: "4
// Bolshack Dragon" or "4x Bolshack Dragon". The x must touch the count, since
// names such as "X Revolver Dragon" start with an X.
var countedLine = regexp.MustCompile(`^(\d{1,6})[xX×]?\s+(\S.*)$`)

// ParseText reads a deck list, one card per line as "4 Bolshack Dragon". A
// line without a count is one copy. Empty lines and lines starting with # or
// // are skipped. It returns a *validation.Error for text beyond the limits.
func ParseText(text string) ([]Line, error) {
	switch {
	case !utf8.ValidString(text):
		return nil, validation.New("text", "must be valid text")
	case utf8.RuneCountInString(text) > TextMaxLength:
		return nil, validation.New("text", fmt.Sprintf("must be at most %d characters", TextMaxLength))
	}

	rawLines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	if len(rawLines) > TextMaxLines {
		return nil, validation.New("text", fmt.Sprintf("must be at most %d lines", TextMaxLines))
	}

	var lines []Line
	for i, raw := range rawLines {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "//") {
			continue
		}

		line := Line{Number: i + 1, Count: 1, Name: trimmed, Text: trimmed}
		if match := countedLine.FindStringSubmatch(trimmed); match != nil {
			// At most six digits, so the count always fits.
			line.Count, _ = strconv.Atoi(match[1])
			line.Name = strings.TrimSpace(match[2])
		}

		lines = append(lines, line)
	}

	return lines, nil
}

// Text writes a deck list, one card per line as "4 Bolshack Dragon", by cost
// and then name. Cards that have left the card database are left out, since
// their names could not be read back.
func Text(entries []Entry, known map[string]cards.Card) string {
	sorted := append([]Entry(nil), entries...)
	SortForDisplay(sorted, known)

	var text strings.Builder
	for _, entry := range sorted {
		if card, found := known[entry.CardID]; found {
			fmt.Fprintf(&text, "%d %s\n", entry.Count, card.Name)
		}
	}

	return text.String()
}
