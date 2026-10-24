package dm10

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// TulkTheOracle ...
func TulkTheOracle(c *match.Card) {

	c.Name = "Tulk, the Oracle"
	c.Civs = []string{civ.Light}
	c.Power = 500
	c.Family = []string{family.LightBringer}
	c.ManaCost = 1
	c.ManaRequirement = []string{civ.Light}

	c.Use(fx.Creature)

}

// KaemiraTheOracle ...
func KaemiraTheOracle(c *match.Card) {

	c.Name = "Kaemira, the Oracle"
	c.Power = 1000
	c.Civs = []string{civ.Light}
	c.Family = []string{family.LightBringer}
	c.ManaCost = 4
	c.ManaRequirement = []string{civ.Light}

	c.Use(fx.Creature, fx.SilentSkill(fx.TopCardToShield))

}
