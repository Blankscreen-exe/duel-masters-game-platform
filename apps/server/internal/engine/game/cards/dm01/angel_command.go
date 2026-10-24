package dm01

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// HanusaRadianceElemental ...
func HanusaRadianceElemental(c *match.Card) {

	c.Name = "Hanusa, Radiance Elemental"
	c.Power = 9500
	c.Civs = []string{civ.Light}
	c.Family = []string{family.AngelCommand}
	c.ManaCost = 7
	c.ManaRequirement = []string{civ.Light}

	c.Use(fx.Creature, fx.Doublebreaker)

}

// UrthPurifyingElemental ...
func UrthPurifyingElemental(c *match.Card) {

	c.Name = "Urth, Purifying Elemental"
	c.Power = 6000
	c.Civs = []string{civ.Light}
	c.Family = []string{family.AngelCommand}
	c.ManaCost = 6
	c.ManaRequirement = []string{civ.Light}

	c.Use(fx.Creature, fx.Doublebreaker, fx.When(fx.EndOfMyTurnCreatureBZ, fx.MayUntapSelf))

}
