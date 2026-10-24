package dm09

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// SnaptongueLizard ...
func SnaptongueLizard(c *match.Card) {

	c.Name = "Snaptongue Lizard"
	c.Power = 3000
	c.Civs = []string{civ.Fire}
	c.Family = []string{family.DuneGecko}
	c.ManaCost = 6
	c.ManaRequirement = []string{civ.Fire}

	c.Use(fx.Creature, fx.PowerAttacker3000, fx.CantBeBlockedWhileAttackingACreature)

}
