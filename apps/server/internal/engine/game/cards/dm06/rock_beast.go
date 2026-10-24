package dm06

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/cnd"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

func RumblesaurQ(c *match.Card) {
	c.Name = "Rumblesaur Q"
	c.Power = 3000
	c.Civs = []string{civ.Fire}
	c.Family = []string{family.RockBeast, family.Survivor}
	c.ManaCost = 6
	c.ManaRequirement = []string{civ.Fire}

	c.Use(fx.Creature, fx.Survivor,
		fx.When(fx.InTheBattlezone, func(card *match.Card, ctx *match.Context) {
			ctx.Match.ApplyPersistentEffect(func(ctx2 *match.Context, exit func()) {
				if card.Zone != match.BATTLEZONE {
					fx.Find(
						card.Player,
						match.BATTLEZONE,
					).Map(func(x *match.Card) {
						x.RemoveConditionBySource(card.ID)
					})

					exit()
					return
				}

				fx.FindFilter(
					card.Player,
					match.BATTLEZONE,
					func(creature *match.Card) bool { return creature.HasFamily(family.Survivor) },
				).Map(func(x *match.Card) { x.AddUniqueSourceCondition(cnd.SpeedAttacker, true, card.ID) })
			})
		}),
	)
}
