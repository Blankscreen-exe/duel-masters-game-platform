package promo

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// AmnisHolyElemental ...
func AmnisHolyElemental(c *match.Card) {

	c.Name = "Amnis, Holy Elemental"
	c.Power = 5000
	c.Civs = []string{civ.Light}
	c.Family = []string{family.AngelCommand}
	c.ManaCost = 7
	c.ManaRequirement = []string{civ.Light}

	c.Use(fx.Creature,
		fx.DarknessBlocker(),
		func(card *match.Card, ctx *match.Context) {

			if event, ok := ctx.Event.(*match.CreatureDestroyed); ok && event.Card == card {
				if event.Context == match.DestroyedInBattle && event.Source.HasCiv(civ.Darkness) {
					ctx.InterruptFlow()
				}
			}

		})

}
