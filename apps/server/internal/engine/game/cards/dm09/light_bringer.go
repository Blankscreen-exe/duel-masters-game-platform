package dm09

import (
	"fmt"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// MicuteTheOracle ...
func MicuteTheOracle(c *match.Card) {

	c.Name = "Micute, the Oracle"
	c.Power = 4000
	c.Civs = []string{civ.Light}
	c.Family = []string{family.LightBringer}
	c.ManaCost = 5
	c.ManaRequirement = []string{civ.Light}

	c.Use(fx.Creature,
		fx.When(fx.AnotherOwnGuardianSummoned, func(card *match.Card, ctx *match.Context) {
			fx.Select(
				card.Player,
				ctx.Match,
				ctx.Match.Opponent(card.Player),
				match.BATTLEZONE,
				fmt.Sprintf("%s's effect: You may choose one of your opponent's creatures in the battlezone and tap it.", card.Name),
				1,
				1,
				true,
			).Map(func(x *match.Card) {
				if card.Player.TapCard(x) {
					ctx.Match.ReportActionInChat(card.Player, fmt.Sprintf("%s was tapped by %s's effect.", x.Name, card.Name))
				}
			})
		}))
}
