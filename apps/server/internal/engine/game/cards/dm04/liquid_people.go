package dm04

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// AquaGuard ...
func AquaGuard(c *match.Card) {

	c.Name = "Aqua Guard"
	c.Power = 2000
	c.Civs = []string{civ.Water}
	c.Family = []string{family.LiquidPeople}
	c.ManaCost = 1
	c.ManaRequirement = []string{civ.Water}

	c.Use(fx.Creature, fx.Blocker(), fx.CantAttackCreatures, fx.CantAttackPlayers)
}

// AquaJolter ...
func AquaJolter(c *match.Card) {

	c.Name = "Aqua Jolter"
	c.Power = 2000
	c.Civs = []string{civ.Water}
	c.Family = []string{family.LiquidPeople}
	c.ManaCost = 3
	c.ManaRequirement = []string{civ.Water}

	c.Use(fx.Creature, fx.ShieldTrigger)
}
