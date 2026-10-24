package dm05

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

func WispHowlerShadowOfTears(c *match.Card) {
	c.Name = "Wisp Howler, Shadow of Tears"
	c.Power = 2000
	c.Civs = []string{civ.Darkness}
	c.Family = []string{family.Ghost}
	c.ManaCost = 3
	c.ManaRequirement = []string{civ.Darkness}

	c.Use(fx.Creature, fx.ConditionalSlayer(func(target *match.Card) bool {
		return target.HasCiv(civ.Nature) || target.HasCiv(civ.Light)
	}))
}
