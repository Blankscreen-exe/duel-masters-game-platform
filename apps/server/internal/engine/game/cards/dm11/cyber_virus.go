package dm11

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// LuckyBall ...
func LuckyBall(c *match.Card) {

	c.Name = "Lucky Ball"
	c.Power = 3000
	c.Civs = []string{civ.Water}
	c.Family = []string{family.CyberVirus}
	c.ManaCost = 4
	c.ManaRequirement = []string{civ.Water}

	c.Use(fx.Creature, fx.When(fx.Summoned, func(card *match.Card, ctx *match.Context) {
		shields, err := ctx.Match.Opponent(card.Player).Container(match.SHIELDZONE)

		if err != nil || len(shields) > 3 {
			return
		}

		fx.DrawUpTo2(card, ctx)
	}))

}
