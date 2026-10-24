package dm08

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// GrapeGlobbo ...
func GrapeGlobbo(c *match.Card) {

	c.Name = "Grape Globbo"
	c.Power = 1000
	c.Civs = []string{civ.Water}
	c.Family = []string{family.CyberVirus}
	c.ManaCost = 2
	c.ManaRequirement = []string{civ.Water}

	c.Use(fx.Creature, fx.When(fx.Summoned, func(card *match.Card, ctx *match.Context) {
		opponentHandIds := make([]string, 0)

		fx.Find(ctx.Match.Opponent(card.Player), match.HAND).Map(func(x *match.Card) {
			opponentHandIds = append(opponentHandIds, x.ImageID)
		})

		ctx.Match.ShowCards(card.Player, "Your opponent's hand:", opponentHandIds)
	}))

}
