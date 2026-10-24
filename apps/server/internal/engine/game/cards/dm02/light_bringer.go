package dm02

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// FonchTheOracle ...
func FonchTheOracle(c *match.Card) {

	c.Name = "Fonch, the Oracle"
	c.Power = 2000
	c.Civs = []string{civ.Light}
	c.Family = []string{family.LightBringer}
	c.ManaCost = 4
	c.ManaRequirement = []string{civ.Light}

	c.Use(fx.Creature, fx.When(fx.Summoned, fx.TapOpCreature))

}

// WynTheOracle ...
func WynTheOracle(c *match.Card) {

	c.Name = "Wyn, the Oracle"
	c.Power = 1500
	c.Civs = []string{civ.Light}
	c.Family = []string{family.LightBringer}
	c.ManaCost = 2
	c.ManaRequirement = []string{civ.Light}

	c.Use(fx.Creature, fx.ShieldsSelectionEffect, fx.WheneverThisAttacksMayLookAtOpShield())

}
