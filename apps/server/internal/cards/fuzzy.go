package cards

import (
	"context"
	"fmt"
	"strings"
	"unicode"
)

// The longest word still needing an exact match, and the shortest allowed two
// mistakes; words between are allowed one.
const (
	exactWordLength   = 3
	twoMistakesLength = 8
)

// textMatches returns the ids of the cards in the card database whose rules
// text fuzzily matches text. The database cannot do this the same way on SQLite
// and Postgres, so it is done here.
func (c *Catalogue) textMatches(ctx context.Context, text string) ([]string, error) {
	wanted := fuzzyWords(text)

	var rows []cardRow
	if err := c.db.WithContext(ctx).Select("id", "rules_text").Where("in_database = ?", true).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("reading rules text: %w", err)
	}

	matches := make([]string, 0, len(rows))
	for _, row := range rows {
		if fuzzyMatch(wanted, fuzzyWords(row.RulesText)) {
			matches = append(matches, row.ID)
		}
	}

	return matches, nil
}

// fuzzyWords splits text into its words, in lowercase without accents.
func fuzzyWords(text string) []string {
	return strings.FieldsFunc(searchText(text), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
}

// fuzzyMatch reports whether each wanted word is close to one of the words, in
// any order. No wanted words matches nothing.
func fuzzyMatch(wanted, words []string) bool {
	if len(wanted) == 0 {
		return false
	}

	for _, word := range wanted {
		found := false
		for _, candidate := range words {
			if fuzzyWordMatch(word, candidate) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// fuzzyWordMatch reports whether a typed word is the start of a word, so "dest"
// finds "destroy", or that word with a few typing mistakes. Short words must be
// typed exactly, as a mistake in one makes it another word, and a mistake is
// taken to spare the first letter, for the same reason.
func fuzzyWordMatch(typed, word string) bool {
	if strings.HasPrefix(word, typed) {
		return true
	}

	a, b := []rune(typed), []rune(word)
	if len(a) <= exactWordLength || len(b) == 0 || a[0] != b[0] {
		return false
	}

	allowed := 1
	if len(a) >= twoMistakesLength {
		allowed = 2
	}
	if difference := len(a) - len(b); difference > allowed || -difference > allowed {
		return false
	}

	return typingMistakes(a, b) <= allowed
}

// typingMistakes counts the letters missed, added or changed, and neighbouring
// letters swapped, that turn a into b (optimal string alignment distance).
func typingMistakes(a, b []rune) int {
	// Three rows of the usual table: only the last two are ever looked back at.
	beforeLast := make([]int, len(b)+1)
	last := make([]int, len(b)+1)
	row := make([]int, len(b)+1)
	for j := range last {
		last[j] = j
	}

	for i := 1; i <= len(a); i++ {
		row[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			row[j] = min(last[j]+1, row[j-1]+1, last[j-1]+cost)
			if i > 1 && j > 1 && a[i-1] == b[j-2] && a[i-2] == b[j-1] {
				row[j] = min(row[j], beforeLast[j-2]+1)
			}
		}
		beforeLast, last, row = last, row, beforeLast
	}

	return last[len(b)]
}
