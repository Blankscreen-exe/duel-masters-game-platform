package dm07

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

func GeoshineSpectralKnight(c *match.Card) {

	c.Name = "Geoshine, Spectral Knight"
	c.Power = 4000
	c.Civs = []string{civ.Light}
	c.Family = []string{family.RainbowPhantom}
	c.ManaCost = 5
	c.ManaRequirement = []string{civ.Light}

	c.Use(fx.Creature, fx.WheneverThisAttacksMayTapDorFCreature())

}
