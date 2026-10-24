package dm04

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// AstralWarper ...
func AstralWarper(c *match.Card) {

	c.Name = "Astral Warper"
	c.Power = 5000
	c.Civs = []string{civ.Water}
	c.Family = []string{family.CyberVirus}
	c.ManaCost = 6
	c.ManaRequirement = []string{civ.Water}

	c.Use(fx.Creature, fx.Evolution, fx.When(fx.Summoned, fx.DrawUpTo3))
}

// KeeperOfTheSunlitAbyss ...
func KeeperOfTheSunlitAbyss(c *match.Card) {

	c.Name = "Keeper of the Sunlit Abyss"
	c.Power = 2000
	c.Civs = []string{civ.Water}
	c.Family = []string{family.CyberVirus}
	c.ManaCost = 4
	c.ManaRequirement = []string{civ.Water}

	c.Use(fx.Creature, func(card *match.Card, ctx *match.Context) {

		if card.Zone != match.BATTLEZONE {
			return
		}

		if event, ok := ctx.Event.(*match.GetPowerEvent); ok {

			if event.Card.HasCiv(civ.Light) || event.Card.HasCiv(civ.Darkness) {
				event.Power += 1000
			}
		}
	})
}
