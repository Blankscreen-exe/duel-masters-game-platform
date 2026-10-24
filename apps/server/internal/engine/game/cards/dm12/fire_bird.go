package dm12

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// PeppiPepper ...
func PeppiPepper(c *match.Card) {

	c.Name = "Peppi Pepper"
	c.Power = 2000
	c.Civs = []string{civ.Fire}
	c.Family = []string{family.FireBird}
	c.ManaCost = 3
	c.ManaRequirement = []string{civ.Fire}

	c.Use(fx.Creature, fx.PowerAttacker3000)

}

// BuzzBetocchi ...
func BuzzBetocchi(c *match.Card) {

	c.Name = "Buzz Betocchi"
	c.Power = 4000
	c.Civs = []string{civ.Fire, civ.Nature}
	c.Family = []string{family.FireBird, family.GiantInsect}
	c.ManaCost = 3
	c.ManaRequirement = []string{civ.Fire, civ.Nature}

	c.Use(fx.Creature, fx.PutIntoManaZoneTapped)

}
