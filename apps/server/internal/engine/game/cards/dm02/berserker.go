package dm02

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// LagunaLightningEnforcer ...
func LagunaLightningEnforcer(c *match.Card) {

	c.Name = "Laguna, Lightning Enforcer"
	c.Power = 2500
	c.Civs = []string{civ.Light}
	c.Family = []string{family.Berserker}
	c.ManaCost = 5
	c.ManaRequirement = []string{civ.Light}

	c.Use(fx.Creature, fx.When(fx.Attacking, fx.SearchDeckTake1Spell))

}
