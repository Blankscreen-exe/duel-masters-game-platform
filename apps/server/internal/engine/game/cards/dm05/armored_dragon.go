package dm05

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// BolgashDragon ...
func BolgashDragon(c *match.Card) {

	c.Name = "Bolgash Dragon"
	c.Power = 4000
	c.Civs = []string{civ.Fire}
	c.Family = []string{family.ArmoredDragon}
	c.ManaCost = 8
	c.ManaRequirement = []string{civ.Fire}

	c.Use(fx.Creature, fx.Triplebreaker, fx.PowerAttacker8000)

}

// BillionDegreeDragon ...
func BillionDegreeDragon(c *match.Card) {

	c.Name = "Billion-Degree Dragon"
	c.Power = 15000
	c.Civs = []string{civ.Fire}
	c.Family = []string{family.ArmoredDragon}
	c.ManaCost = 10
	c.ManaRequirement = []string{civ.Fire}

	c.Use(fx.Creature, fx.Triplebreaker)

}
