package dm10

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// ClearloGraceEnforcer ...
func ClearloGraceEnforcer(c *match.Card) {

	c.Name = "Clearlo, Grace Enforcer"
	c.Power = 1000
	c.Civs = []string{civ.Light}
	c.Family = []string{family.Berserker}
	c.ManaCost = 3
	c.ManaRequirement = []string{civ.Light}

	// Get +1000 power for each of your other untapped creatures in the battle zone
	c.PowerModifier = func(m *match.Match, attacking bool) int {
		return getYourOtherUntappedCreatures(c) * 1000
	}

	c.Use(fx.Creature)

}

// Return the number of your other untapped creatures in the battle zone
func getYourOtherUntappedCreatures(card *match.Card) int {
	battleZone, err := card.Player.Container(match.BATTLEZONE)

	if err != nil {
		return 0
	}

	count := 0

	for _, battleZoneCard := range battleZone {
		if battleZoneCard.ID != card.ID && !battleZoneCard.Tapped {
			count++
		}
	}

	return count
}
