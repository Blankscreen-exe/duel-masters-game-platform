package dm07

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

func RodiGaleNightGuardian(c *match.Card) {

	c.Name = "Rodi Gale, Night Guardian"
	c.Power = 3500
	c.Civs = []string{civ.Light}
	c.Family = []string{family.Guardian}
	c.ManaCost = 4
	c.ManaRequirement = []string{civ.Light}

	c.Use(fx.Creature, fx.DarknessStealth)
}
