package dm08

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// TyrantWorm ...
func TyrantWorm(c *match.Card) {

	c.Name = "Tyrant Worm"
	c.Power = 2000
	c.Civs = []string{civ.Darkness}
	c.Family = []string{family.ParasiteWorm}
	c.ManaCost = 1
	c.ManaRequirement = []string{civ.Darkness}

	c.Use(fx.Creature, fx.When(fx.AnotherOwnCreatureSummoned, fx.DestroyYourself))

}
