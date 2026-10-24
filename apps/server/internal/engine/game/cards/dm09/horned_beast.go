package dm09

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// SolidHorn ...
func SolidHorn(c *match.Card) {

	c.Name = "Solid Horn"
	c.Power = 5000
	c.Civs = []string{civ.Nature}
	c.Family = []string{family.HornedBeast}
	c.ManaCost = 6
	c.ManaRequirement = []string{civ.Nature}

	c.Use(fx.Creature, fx.When(fx.WouldBeDestroyed, fx.ReturnToMana))

}
