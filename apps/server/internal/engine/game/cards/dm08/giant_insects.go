package dm08

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// CarboniteScarab ...
func CarboniteScarab(c *match.Card) {

	c.Name = "Carbonite Scarab"
	c.Power = 3000
	c.Civs = []string{civ.Nature}
	c.Family = []string{family.GiantInsect}
	c.ManaCost = 4
	c.ManaRequirement = []string{civ.Nature}

	turboRush := false

	c.Use(
		fx.Creature,
		fx.When(fx.TurboRushCondition, func(card *match.Card, ctx *match.Context) { turboRush = true }),
		fx.When(fx.EndOfMyTurn, func(card *match.Card, ctx *match.Context) {
			turboRush = false
		}),
		fx.WhenAll([]func(*match.Card, *match.Context) bool{func(card *match.Card, ctx *match.Context) bool { return turboRush }, fx.WheneverThisAttacksPlayerAndBecomesBlocked},
			func(card *match.Card, ctx *match.Context) {
				fx.DestroyOpShield(card, ctx)
			}))

}
