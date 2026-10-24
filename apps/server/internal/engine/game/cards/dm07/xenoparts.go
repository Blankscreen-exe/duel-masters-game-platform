package dm07

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

func KiposContraption(c *match.Card) {

	c.Name = "Kipo's Contraption"
	c.Power = 3000
	c.Civs = []string{civ.Fire}
	c.Family = []string{family.Xenoparts}
	c.ManaCost = 6
	c.ManaRequirement = []string{civ.Fire}
	c.TapAbility = fx.DestroyByMiscOpCreature2000OrLess

	c.Use(fx.Creature, fx.TapAbility)
}
