package dm04

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// PurplePiercer ...
func PurplePiercer(c *match.Card) {

	c.Name = "Purple Piercer"
	c.Power = 2000
	c.Civs = []string{civ.Darkness}
	c.Family = []string{family.BrainJacker}
	c.ManaCost = 3
	c.ManaRequirement = []string{civ.Darkness}

	c.Use(
		fx.Creature,
		fx.CantBeBlockedByLight,
		fx.CantBeAttackedIf(func(attacker *match.Card) bool {
			return attacker.HasCiv(civ.Light)
		}),
	)

}
