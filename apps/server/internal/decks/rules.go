// Package decks keeps the players' decks, reads and writes them as text, and
// holds the one check of whether a deck keeps to a format, used everywhere a
// deck's legality matters: the deck builder, the lobby and the games.
package decks

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/cards"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/formats"
)

// The game's own deck rules (design/SPEC.md, section 5).
const (
	// Size is how many cards a deck needs to be played.
	Size = 40
	// MaxCopies is the most copies of one card a deck may have.
	MaxCopies = 4
)

// LegalityIn is what a format allows, for a card search to keep to.
func LegalityIn(format formats.Format) cards.Legality {
	legal := cards.Legality{AllSets: format.AllSets, Sets: format.Sets, MaxCivilizations: format.MaxCivilizations}
	for cardID, limit := range format.Limits {
		if limit == 0 {
			legal.Banned = append(legal.Banned, cardID)
		}
	}
	return legal
}

// Entry is how many copies of a card a deck has.
type Entry struct {
	CardID string
	Count  int
}

// ProblemKind names a way a deck breaks a format's rules. The values are part
// of the API.
type ProblemKind string

const (
	// ProblemSize: the deck does not have exactly Size cards.
	ProblemSize ProblemKind = "deck_size"
	// ProblemNotPlayable: the game cannot play the card, or it has left the
	// card database.
	ProblemNotPlayable ProblemKind = "not_playable"
	// ProblemNotInFormat: none of the card's printings is in a set the
	// format allows.
	ProblemNotInFormat ProblemKind = "not_in_format"
	// ProblemBanned: the format bans the card.
	ProblemBanned ProblemKind = "banned"
	// ProblemOverLimit: the format limits the card to fewer copies.
	ProblemOverLimit ProblemKind = "over_limit"
	// ProblemTooManyCopies: more than MaxCopies copies of the card.
	ProblemTooManyCopies ProblemKind = "too_many_copies"
	// ProblemTooManyCivilizations: the deck's cards have more civilizations
	// than its Rules allow.
	ProblemTooManyCivilizations ProblemKind = "too_many_civilizations"
	// ProblemTooFewCivilizations: the deck's cards have fewer civilizations
	// than its format needs.
	ProblemTooFewCivilizations ProblemKind = "too_few_civilizations"
	// ProblemCardType: the deck's Rules do not allow the card's type.
	ProblemCardType ProblemKind = "card_type"
	// ProblemFormatRetired: the format the deck keeps to has been retired.
	ProblemFormatRetired ProblemKind = "format_retired"
)

// Problem is one way a deck breaks a format's rules.
type Problem struct {
	Kind ProblemKind
	// CardID and CardName are the card at fault; empty for ProblemSize.
	// CardName is empty for a card that has left the card database.
	CardID   string
	CardName string
	// Count is how many there are: cards in the deck for ProblemSize,
	// civilizations for ProblemTooManyCivilizations and
	// ProblemTooFewCivilizations, copies of the card otherwise.
	Count int
	// Limit is how many are allowed, for ProblemSize, ProblemOverLimit,
	// ProblemTooManyCopies and ProblemTooManyCivilizations, or needed, for
	// ProblemTooFewCivilizations.
	Limit int
}

// Message explains the problem to the player.
func (p Problem) Message() string {
	name := p.CardName
	if name == "" {
		name = "A card no longer in the card database"
	}

	switch p.Kind {
	case ProblemSize:
		return fmt.Sprintf("The deck has %d %s; it needs exactly %d.", p.Count, plural(p.Count, "card", "cards"), p.Limit)
	case ProblemNotPlayable:
		return fmt.Sprintf("%s cannot be played yet.", name)
	case ProblemNotInFormat:
		return fmt.Sprintf("%s is not from a set this format allows.", name)
	case ProblemBanned:
		return fmt.Sprintf("%s is banned in this format.", name)
	case ProblemOverLimit, ProblemTooManyCopies:
		return fmt.Sprintf("%s: %d copies, but at most %d %s allowed.", name, p.Count, p.Limit, plural(p.Limit, "is", "are"))
	case ProblemTooManyCivilizations:
		return fmt.Sprintf("The deck's cards have %d civilizations, but at most %d %s allowed.", p.Count, p.Limit, plural(p.Limit, "is", "are"))
	case ProblemTooFewCivilizations:
		return fmt.Sprintf("The deck's cards have %d %s, but at least %d %s needed.", p.Count, plural(p.Count, "civilization", "civilizations"), p.Limit, plural(p.Limit, "is", "are"))
	case ProblemCardType:
		return fmt.Sprintf("%s is a type of card this deck may not have.", name)
	case ProblemFormatRetired:
		return "The deck's format has been retired."
	default:
		return string(p.Kind)
	}
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// Check lists every way a deck breaks a format's rules, or nothing for a legal
// deck. known holds the deck's cards from the catalogue; a card missing from
// it has left the card database. Problems come in the deck's order, after the
// deck's size and its civilizations.
func Check(entries []Entry, known map[string]cards.Card, format formats.Format) []Problem {
	var problems []Problem
	if total := Count(entries); total != Size {
		problems = append(problems, Problem{Kind: ProblemSize, Count: total, Limit: Size})
	}
	problems = append(problems, civilizationProblems(entries, known, format.MinCivilizations, format.MaxCivilizations)...)

	for _, entry := range entries {
		card, found := known[entry.CardID]
		problem := Problem{CardID: entry.CardID, CardName: card.Name, Count: entry.Count}

		limit, limited := format.Limit(entry.CardID)
		switch {
		case !found || !card.Playable():
			problem.Kind = ProblemNotPlayable
		case !slices.ContainsFunc(card.Printings, func(p cards.Printing) bool { return format.AllowsSet(p.SetCode) }):
			problem.Kind = ProblemNotInFormat
		case limited && limit == 0:
			problem.Kind = ProblemBanned
		case limited && entry.Count > limit:
			problem.Kind, problem.Limit = ProblemOverLimit, limit
		case entry.Count > MaxCopies:
			problem.Kind, problem.Limit = ProblemTooManyCopies, MaxCopies
		default:
			continue
		}

		problems = append(problems, problem)
	}

	return problems
}

// Rules are limits a deck may be held to beyond its format's, such as a
// computer opponent's deck.
type Rules struct {
	// MaxCivilizations is the most civilizations the deck's cards may have,
	// 0 for any number.
	MaxCivilizations int
	// CardTypes are the types of card the deck may have, by the card
	// database's names for them; empty allows every type.
	CardTypes []string
}

// CheckRules lists every way a deck breaks the rules, or nothing for a deck
// keeping to them. known holds the deck's cards from the catalogue; a card
// missing from it is left to Check. Problems come with the civilizations
// first, then in the deck's order.
func CheckRules(entries []Entry, known map[string]cards.Card, rules Rules) []Problem {
	problems := civilizationProblems(entries, known, 0, rules.MaxCivilizations)

	for _, entry := range entries {
		card, found := known[entry.CardID]
		if found && len(rules.CardTypes) > 0 && !slices.Contains(rules.CardTypes, card.Type) {
			problems = append(problems, Problem{Kind: ProblemCardType, CardID: entry.CardID, CardName: card.Name, Count: entry.Count})
		}
	}

	return problems
}

// civilizationProblems holds a deck's civilizations to the fewest and the most
// allowed, 0 leaving either open. A deck with no civilizations yet has only begun,
// which its size already says, so it is not told it has too few as well.
func civilizationProblems(entries []Entry, known map[string]cards.Card, fewest, most int) []Problem {
	civilizations := len(Civilizations(entries, known))
	switch {
	case most > 0 && civilizations > most:
		return []Problem{{Kind: ProblemTooManyCivilizations, Count: civilizations, Limit: most}}
	case civilizations > 0 && civilizations < fewest:
		return []Problem{{Kind: ProblemTooFewCivilizations, Count: civilizations, Limit: fewest}}
	default:
		return nil
	}
}

// Count is how many cards the entries hold.
func Count(entries []Entry) int {
	total := 0
	for _, entry := range entries {
		total += entry.Count
	}
	return total
}

// civilizationOrder is the order the game lists civilizations in.
var civilizationOrder = []string{"Light", "Water", "Darkness", "Fire", "Nature"}

// Civilizations lists the civilizations of a deck's cards, in the game's
// order.
func Civilizations(entries []Entry, known map[string]cards.Card) []string {
	present := map[string]bool{}
	for _, entry := range entries {
		for _, civilization := range known[entry.CardID].Civilizations {
			present[civilization] = true
		}
	}

	found := make([]string, 0, len(present))
	for civilization := range present {
		found = append(found, civilization)
	}
	slices.SortFunc(found, CompareCivilizations)

	return found
}

// CompareCivilizations orders civilizations in the game's order, any it does
// not know last, by name.
func CompareCivilizations(a, b string) int {
	return cmp.Or(cmp.Compare(civilizationRank(a), civilizationRank(b)), cmp.Compare(a, b))
}

func civilizationRank(civilization string) int {
	if rank := slices.Index(civilizationOrder, civilization); rank >= 0 {
		return rank
	}
	return len(civilizationOrder)
}

// SortForDisplay orders entries as deck lists show them: by cost, then name.
// Cards that have left the card database go last, by id.
func SortForDisplay(entries []Entry, known map[string]cards.Card) {
	slices.SortStableFunc(entries, func(a, b Entry) int {
		cardA, foundA := known[a.CardID]
		cardB, foundB := known[b.CardID]
		if foundA != foundB {
			if foundA {
				return -1
			}
			return 1
		}

		return cmp.Or(cmp.Compare(cardA.Cost, cardB.Cost), cmp.Compare(cardA.Name, cardB.Name), cmp.Compare(a.CardID, b.CardID))
	})
}

// Cover is the card shown for a deck: the one its player chose, or else its
// most expensive card (by name between equals). It is empty for an empty deck.
func Cover(chosen string, entries []Entry, known map[string]cards.Card) string {
	if chosen != "" {
		return chosen
	}

	var cover cards.Card
	for _, entry := range entries {
		card, found := known[entry.CardID]
		if found && (cover.ID == "" || card.Cost > cover.Cost || (card.Cost == cover.Cost && card.Name < cover.Name)) {
			cover = card
		}
	}

	return cover.ID
}
