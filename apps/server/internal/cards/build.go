package cards

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/games"
)

// engineNameAliases maps names the engine spells differently from the card
// database to the card database's spelling. Matching already ignores case,
// accents, spaces and punctuation, so only real spelling differences belong
// here.
var engineNameAliases = map[string]string{
	"Hypersprint Warior Uzesol": "Hypersprint Warrior Uzesol",
}

// Card is a card as the catalogue keeps it, with every printing of it.
type Card struct {
	// ID is its first printing's id in the card database, such as "0079".
	ID            string
	Name          string
	Type          string
	Cost          int
	Power         *int
	PowerText     string
	Civilizations []string
	Races         []string
	RulesText     string
	Abilities     []Ability
	Keywords      []string
	Categories    []string
	// Printings are in release order; the details above are the first's.
	Printings []Printing
	// EngineID is the rules engine's id for the card, or empty when the engine
	// cannot play it yet.
	EngineID string
}

// Ability is one line of a card's rules text.
type Ability struct {
	Text     string
	Reminder string
}

// Printing is one printing of a card.
type Printing struct {
	ID              string
	SetCode         string
	CollectorNumber string
	Rarity          string
	RarityCode      string
	// ImagePath is the picture's path inside the card database's data folder,
	// or empty when the picture is missing.
	ImagePath string
}

// Playable reports whether the rules engine can play the card.
func (c Card) Playable() bool { return c.EngineID != "" }

// Build groups the card database's printings into cards, in name order, and
// links each card the engine can play to it. problems lists what could not be
// linked or lacks a picture; those cards are kept, unplayable or without a
// picture.
func Build(source Source, playable []games.PlayableCard) (cards []Card, problems []string) {
	cards = groupPrintings(source)
	problems = linkEngineCards(cards, playable)

	for _, card := range cards {
		for _, printing := range card.Printings {
			if printing.ImagePath == "" {
				problems = append(problems, fmt.Sprintf("printing %s of %q has no picture", printing.ID, card.Name))
			}
		}
	}

	return cards, problems
}

// groupPrintings puts the printings of each name together. Names are compared
// exactly, so two different cards whose names differ only in punctuation are
// never merged.
func groupPrintings(source Source) []Card {
	setRank := make(map[string]int, len(source.Sets))
	for i, set := range source.Sets {
		setRank[set.Code] = i
	}

	byName := make(map[string][]SourcePrinting)
	for _, printing := range source.Printings {
		name := strings.TrimSpace(printing.Name)
		byName[name] = append(byName[name], printing)
	}

	cards := make([]Card, 0, len(byName))
	for name, printings := range byName {
		sort.Slice(printings, func(i, j int) bool {
			a, b := printings[i], printings[j]
			if setRank[a.SetCode] != setRank[b.SetCode] {
				return setRank[a.SetCode] < setRank[b.SetCode]
			}
			if c := compareCollectorNumbers(a.CollectorNumber, b.CollectorNumber); c != 0 {
				return c < 0
			}
			return a.ID < b.ID
		})

		cards = append(cards, cardFrom(name, printings))
	}

	sort.Slice(cards, func(i, j int) bool {
		a, b := strings.ToLower(cards[i].Name), strings.ToLower(cards[j].Name)
		if a != b {
			return a < b
		}
		return cards[i].ID < cards[j].ID
	})

	return cards
}

func cardFrom(name string, printings []SourcePrinting) Card {
	first := printings[0]

	card := Card{
		ID:            first.ID,
		Name:          name,
		Type:          first.Type,
		Cost:          first.Cost,
		Power:         first.Power,
		Civilizations: nonNil(first.Civilizations),
		Races:         nonNil(first.Races),
		RulesText:     first.RulesText,
		Abilities:     make([]Ability, 0, len(first.Abilities)),
		Keywords:      nonNil(first.Keywords),
		Categories:    nonNil(first.Categories),
		Printings:     make([]Printing, 0, len(printings)),
	}
	if first.PowerText != nil {
		card.PowerText = *first.PowerText
	}

	for _, ability := range first.Abilities {
		line := Ability{Text: ability.Text}
		if ability.Reminder != nil {
			line.Reminder = *ability.Reminder
		}
		card.Abilities = append(card.Abilities, line)
	}

	for _, printing := range printings {
		kept := Printing{
			ID:              printing.ID,
			SetCode:         printing.SetCode,
			CollectorNumber: printing.CollectorNumber,
			Rarity:          printing.Rarity,
			RarityCode:      printing.RarityCode,
		}
		if printing.HasImage {
			kept.ImagePath = printing.ImagePath
		}
		card.Printings = append(card.Printings, kept)
	}

	return card
}

// linkEngineCards sets the engine id of every card the engine can play,
// matching names exactly first and then ignoring case, accents, spaces and
// punctuation.
func linkEngineCards(cards []Card, playable []games.PlayableCard) []string {
	byName := make(map[string]int, len(cards))
	byKey := make(map[string][]int, len(cards))
	for i, card := range cards {
		byName[card.Name] = i
		key := nameKey(card.Name)
		byKey[key] = append(byKey[key], i)
	}

	var problems []string
	for _, engineCard := range playable {
		name := engineCard.Name
		if alias, ok := engineNameAliases[name]; ok {
			name = alias
		}

		index, found := byName[name]
		if !found {
			switch matches := byKey[nameKey(name)]; len(matches) {
			case 0:
				problems = append(problems, fmt.Sprintf("the engine's %q (%s) is not in the card database", engineCard.Name, engineCard.EngineID))
				continue
			case 1:
				index = matches[0]
			default:
				problems = append(problems, fmt.Sprintf("the engine's %q (%s) matches several cards in the card database", engineCard.Name, engineCard.EngineID))
				continue
			}
		}

		if linked := cards[index].EngineID; linked != "" {
			problems = append(problems, fmt.Sprintf("the engine's %q (%s) and %s are both %q in the card database", engineCard.Name, engineCard.EngineID, linked, cards[index].Name))
			continue
		}
		cards[index].EngineID = engineCard.EngineID
	}

	return problems
}

// nameKey is a name with case, accents, spaces and punctuation removed:
// "Überdragon Zaschack" and "Uberdragon Zaschack" share "uberdragonzaschack".
func nameKey(name string) string {
	stripped, _, err := transform.String(transform.Chain(norm.NFKD, runes.Remove(runes.In(unicode.Mn)), norm.NFC), name)
	if err != nil {
		stripped = name
	}

	var key strings.Builder
	for _, r := range strings.ToLower(stripped) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			key.WriteRune(r)
		}
	}

	return key.String()
}

// compareCollectorNumbers orders collector numbers such as "69/110", "S1/S10"
// and "L6/Y2" by their first number, then as text, so "S2" comes before "S10".
func compareCollectorNumbers(a, b string) int {
	if na, nb := firstNumber(a), firstNumber(b); na != nb {
		if na < nb {
			return -1
		}
		return 1
	}

	return strings.Compare(a, b)
}

// firstNumber is the first run of digits in s, or -1 when it has none.
func firstNumber(s string) int {
	start := strings.IndexFunc(s, unicode.IsDigit)
	if start < 0 {
		return -1
	}

	end := start
	for end < len(s) && s[end] >= '0' && s[end] <= '9' {
		end++
	}

	n, err := strconv.Atoi(s[start:end])
	if err != nil {
		return -1
	}

	return n
}

func nonNil(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}
