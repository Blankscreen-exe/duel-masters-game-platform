package dm01

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// DeathligerLionOfChaos ...
func DeathligerLionOfChaos(c *match.Card) {

	c.Name = "Deathliger, Lion of Chaos"
	c.Power = 9000
	c.Civs = []string{civ.Darkness}
	c.Family = []string{family.DemonCommand}
	c.ManaCost = 7
	c.ManaRequirement = []string{civ.Darkness}

	c.Use(fx.Creature, fx.Doublebreaker)

}

// ZagaanKnightOfDarkness ...
func ZagaanKnightOfDarkness(c *match.Card) {

	c.Name = "Zagaan, Knight of Darkness"
	c.Power = 7000
	c.Civs = []string{civ.Darkness}
	c.Family = []string{family.DemonCommand}
	c.ManaCost = 6
	c.ManaRequirement = []string{civ.Darkness}

	c.Use(fx.Creature, fx.Doublebreaker)

}
