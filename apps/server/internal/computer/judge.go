package computer

import (
	"slices"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/cnd"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/server"
)

// keywordValue is what each keyword ability adds to a card's worth.
var keywordValue = map[string]float64{
	"blocker":        1,
	"speed_attack":   1,
	"double_break":   1.5,
	"triple_break":   2.5,
	"slayer":         1,
	"power_attack":   0.5,
	"shield_trigger": 0.5,
}

// categoryValue is what an effect adds to a card's worth.
var categoryValue = map[string]float64{
	"removal":     2,
	"draw":        1.5,
	"search":      1,
	"discard_opp": 1,
	"tap_opp":     1,
	"ramp":        1,
	"trigger_etb": 0.5,
}

// worth is how much a card is worth to have, from what is printed on it:
// higher for more power, cost and useful abilities.
func worth(card Card) float64 {
	value := float64(card.Cost) * 0.5
	if card.IsCreature() {
		value += float64(card.Power) / 1000
	}
	for keyword, bonus := range keywordValue {
		if card.Has(keyword) {
			value += bonus
		}
	}
	for category, bonus := range categoryValue {
		if card.Does(category) {
			value += bonus
		}
	}
	return value
}

// board is one side of the battle zone, as the computer reads it.
type board struct {
	side *server.PlayerState
}

func (b board) power(cardID string) int {
	return b.side.Powers[cardID].Current
}

func (b board) attackingPower(cardID string) int {
	return b.side.Powers[cardID].Attacking
}

func (b board) has(cardID, status string) bool {
	return slices.ContainsFunc(b.side.Statuses[cardID], func(s server.CreatureStatus) bool { return s.ID == status })
}

// untappedBlockers are the creatures that could block an attack now.
func (b board) untappedBlockers() []server.CardState {
	var blockers []server.CardState
	for _, creature := range b.side.Battlezone {
		if creature.Flags&uint8(match.TappedFlag) == 0 && b.has(creature.CardID, cnd.Blocker) {
			blockers = append(blockers, creature)
		}
	}
	return blockers
}

// creatureWorth is what a creature in the battle zone is worth now, with its
// power as it is rather than as printed.
func (b board) creatureWorth(library *Library, creature server.CardState) float64 {
	value := float64(b.power(creature.CardID)) / 1000
	if card, ok := library.Card(creature.ImageID); ok {
		value += worth(card) - float64(card.Power)/1000
	}
	return value
}
