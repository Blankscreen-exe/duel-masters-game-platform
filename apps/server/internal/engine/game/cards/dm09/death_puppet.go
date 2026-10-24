package dm09

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// TrixoWickedDoll ...
func TrixoWickedDoll(c *match.Card) {

	c.Name = "Trixo, Wicked Doll"
	c.Power = 2000
	c.Civs = []string{civ.Darkness}
	c.Family = []string{family.DeathPuppet}
	c.ManaCost = 4
	c.ManaRequirement = []string{civ.Darkness}

	c.Use(fx.Creature, fx.When(fx.WheneverThisAttacksPlayerAndIsntBlocked, fx.OpponentChoosesAndDestroysCreature))

}
