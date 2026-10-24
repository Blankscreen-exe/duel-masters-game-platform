package dm09

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// MarchingMotherboard ...
func MarchingMotherboard(c *match.Card) {

	c.Name = "Marching Motherboard"
	c.Power = 2000
	c.Civs = []string{civ.Water}
	c.Family = []string{family.CyberVirus}
	c.ManaCost = 6
	c.ManaRequirement = []string{civ.Water}

	c.Use(fx.Creature,
		fx.When(fx.AnotherOwnCyberSummoned, func(card *match.Card, ctx *match.Context) {
			fx.MayDraw1(card, ctx)
		}))
}

// KelpCandle ...
func KelpCandle(c *match.Card) {

	c.Name = "Kelp Candle"
	c.Power = 1000
	c.Civs = []string{civ.Water}
	c.Family = []string{family.CyberVirus}
	c.ManaCost = 2
	c.ManaRequirement = []string{civ.Water}

	c.Use(fx.Creature, fx.Blocker(), fx.CantAttackCreatures, fx.CantAttackPlayers,
		fx.When(fx.Blocks, fx.LookTop4Put1IntoHandReorderRestOnBottomDeck))
}
