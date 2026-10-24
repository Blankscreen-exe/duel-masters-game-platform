package dm09

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// CyclolinkSpectralKnight ...
func CyclolinkSpectralKnight(c *match.Card) {

	c.Name = "Cyclolink, Spectral Knight"
	c.Power = 3000
	c.Civs = []string{civ.Light}
	c.Family = []string{family.RainbowPhantom}
	c.ManaCost = 4
	c.ManaRequirement = []string{civ.Light}

	c.Use(fx.Creature, fx.When(fx.WheneverThisAttacksPlayerAndIsntBlocked, fx.SearchDeckTake1Spell))

}
