package dm06

import (
	"fmt"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

func SchukaDukeOfAmnesia(c *match.Card) {

	c.Name = "Schuka, Duke of Amnesia"
	c.Power = 5000
	c.Civs = []string{civ.Darkness}
	c.Family = []string{family.DarkLord}
	c.ManaCost = 6
	c.ManaRequirement = []string{civ.Darkness}

	c.Use(fx.Creature, fx.When(fx.Destroyed, func(card *match.Card, ctx *match.Context) {

		for _, c := range append(fx.Find(card.Player, match.HAND), fx.Find(ctx.Match.Opponent(card.Player), match.HAND)...) {
			c.Player.MoveCard(c.ID, match.HAND, match.GRAVEYARD, card.ID)
			ctx.Match.ReportActionInChat(c.Player, fmt.Sprintf("%s was was discarded from %s's hand by %s", c.Name, c.Player.Username(), card.Name))
		}
	}))
}
