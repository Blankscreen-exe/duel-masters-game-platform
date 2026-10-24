package dm07

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

func WorldTreeRootOfLife(c *match.Card) {

	c.Name = "World Tree, Root of Life"
	c.Power = 7000
	c.Civs = []string{civ.Nature}
	c.Family = []string{family.TreeFolk}
	c.ManaCost = 6
	c.ManaRequirement = []string{civ.Nature}

	c.Use(fx.Creature, fx.Evolution, fx.PowerAttacker2000, fx.DarknessStealth, fx.Doublebreaker)

}
