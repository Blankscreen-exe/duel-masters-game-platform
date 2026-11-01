package computer

import (
	"slices"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/cards"
)

// Card types, as the card database names them.
const (
	typeCreature          = "Creature"
	typeEvolutionCreature = "Evolution Creature"
	typeSpell             = "Spell"
)

// Card is what the computer knows of a card: what is printed on it, which
// every player may know, from the card database.
type Card struct {
	EngineID      string
	Name          string
	Type          string
	Cost          int
	Power         int
	Civilizations []string
	// Text is its rules text.
	Text       string
	keywords   []string
	categories []string
}

// IsCreature reports whether the card is a creature, evolution or not.
func (c Card) IsCreature() bool {
	return c.Type == typeCreature || c.Type == typeEvolutionCreature
}

// IsEvolution reports whether the card is an evolution creature.
func (c Card) IsEvolution() bool { return c.Type == typeEvolutionCreature }

// IsSpell reports whether the card is a spell.
func (c Card) IsSpell() bool { return c.Type == typeSpell }

// Has reports whether the card has a keyword ability, such as "blocker" or
// "shield_trigger", as the card database names them.
func (c Card) Has(keyword string) bool { return slices.Contains(c.keywords, keyword) }

// Does reports whether the card's effect falls in a category, such as
// "removal" or "discard_opp", as the card database names them.
func (c Card) Does(category string) bool { return slices.Contains(c.categories, category) }

// Library is every card the computer may meet, by the engine's id for it.
// It never changes once made, so it is safe for concurrent use.
type Library struct {
	byEngineID map[string]Card
	byName     map[string]Card
}

// NewLibrary returns the library of the cards the engine can play.
func NewLibrary(catalogue []cards.Card) *Library {
	l := &Library{byEngineID: map[string]Card{}, byName: map[string]Card{}}
	for _, card := range catalogue {
		if !card.Playable() {
			continue
		}

		power := 0
		if card.Power != nil {
			power = *card.Power
		}
		known := Card{
			EngineID:      card.EngineID,
			Name:          card.Name,
			Type:          card.Type,
			Cost:          card.Cost,
			Power:         power,
			Civilizations: slices.Clone(card.Civilizations),
			Text:          card.RulesText,
			keywords:      slices.Clone(card.Keywords),
			categories:    slices.Clone(card.Categories),
		}
		l.byEngineID[card.EngineID] = known
		l.byName[card.Name] = known
	}
	return l
}

// Card returns the card the engine knows by the id, which the engine sends
// as a card's "uid".
func (l *Library) Card(engineID string) (Card, bool) {
	card, ok := l.byEngineID[engineID]
	return card, ok
}

// Named returns the card with the name.
func (l *Library) Named(name string) (Card, bool) {
	card, ok := l.byName[name]
	return card, ok
}
