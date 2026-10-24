package dm10

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/cnd"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// MykeesPliers ...
func MykeesPliers(c *match.Card) {

	c.Name = "Mykee's Pliers"
	c.Power = 2000
	c.Civs = []string{civ.Fire}
	c.Family = []string{family.Xenoparts}
	c.ManaCost = 4
	c.ManaRequirement = []string{civ.Fire}

	c.Use(fx.Creature, fx.When(fx.InTheBattlezone, func(card *match.Card, ctx *match.Context) {
		ctx.Match.ApplyPersistentEffect(func(ctx2 *match.Context, exit func()) {
			if card.Zone != match.BATTLEZONE {
				fx.FindFilter(
					card.Player,
					match.BATTLEZONE,
					func(x *match.Card) bool {
						return x.HasCiv(civ.Darkness) || x.HasCiv(civ.Nature)
					},
				).Map(func(x *match.Card) {
					x.RemoveConditionBySource(card.ID)
				})

				exit()
				return
			}

			fx.FindFilter(
				card.Player,
				match.BATTLEZONE,
				func(x *match.Card) bool {
					return x.HasCiv(civ.Darkness) || x.HasCiv(civ.Nature)
				},
			).Map(func(x *match.Card) {
				x.AddUniqueSourceCondition(cnd.SpeedAttacker, true, card.ID)
			})
		})
	}))

}

// VorgsEngine ...
func VorgsEngine(c *match.Card) {

	c.Name = "Vorg's Engine"
	c.Power = 2000
	c.Civs = []string{civ.Fire}
	c.Family = []string{family.Xenoparts}
	c.ManaCost = 2
	c.ManaRequirement = []string{civ.Fire}

	c.Use(fx.Creature, fx.SilentSkill(fx.DestroyAllCreaturesXPowerOrLess(2000, match.DestroyedByMiscAbility)))

}
