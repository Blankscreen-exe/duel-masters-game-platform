package dm02

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// SpiralGrass ...
func SpiralGrass(c *match.Card) {

	c.Name = "Spiral Grass"
	c.Power = 2500
	c.Civs = []string{civ.Light}
	c.Family = []string{family.StarlightTree}
	c.ManaCost = 4
	c.ManaRequirement = []string{civ.Light}

	c.Use(fx.Creature, fx.Blocker(), func(card *match.Card, ctx *match.Context) {
		if event, ok := ctx.Event.(*match.Battle); ok {
			if event.Defender == card && event.Blocked {
				ctx.ScheduleAfter(func() {
					if card.Zone == match.BATTLEZONE {
						card.Tapped = false
						ctx.Match.BroadcastState()
					}
				})
			}
		}
	})

}
