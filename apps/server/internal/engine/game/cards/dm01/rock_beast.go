package dm01

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// Meteosaur ...
func Meteosaur(c *match.Card) {

	c.Name = "Meteosaur"
	c.Power = 2000
	c.Civs = []string{civ.Fire}
	c.Family = []string{family.RockBeast}
	c.ManaCost = 5
	c.ManaRequirement = []string{civ.Fire}

	c.Use(fx.Creature, fx.When(fx.Summoned, fx.DestroyByMiscOpCreature2000OrLess))

}

// Stonesaur ...
func Stonesaur(c *match.Card) {

	c.Name = "Stonesaur"
	c.Power = 4000
	c.Civs = []string{civ.Fire}
	c.Family = []string{family.RockBeast}
	c.ManaCost = 5
	c.ManaRequirement = []string{civ.Fire}

	c.Use(fx.Creature, fx.PowerAttacker2000)

}
