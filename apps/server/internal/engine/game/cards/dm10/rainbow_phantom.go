package dm10

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// FerrosaturnSpectralKnight ...
func FerrosaturnSpectralKnight(c *match.Card) {

	c.Name = "Ferrosaturn, Spectral Knight"
	c.Power = 2000
	c.Civs = []string{civ.Light}
	c.Family = []string{family.RainbowPhantom}
	c.ManaCost = 1
	c.ManaRequirement = []string{civ.Light}

	c.Use(fx.Creature, fx.Blocker(), fx.CantAttackPlayers)

}
