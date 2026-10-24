package dm04

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// BlastoExplosiveSoldier ...
func BlastoExplosiveSoldier(c *match.Card) {

	c.Name = "Blasto, Explosive Soldier"
	c.Power = 2000
	c.Civs = []string{civ.Fire}
	c.Family = []string{family.Dragonoid}
	c.ManaCost = 3
	c.ManaRequirement = []string{civ.Fire}

	c.PowerModifier = func(m *match.Match, attacking bool) int {

		power := 0

		if match.ContainerHas(c.Player, match.BATTLEZONE, func(x *match.Card) bool { return x.HasCiv(civ.Darkness) }) {
			power += 2000
		}

		return power
	}

	c.Use(fx.Creature)
}
