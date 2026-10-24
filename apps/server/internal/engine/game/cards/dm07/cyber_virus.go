package dm07

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

func CosmicNebula(c *match.Card) {

	c.Name = "Cosmic Nebula"
	c.Power = 3000
	c.Civs = []string{civ.Water}
	c.Family = []string{family.CyberVirus}
	c.ManaCost = 5
	c.ManaRequirement = []string{civ.Water}

	c.Use(fx.Creature, fx.Evolution, fx.When(fx.MyDrawStep, func(card *match.Card, ctx *match.Context) {
		if card.Zone == match.BATTLEZONE {
			fx.MayDraw1(card, ctx)
		}
	}))

}

func CuriousEye(c *match.Card) {

	c.Name = "Curious Eye"
	c.Power = 1000
	c.Civs = []string{civ.Water}
	c.Family = []string{family.CyberVirus}
	c.ManaCost = 3
	c.ManaRequirement = []string{civ.Water}

	c.Use(fx.Creature, fx.ShieldsSelectionEffect, fx.WheneverThisAttacksMayLookAtOpShield())

}
