package dm01

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// EmeraldGrass ...
func EmeraldGrass(c *match.Card) {

	c.Name = "Emerald Grass"
	c.Power = 3000
	c.Civs = []string{civ.Light}
	c.Family = []string{family.StarlightTree}
	c.ManaCost = 2
	c.ManaRequirement = []string{civ.Light}

	c.Use(fx.Creature, fx.Blocker(), fx.CantAttackPlayers)

}

// RubyGrass ...
func RubyGrass(c *match.Card) {

	c.Name = "Ruby Grass"
	c.Power = 3000
	c.Civs = []string{civ.Light}
	c.Family = []string{family.StarlightTree}
	c.ManaCost = 3
	c.ManaRequirement = []string{civ.Light}

	c.Use(fx.Creature, fx.Blocker(), fx.CantAttackPlayers, fx.When(fx.EndOfMyTurnCreatureBZ, fx.MayUntapSelf))

}

// SenatineJadeTree ...
func SenatineJadeTree(c *match.Card) {

	c.Name = "Senatine Jade Tree"
	c.Power = 4000
	c.Civs = []string{civ.Light}
	c.Family = []string{family.StarlightTree}
	c.ManaCost = 3
	c.ManaRequirement = []string{civ.Light}

	c.Use(fx.Creature, fx.Blocker(), fx.CantAttackPlayers)

}
