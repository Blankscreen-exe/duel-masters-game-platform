package dm02

import (
	"fmt"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// StainedGlass ...
func StainedGlass(c *match.Card) {

	c.Name = "Stained Glass"
	c.Power = 1000
	c.Civs = []string{civ.Water}
	c.Family = []string{family.CyberVirus}
	c.ManaCost = 3
	c.ManaRequirement = []string{civ.Water}

	c.Use(fx.Creature, fx.When(fx.AttackConfirmed, func(card *match.Card, ctx *match.Context) {

		fx.SelectFilter(
			card.Player,
			ctx.Match,
			ctx.Match.Opponent(card.Player),
			match.BATTLEZONE,
			fmt.Sprintf("%s: You may select 1 of your opponent's fire or nature creatures that will be returned to their hand", card.Name),
			1,
			1,
			true,
			func(x *match.Card) bool { return x.HasCiv(civ.Fire) || x.HasCiv(civ.Nature) },
			false,
		).Map(func(x *match.Card) {
			x.Player.MoveCard(x.ID, match.BATTLEZONE, match.HAND, card.ID)
			ctx.Match.ReportActionInChat(ctx.Match.Opponent(card.Player), fmt.Sprintf("%s was sent to %s's hand from the battle zone by %s's effect", x.Name, x.Player.Username(), card.Name))
		})

	}))

}
