package dm11

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// HazariaDukeOfThorns ...
func HazariaDukeOfThorns(c *match.Card) {

	c.Name = "Hazaria, Duke of Thorns"
	c.Power = 2000
	c.Civs = []string{civ.Darkness}
	c.Family = []string{family.DarkLord}
	c.ManaCost = 4
	c.ManaRequirement = []string{civ.Darkness}

	c.Use(fx.Creature, fx.WaveStriker, fx.WhileWaveStriker(fx.When(fx.Summoned, fx.OpponentChoosesAndDestroysCreature)))

}
