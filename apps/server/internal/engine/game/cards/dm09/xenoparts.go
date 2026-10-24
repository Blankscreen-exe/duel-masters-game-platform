package dm09

import (
	"fmt"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/cnd"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// GigiosHammer ...
func GigiosHammer(c *match.Card) {

	c.Name = "Gigio's Hammer"
	c.Power = 2000
	c.Civs = []string{civ.Fire}
	c.Family = []string{family.Xenoparts}
	c.ManaCost = 3
	c.ManaRequirement = []string{civ.Fire}
	c.TapAbility = gigiosHammerTapAbility

	c.Use(fx.Creature, fx.TapAbility)
}

func gigiosHammerTapAbility(card *match.Card, ctx *match.Context) {
	family := fx.ChooseAFamily(card, ctx, fmt.Sprintf("%s's effect: Choose a race. Each creature of that race attacks this turn if able and gets 'Power attacker +4000'.", card.Name))

	if family != "" {
		ctx.Match.ReportActionInChat(card.Player, fmt.Sprintf("This turn, each '%s' creature attacks if able and gets 'Power attacker +4000'.", family))

		ctx.Match.ApplyPersistentEffect(func(ctx2 *match.Context, exit func()) {
			fx.FindFilter(
				card.Player,
				match.BATTLEZONE,
				func(x *match.Card) bool {
					return x.HasFamily(family) && x.HasCondition(cnd.Creature)
				},
			).Map(func(x *match.Card) {
				if _, ok := ctx2.Event.(*match.EndOfTurnStep); ok {
					x.RemoveConditionBySource(card.ID)
					exit()
					return
				}

				x.AddUniqueSourceCondition(cnd.PowerAttacker, 4000, card.ID)
				fx.ForceAttack(x, ctx2)
			})
		})
	}
}
