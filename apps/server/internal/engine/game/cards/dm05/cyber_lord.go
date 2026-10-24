package dm05

import (
	"fmt"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// Pokolul ...
func Pokolul(c *match.Card) {

	c.Name = "Pokolul"
	c.Power = 2000
	c.Civs = []string{civ.Water}
	c.Family = []string{family.CyberLord}
	c.ManaCost = 4
	c.ManaRequirement = []string{civ.Water}

	isAttacking := false

	c.Use(fx.Creature, func(card *match.Card, ctx *match.Context) {

		if event, ok := ctx.Event.(*match.AttackPlayer); ok {

			if event.CardID == card.ID {
				isAttacking = true
			} else {
				isAttacking = false
			}

		}

		if _, ok := ctx.Event.(*match.ShieldTriggerPlayedEvent); ok && isAttacking {
			card.Tapped = false
			ctx.Match.ReportActionInChat(card.Player, fmt.Sprintf("%s untapped", card.Name))
		}

	})

}
