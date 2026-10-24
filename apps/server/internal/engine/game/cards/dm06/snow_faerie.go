package dm06

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

func CharmiliaTheEnticer(c *match.Card) {

	c.Name = "Charmilia, the Enticer"
	c.Power = 3000
	c.Civs = []string{civ.Nature}
	c.Family = []string{family.SnowFaerie}
	c.ManaCost = 4
	c.ManaRequirement = []string{civ.Nature}
	c.TapAbility = fx.SearchDeckTake1Creature

	c.Use(fx.Creature, fx.TapAbility)
}

func GarabonTheGlider(c *match.Card) {

	c.Name = "Garabon, the Glider"
	c.Power = 1000
	c.Civs = []string{civ.Nature}
	c.Family = []string{family.SnowFaerie}
	c.ManaCost = 2
	c.ManaRequirement = []string{civ.Nature}

	c.Use(fx.Creature, fx.PowerAttacker2000)
}
