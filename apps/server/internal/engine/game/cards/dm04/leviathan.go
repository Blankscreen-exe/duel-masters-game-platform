package dm04

import (
	"fmt"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// KingAquakamui ...
func KingAquakamui(c *match.Card) {

	c.Name = "King Aquakamui"
	c.Power = 5000
	c.Civs = []string{civ.Water}
	c.Family = []string{family.Leviathan}
	c.ManaCost = 7
	c.ManaRequirement = []string{civ.Water}

	c.Use(fx.Creature,
		fx.When(fx.Summoned, func(card *match.Card, ctx *match.Context) {

			fx.FindFilter(
				card.Player,
				match.GRAVEYARD,
				func(x *match.Card) bool { return x.HasFamily(family.AngelCommand) || x.HasFamily(family.DemonCommand) },
			).Map(func(x *match.Card) {

				x.Player.MoveCard(x.ID, match.GRAVEYARD, match.HAND, card.ID)
				ctx.Match.ReportActionInChat(card.Player, fmt.Sprintf("%s was moved to %s's hand from their graveyard by King Aquakamui", x.Name, card.Player.Username()))
			})

		}),
		func(card *match.Card, ctx *match.Context) {

			if card.Zone != match.BATTLEZONE {
				return
			}

			if event, ok := ctx.Event.(*match.GetPowerEvent); ok {

				if event.Card.HasFamily(family.AngelCommand) || event.Card.HasFamily(family.DemonCommand) {
					event.Power += 2000
				}
			}
		})
}
