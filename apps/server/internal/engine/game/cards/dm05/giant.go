package dm05

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// AvalancheGiant ...
func AvalancheGiant(c *match.Card) {

	c.Name = "Avalanche Giant"
	c.Power = 8000
	c.Civs = []string{civ.Nature}
	c.Family = []string{family.Giant}
	c.ManaCost = 6
	c.ManaRequirement = []string{civ.Nature}

	c.Use(fx.Creature, fx.Doublebreaker, fx.CantAttackCreatures, fx.When(fx.Blocked, fx.DestroyOpShield))

}

// NocturnalGiant ...
func NocturnalGiant(c *match.Card) {

	c.Name = "Nocturnal Giant"
	c.Power = 7000
	c.Civs = []string{civ.Nature}
	c.Family = []string{family.Giant}
	c.ManaCost = 7
	c.ManaRequirement = []string{civ.Nature}

	c.Use(fx.Creature, fx.Triplebreaker, fx.CantAttackCreatures, func(card *match.Card, ctx *match.Context) {
		fx.PowerAttacker(card, ctx, 7000)
	})
}
