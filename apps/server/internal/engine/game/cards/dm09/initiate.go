package dm09

import (
	"fmt"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// KaluteVizierOfEternity ...
func KaluteVizierOfEternity(c *match.Card) {

	c.Name = "Kalute, Vizier of Eternity"
	c.Power = 1000
	c.Civs = []string{civ.Light}
	c.Family = []string{family.Initiate}
	c.ManaCost = 2
	c.ManaRequirement = []string{civ.Light}

	c.Use(fx.Creature,
		func(card *match.Card, ctx *match.Context) {
			if event, ok := ctx.Event.(*match.CreatureDestroyed); ok {
				if event.Card.ID == card.ID {
					fx.FindFilter(
						card.Player,
						match.BATTLEZONE,
						func(x *match.Card) bool {
							return x.Name == card.Name && x.ID != card.ID
						},
					).Map(func(x *match.Card) {
						ctx.InterruptFlow()

						_, err := card.Player.MoveCard(card.ID, match.BATTLEZONE, match.HAND, card.ID)

						if err != nil {
							ctx.Match.ReportActionInChat(card.Player, fmt.Sprintf("%s was moved to hand instead of being destroyed.", card.Name))
						}
					})
				}
			}
		},
	)

}
