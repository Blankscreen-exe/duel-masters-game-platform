package dm01

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// LahPurificationEnforcer ...
func LahPurificationEnforcer(c *match.Card) {

	c.Name = "Lah, Purification Enforcer"
	c.Power = 5500
	c.Civs = []string{civ.Light}
	c.Family = []string{family.Berserker}
	c.ManaCost = 5
	c.ManaRequirement = []string{civ.Light}

	c.Use(fx.Creature)

}

// RaylaTruthEnforcer ...
func RaylaTruthEnforcer(c *match.Card) {

	c.Name = "Rayla, Truth Enforcer"
	c.Power = 3000
	c.Civs = []string{civ.Light}
	c.Family = []string{family.Berserker}
	c.ManaCost = 6
	c.ManaRequirement = []string{civ.Light}

	c.Use(fx.Creature, fx.When(fx.Summoned, fx.SearchDeckTake1Spell))

}
