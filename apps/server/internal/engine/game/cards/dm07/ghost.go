package dm07

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

func DreamPirateShadowOfTheft(c *match.Card) {

	c.Name = "Dream Pirate, Shadow of Theft"
	c.Power = 3000
	c.Civs = []string{civ.Darkness}
	c.Family = []string{family.Ghost}
	c.ManaCost = 4
	c.ManaRequirement = []string{civ.Darkness}

	c.Use(fx.Creature, fx.When(fx.WouldBeDestroyed, fx.MayReturnToHandAndDiscardACard))

}
