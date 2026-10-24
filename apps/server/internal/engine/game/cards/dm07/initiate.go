package dm07

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

func KizarBasikuTheOutrageous(c *match.Card) {

	c.Name = "Kizar Basiku, the Outrageous"
	c.Power = 8500
	c.Civs = []string{civ.Light}
	c.Family = []string{family.Initiate}
	c.ManaCost = 5
	c.ManaRequirement = []string{civ.Light}

	c.Use(fx.Creature, fx.Blocker(), fx.Evolution, fx.FireStealth, fx.Doublebreaker)
}

func RomVizierofTendrils(c *match.Card) {

	c.Name = "Rom, Vizier of Tendrils"
	c.Power = 2000
	c.Civs = []string{civ.Light}
	c.Family = []string{family.Initiate}
	c.ManaCost = 4
	c.ManaRequirement = []string{civ.Light}

	c.Use(fx.Creature, fx.When(fx.Summoned, fx.MayTapOpCreature))
}
