package dm04

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// HunterCluster ...
func HunterCluster(c *match.Card) {

	c.Name = "Hunter Cluster"
	c.Power = 1000
	c.Civs = []string{civ.Water}
	c.Family = []string{family.CyberCluster}
	c.ManaCost = 4
	c.ManaRequirement = []string{civ.Water}

	c.Use(fx.Creature, fx.Blocker(), fx.ShieldTrigger)

}
