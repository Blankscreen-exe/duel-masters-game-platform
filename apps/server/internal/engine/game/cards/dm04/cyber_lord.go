package dm04

import (
	"fmt"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// Marinomancer ...
func Marinomancer(c *match.Card) {

	c.Name = "Marinomancer"
	c.Power = 2000
	c.Civs = []string{civ.Water}
	c.Family = []string{family.CyberLord}
	c.ManaCost = 5
	c.ManaRequirement = []string{civ.Water}

	c.Use(fx.Creature, fx.When(fx.Summoned, func(card *match.Card, ctx *match.Context) {

		cards := card.Player.PeekDeck(3)

		for _, toMove := range cards {

			if toMove.HasCiv(civ.Light) || toMove.HasCiv(civ.Darkness) {
				card.Player.MoveCard(toMove.ID, match.DECK, match.HAND, card.ID)
				ctx.Match.ReportActionInChat(card.Player, fmt.Sprintf("%s put %s into the hand from the top of their deck", card.Player.Username(), toMove.Name))
			} else {
				card.Player.MoveCard(toMove.ID, match.DECK, match.GRAVEYARD, card.ID)
				ctx.Match.ReportActionInChat(card.Player, fmt.Sprintf("%s put %s into the graveyard from the top of their deck", card.Player.Username(), toMove.Name))
			}
		}

	}))

}
