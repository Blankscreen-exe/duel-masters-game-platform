package dm08

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// BakkraHornTheSilent ...
func BakkraHornTheSilent(c *match.Card) {

	c.Name = "Bakkra Horn, the Silent"
	c.Power = 2000
	c.Civs = []string{civ.Nature}
	c.Family = []string{family.HornedBeast}
	c.ManaCost = 4
	c.ManaRequirement = []string{civ.Nature}

	c.Use(fx.Creature,
		fx.When(
			fx.AnotherOwnDragonoidOrDragonSummoned,
			fx.Draw1ToMana,
		))
}
