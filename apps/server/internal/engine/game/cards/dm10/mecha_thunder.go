package dm10

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// BalzaSeekerOfHyperpearls ...
func BalzaSeekerOfHyperpearls(c *match.Card) {
	c.Name = "Balza, Seeker of Hyperpearls"
	c.Power = 4000
	c.Civs = []string{civ.Light}
	c.Family = []string{family.MechaThunder}
	c.ManaCost = 8
	c.ManaRequirement = []string{civ.Light}

	c.Use(fx.Creature, fx.ShieldTrigger, fx.Blocker(), fx.CantAttackPlayers)
}
