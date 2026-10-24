package dm02

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// EngineerKipo ...
func EngineerKipo(c *match.Card) {

	c.Name = "Engineer Kipo"
	c.Power = 2000
	c.Civs = []string{civ.Fire}
	c.Family = []string{family.MachineEater}
	c.ManaCost = 2
	c.ManaRequirement = []string{civ.Fire}

	c.Use(fx.Creature, fx.When(fx.Destroyed, fx.EachPlayerDestroys1Mana))

}
