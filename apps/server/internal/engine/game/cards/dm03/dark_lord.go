package dm03

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// BaragaBladeOfGloom ...
func BaragaBladeOfGloom(c *match.Card) {

	c.Name = "Baraga, Blade of Gloom"
	c.Power = 4000
	c.Civs = []string{civ.Darkness}
	c.Family = []string{family.DarkLord}
	c.ManaCost = 4
	c.ManaRequirement = []string{civ.Darkness}

	c.Use(fx.Creature, fx.When(fx.Summoned, fx.PutShieldIntoHand))

}
