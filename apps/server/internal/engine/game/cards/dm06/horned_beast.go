package dm06

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

func FeatherHornTheTracker(c *match.Card) {

	c.Name = "Feather Horn, the Tracker"
	c.Power = 4000
	c.Civs = []string{civ.Nature}
	c.Family = []string{family.HornedBeast}
	c.ManaCost = 4
	c.ManaRequirement = []string{civ.Nature}

	c.Use(fx.Creature)
}

func ParadiseHorn(c *match.Card) {

	c.Name = "Paradise Horn"
	c.Power = 3000
	c.Civs = []string{civ.Nature}
	c.Family = []string{family.HornedBeast}
	c.ManaCost = 4
	c.ManaRequirement = []string{civ.Nature}

	c.Use(fx.Creature, fx.PowerAttacker2000)
}
