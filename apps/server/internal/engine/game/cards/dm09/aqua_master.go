package dm09

import (
	"fmt"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// AquaMaster ...
func AquaMaster(c *match.Card) {

	c.Name = "Aqua Master"
	c.Power = 4000
	c.Civs = []string{civ.Water}
	c.Family = []string{family.LiquidPeople}
	c.ManaCost = 6
	c.ManaRequirement = []string{civ.Water}

	c.Use(fx.Creature, fx.ShieldsSelectionEffect,
		fx.When(fx.WheneverThisAttacksPlayerAndIsntBlocked,
			func(card *match.Card, ctx *match.Context) {
				fx.SelectBackside(
					card.Player,
					ctx.Match,
					ctx.Match.Opponent(card.Player),
					match.SHIELDZONE,
					fmt.Sprintf("%s's effect: Choose one of your opponent's shield and turn it face up.", card.Name),
					1,
					1,
					false,
				).Map(func(x *match.Card) {
					x.ShieldFaceUp = true
					ctx.Match.BroadcastState()
					ctx.Match.ReportActionInChat(card.Player, fmt.Sprintf("%s was turned face up from %s's shieldzone.", x.Name, ctx.Match.Opponent(card.Player).Username()))
				})
			}))
}
