package dm02

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// ScissorEye ...
func ScissorEye(c *match.Card) {

	c.Name = "Scissor Eye"
	c.Power = 3000
	c.Civs = []string{civ.Water}
	c.Family = []string{family.GelFish}
	c.ManaCost = 4
	c.ManaRequirement = []string{civ.Water}

	c.Use(fx.Creature)

}

// PlasmaChaser ...
func PlasmaChaser(c *match.Card) {

	c.Name = "Plasma Chaser"
	c.Power = 4000
	c.Civs = []string{civ.Water}
	c.Family = []string{family.GelFish}
	c.ManaCost = 6
	c.ManaRequirement = []string{civ.Water}

	c.Use(fx.Creature, fx.When(fx.AttackConfirmed, func(card *match.Card, ctx *match.Context) {

		toDraw := len(fx.Find(ctx.Match.Opponent(card.Player), match.BATTLEZONE))

		if toDraw < 1 {
			return
		}

		fx.MayDrawAmount(card, ctx, toDraw)

	}))

}
