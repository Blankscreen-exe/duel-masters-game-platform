package dm09

import (
	"fmt"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// TentacleCluster ...
func TentacleCluster(c *match.Card) {

	c.Name = "Tentacle Cluster"
	c.Power = 2000
	c.Civs = []string{civ.Water}
	c.Family = []string{family.CyberCluster}
	c.ManaCost = 5
	c.ManaRequirement = []string{civ.Water}

	c.Use(fx.Creature, fx.When(fx.WheneverThisAttacksPlayerAndIsntBlocked, fx.ReturnCreatureToOwnersHand))

}

// ScoutCluster ...
func ScoutCluster(c *match.Card) {

	c.Name = "Scout Cluster"
	c.Power = 4000
	c.Civs = []string{civ.Water}
	c.Family = []string{family.CyberCluster}
	c.ManaCost = 3
	c.ManaRequirement = []string{civ.Water}

	c.Use(fx.Creature, fx.Blocker(), fx.When(fx.AnotherOwnCreatureSummoned, func(card *match.Card, ctx *match.Context) {
		card.Player.MoveCard(card.ID, match.BATTLEZONE, match.HAND, card.ID)
		ctx.Match.ReportActionInChat(card.Player, fmt.Sprintf("%s was returned to the hand", card.Name))
	}))

}
