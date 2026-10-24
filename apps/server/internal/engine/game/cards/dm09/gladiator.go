package dm09

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// BetraleTheExplorer ...
func BetraleTheExplorer(c *match.Card) {

	c.Name = "Betrale, the Explorer"
	c.Power = 5000
	c.Civs = []string{civ.Light}
	c.Family = []string{family.Gladiator}
	c.ManaCost = 5
	c.ManaRequirement = []string{civ.Light}

	c.Use(fx.Creature, fx.CantAttackPlayers, fx.Blocker(),
		fx.When(fx.EndOfMyTurnCreatureBZ, fx.MayUntapSelf))

}
