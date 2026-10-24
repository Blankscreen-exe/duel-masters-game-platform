package dm11

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// FantasyFish ...
func FantasyFish(c *match.Card) {

	c.Name = "Fantasy Fish"
	c.Power = 2000
	c.Civs = []string{civ.Water}
	c.Family = []string{family.GelFish}
	c.ManaCost = 7
	c.ManaRequirement = []string{civ.Water}

	c.Use(fx.Creature, fx.ShieldTrigger, fx.Blocker())

}
