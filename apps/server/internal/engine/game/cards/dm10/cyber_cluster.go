package dm10

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// BatteryCluster ...
func BatteryCluster(c *match.Card) {

	c.Name = "Battery Cluster"
	c.Power = 3000
	c.Civs = []string{civ.Water}
	c.Family = []string{family.CyberCluster}
	c.ManaCost = 2
	c.ManaRequirement = []string{civ.Water}

	c.Use(fx.Creature, fx.Blocker(), fx.CantAttackCreatures, fx.CantAttackPlayers)

}

// TorpedoCluster ...
func TorpedoCluster(c *match.Card) {

	c.Name = "Torpedo Cluster"
	c.Power = 3000
	c.Civs = []string{civ.Water}
	c.Family = []string{family.CyberCluster}
	c.ManaCost = 3
	c.ManaRequirement = []string{civ.Water}

	c.Use(fx.Creature, fx.When(fx.Summoned, fx.ReturnMyCardFromMZToHand))

}
