package store

import (
	"strings"

	"gorm.io/gorm/clause"
)

// Contains is the condition that a text column contains text, in the same
// way on every database: LIKE's wildcards % and _ in text, and its escape
// character, match themselves. Matching is as exact as the column's values,
// so search a lowercase column with lowercase text to ignore case.
func Contains(column, text string) clause.Expression {
	return clause.Expr{
		SQL:  `? LIKE ? ESCAPE '\'`,
		Vars: []any{clause.Column{Name: column}, "%" + escapeLike(text) + "%"},
	}
}

func escapeLike(text string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(text)
}
