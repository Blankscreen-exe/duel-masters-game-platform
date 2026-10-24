package dm07

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

func PoppleFlowerpetalDancer(c *match.Card) {

	c.Name = "Popple, Flowerpetal Dancer"
	c.Power = 2000
	c.Civs = []string{civ.Nature}
	c.Family = []string{family.SnowFaerie}
	c.ManaCost = 4
	c.ManaRequirement = []string{civ.Nature}
	c.TapAbility = fx.Draw1ToMana

	c.Use(fx.Creature, fx.TapAbility)
}
