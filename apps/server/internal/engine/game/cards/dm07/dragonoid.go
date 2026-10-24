package dm07

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

func SkyCrushertheAgitator(c *match.Card) {

	c.Name = "Sky Crusher, the Agitator"
	c.Power = 4000
	c.Civs = []string{civ.Fire}
	c.Family = []string{family.Dragonoid}
	c.ManaCost = 7
	c.ManaRequirement = []string{civ.Fire}
	c.TapAbility = fx.EachPlayerDestroys1Mana

	c.Use(fx.Creature, fx.TapAbility)
}
