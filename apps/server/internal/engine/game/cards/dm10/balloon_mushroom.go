package dm10

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// SporeblastErengi ...
func SporeblastErengi(c *match.Card) {

	c.Name = "Sporeblast Erengi"
	c.Power = 4000
	c.Civs = []string{civ.Nature}
	c.Family = []string{family.BalloonMushroom}
	c.ManaCost = 4
	c.ManaRequirement = []string{civ.Nature}

	c.Use(fx.Creature, fx.SilentSkill(fx.SearchDeckTake1Creature))

}
