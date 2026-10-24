package dm02

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// MarrowOozeTheTwister ...
func MarrowOozeTheTwister(c *match.Card) {

	c.Name = "Marrow Ooze, the Twister"
	c.Power = 1000
	c.Civs = []string{civ.Darkness}
	c.Family = []string{family.LivingDead}
	c.ManaCost = 1
	c.ManaRequirement = []string{civ.Darkness}

	c.Use(fx.Creature, fx.Blocker(), fx.When(fx.AttackingPlayer, func(card *match.Card, ctx *match.Context) {

		ctx.ScheduleAfter(func() {
			ctx.Match.Destroy(card, card, match.DestroyedByMiscAbility)
		})

	}))

}
