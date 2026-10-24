package dm04

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// CannonShell ...
func CannonShell(c *match.Card) {

	c.Name = "Cannon Shell"
	c.Power = 1000
	c.Civs = []string{civ.Nature}
	c.Family = []string{family.ColonyBeetle}
	c.ManaCost = 4
	c.ManaRequirement = []string{civ.Nature}

	c.PowerModifier = func(m *match.Match, attacking bool) int {

		shields, err := c.Player.Container(match.SHIELDZONE)

		if err != nil {
			return 0
		}

		return 1000 * len(shields)
	}

	c.Use(fx.Creature, fx.ShieldTrigger)

}
