package dm01

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// DeathbladeBeetle ...
func DeathbladeBeetle(c *match.Card) {

	c.Name = "Deathblade Beetle"
	c.Power = 3000
	c.Civs = []string{civ.Nature}
	c.Family = []string{family.GiantInsect}
	c.ManaCost = 5
	c.ManaRequirement = []string{civ.Nature}

	c.Use(fx.Creature, fx.Doublebreaker, fx.PowerAttacker4000)

}

// ForestHornet ...
func ForestHornet(c *match.Card) {

	c.Name = "Forest Hornet"
	c.Power = 4000
	c.Civs = []string{civ.Nature}
	c.Family = []string{family.GiantInsect}
	c.ManaCost = 4
	c.ManaRequirement = []string{civ.Nature}

	c.Use(fx.Creature)

}

// RedEyeScorpion ...
func RedEyeScorpion(c *match.Card) {

	c.Name = "Red-Eye Scorpion"
	c.Power = 4000
	c.Civs = []string{civ.Nature}
	c.Family = []string{family.GiantInsect}
	c.ManaCost = 5
	c.ManaRequirement = []string{civ.Nature}

	c.Use(fx.Creature, fx.ReturnToMana)

}
