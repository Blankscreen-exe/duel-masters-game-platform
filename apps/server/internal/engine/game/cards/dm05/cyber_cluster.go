package dm05

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

func SteelTurretCluster(c *match.Card) {
	c.Name = "Steel-Turret Cluster"
	c.Power = 3000
	c.Civs = []string{civ.Water}
	c.Family = []string{family.CyberCluster}
	c.ManaCost = 5
	c.ManaRequirement = []string{civ.Water}

	c.Use(
		fx.Creature,
		fx.CantBeAttackedIf(func(attacker *match.Card) bool {
			return attacker.HasCiv(civ.Fire) || attacker.HasCiv(civ.Nature)
		}),
	)
}
