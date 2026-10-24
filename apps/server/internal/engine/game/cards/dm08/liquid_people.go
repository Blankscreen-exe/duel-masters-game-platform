package dm08

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// AquaRanger ...
func AquaRanger(c *match.Card) {

	c.Name = "Aqua Ranger"
	c.Power = 3000
	c.Civs = []string{civ.Water}
	c.Family = []string{family.LiquidPeople}
	c.ManaCost = 6
	c.ManaRequirement = []string{civ.Water}

	c.Use(fx.Creature, fx.CantBeBlocked, fx.When(fx.WouldBeDestroyed, fx.ReturnToHand))

}

// AquaGrappler ...
func AquaGrappler(c *match.Card) {

	c.Name = "Aqua Grappler"
	c.Power = 3000
	c.Civs = []string{civ.Water}
	c.Family = []string{family.LiquidPeople}
	c.ManaCost = 5
	c.ManaRequirement = []string{civ.Water}

	c.Use(fx.Creature, fx.When(fx.AttackConfirmed, func(card *match.Card, ctx *match.Context) {
		numberOfMyOtherTappedCreatures := len(fx.FindFilter(
			card.Player,
			match.BATTLEZONE,
			func(x *match.Card) bool {
				return x.Tapped && x.ID != card.ID
			},
		))

		fx.DrawUpto(card, ctx, numberOfMyOtherTappedCreatures)
	}))

}
