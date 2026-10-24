package dm04

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// KamikazeChainsawWarrior ...
func KamikazeChainsawWarrior(c *match.Card) {

	c.Name = "Kamikaze, Chainsaw Warrior"
	c.Power = 1000
	c.Civs = []string{civ.Fire}
	c.Family = []string{family.Armorloid}
	c.ManaCost = 2
	c.ManaRequirement = []string{civ.Fire}

	c.Use(fx.Creature, fx.ShieldTrigger)
}
