package dm07

import (
	"fmt"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

func RondobilTheExplorer(c *match.Card) {

	c.Name = "Rondobil, the Explorer"
	c.Power = 5000
	c.Civs = []string{civ.Light}
	c.Family = []string{family.Gladiator}
	c.ManaCost = 6
	c.ManaRequirement = []string{civ.Light}
	c.TapAbility = func(card *match.Card, ctx *match.Context) {
		fx.Select(card.Player, ctx.Match, card.Player, match.BATTLEZONE,
			"Select creature to add to shields", 1, 1, false,
		).Map(func(x *match.Card) {
			card.Player.MoveCard(x.ID, match.BATTLEZONE, match.SHIELDZONE, card.ID)
			ctx.Match.ReportActionInChat(
				card.Player,
				fmt.Sprintf("Rondobil, the Explorer effect: %s was added to shieldzone", c.Name),
			)
		})
	}

	c.Use(fx.Creature, fx.TapAbility)
}
