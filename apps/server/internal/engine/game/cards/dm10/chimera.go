package dm10

import (
	"fmt"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// Gigandura ...
func Gigandura(c *match.Card) {

	c.Name = "Gigandura"
	c.Power = 3000
	c.Civs = []string{civ.Darkness}
	c.Family = []string{family.Chimera}
	c.ManaCost = 5
	c.ManaRequirement = []string{civ.Darkness}

	c.Use(fx.Creature, fx.When(fx.Summoned, func(card *match.Card, ctx *match.Context) {
		fx.Select(
			card.Player,
			ctx.Match,
			ctx.Match.Opponent(card.Player),
			match.HAND,
			fmt.Sprintf("%s's effect: You may choose a card from your opponent's hand and put it into his mana zone. If you do, choose a card in his mana zone and put it into his hand.", card.Name),
			1,
			1,
			true,
		).Map(func(x *match.Card) {
			ctx.Match.MoveCard(x, match.MANAZONE, card)

			fx.Select(
				card.Player,
				ctx.Match,
				ctx.Match.Opponent(card.Player),
				match.MANAZONE,
				fmt.Sprintf("%s's effect: Choose a card in your opponent's mana zone and put it into his hand.", card.Name),
				1,
				1,
				false,
			).Map(func(x *match.Card) {
				ctx.Match.MoveCard(x, match.HAND, card)
			})
		})
	}))

}

// Gigamente ...
func Gigamente(c *match.Card) {

	c.Name = "Gigamente"
	c.Power = 3000
	c.Civs = []string{civ.Darkness}
	c.Family = []string{family.Chimera}
	c.ManaCost = 4
	c.ManaRequirement = []string{civ.Darkness}

	c.Use(fx.Creature, fx.SilentSkill(fx.ReturnCreaturesFromGraveToHand(1, false)))

}
