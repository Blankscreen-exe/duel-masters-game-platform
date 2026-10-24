package dm03

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// RazaVegaThunderGuardian ...
func RazaVegaThunderGuardian(c *match.Card) {

	c.Name = "Raza Vega, Thunder Guardian"
	c.Power = 3000
	c.Civs = []string{civ.Light}
	c.Family = []string{family.Guardian}
	c.ManaCost = 10
	c.ManaRequirement = []string{civ.Light}

	c.Use(fx.Creature, fx.Blocker(), fx.ReturnToShield)
}
