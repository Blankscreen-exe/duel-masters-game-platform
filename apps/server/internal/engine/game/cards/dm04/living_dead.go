package dm04

import (
	"fmt"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// SkeletonThiefTheRevealer ...
func SkeletonThiefTheRevealer(c *match.Card) {

	c.Name = "Skeleton Thief, the Revealer"
	c.Power = 2000
	c.Civs = []string{civ.Darkness}
	c.Family = []string{family.LivingDead}
	c.ManaCost = 4
	c.ManaRequirement = []string{civ.Darkness}

	c.Use(fx.Creature, fx.When(fx.Summoned, func(card *match.Card, ctx *match.Context) {
		fx.SelectFilter(
			card.Player,
			ctx.Match,
			card.Player,
			match.GRAVEYARD,
			fmt.Sprintf("%s: You may return a Living Dead from your graveyard to your hand", card.Name),
			1,
			1,
			true,
			func(x *match.Card) bool { return x.HasFamily(family.LivingDead) },
			false,
		).Map(func(x *match.Card) {
			x.Player.MoveCard(x.ID, match.GRAVEYARD, match.HAND, card.ID)
			ctx.Match.ReportActionInChat(x.Player, fmt.Sprintf("%s was moved to %s's hand from their graveyard by %s", x.Name, card.Player.Username(), card.Name))
		})

	}))

}
