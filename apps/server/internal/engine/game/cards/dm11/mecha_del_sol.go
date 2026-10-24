package dm11

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// YulianaChannelerOfSuns ...
func YulianaChannelerOfSuns(c *match.Card) {

	c.Name = "Yuliana, Channeler of Suns"
	c.Power = 3000
	c.Civs = []string{civ.Light}
	c.Family = []string{family.MechaDelSol}
	c.ManaCost = 3
	c.ManaRequirement = []string{civ.Light}

	// Being unchoosable does not make it untouchable: it can still be attacked
	// and blocked, which is exactly what CantBeSelectedByOpp models.
	c.Use(fx.Creature, fx.Blocker(), fx.CantBeSelectedByOpp, fx.CantAttackPlayers)

}
