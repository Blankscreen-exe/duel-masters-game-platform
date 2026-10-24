package dm07

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/cnd"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

func ScalpelSpider(c *match.Card) {

	c.Name = "Scalpel Spider"
	c.Power = 2000
	c.Civs = []string{civ.Darkness}
	c.Family = []string{family.BrainJacker}
	c.ManaCost = 3
	c.ManaRequirement = []string{civ.Darkness}

	c.Use(fx.Creature, fx.When(fx.Attacked, func(c *match.Card, ctx *match.Context) {
		c.AddCondition(cnd.Slayer, nil, c.ID)
	}))

}
