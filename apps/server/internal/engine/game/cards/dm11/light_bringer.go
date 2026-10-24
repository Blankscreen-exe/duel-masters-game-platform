package dm11

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/cnd"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// MerleeTheOracle ...
func MerleeTheOracle(c *match.Card) {

	c.Name = "Merlee, the Oracle"
	c.Power = 1500
	c.Civs = []string{civ.Light}
	c.Family = []string{family.LightBringer}
	c.ManaCost = 2
	c.ManaRequirement = []string{civ.Light}

	// "Each of your creatures" includes Merlee itself, so the grant is not
	// filtered down to the others.
	c.Use(fx.Creature, fx.WaveStriker, fx.WaveStrikerGrantToOwnCreatures(cnd.PowerAmplifier, 1000))

}
