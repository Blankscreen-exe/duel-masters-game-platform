package dm07

import (
	"fmt"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

func GandarSeekerofExplosions(c *match.Card) {

	c.Name = "Gandar, Seeker of Explosions"
	c.Power = 6500
	c.Civs = []string{civ.Light}
	c.Family = []string{family.MechaThunder}
	c.ManaCost = 7
	c.ManaRequirement = []string{civ.Light}
	c.TapAbility = func(card *match.Card, ctx *match.Context) {
		ctx.Match.ApplyPersistentEffect(func(ctx2 *match.Context, exit func()) {
			if _, ok := ctx2.Event.(*match.EndOfTurnStep); ok {
				fx.Find(
					card.Player,
					match.BATTLEZONE,
				).Map(func(x *match.Card) {
					if x.Tapped && x.HasCiv(civ.Light) {
						x.Tapped = false
						ctx2.Match.ReportActionInChat(card.Player, fmt.Sprintf("%s was untapped by %s's effect", x.Name, card.Name))
					}
				})
				exit()
			}
		})
	}

	c.Use(fx.Creature, fx.Doublebreaker, fx.TapAbility)

}
