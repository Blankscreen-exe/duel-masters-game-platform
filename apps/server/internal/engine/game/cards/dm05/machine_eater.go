package dm05

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// RikabuTheDismantler ...
func RikabuTheDismantler(c *match.Card) {

	c.Name = "Rikabu, the Dismantler"
	c.Power = 1000
	c.Civs = []string{civ.Fire}
	c.Family = []string{family.MachineEater}
	c.ManaCost = 3
	c.ManaRequirement = []string{civ.Fire}

	c.Use(fx.Creature, fx.SpeedAttacker)

}
