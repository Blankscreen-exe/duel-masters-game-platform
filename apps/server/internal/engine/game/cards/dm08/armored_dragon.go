package dm08

import (
	"fmt"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// UberdragonBajula ...
func UberdragonBajula(c *match.Card) {

	c.Name = "Überdragon Bajula"
	c.Power = 13000
	c.Civs = []string{civ.Fire}
	c.Family = []string{family.ArmoredDragon}
	c.ManaCost = 7
	c.ManaRequirement = []string{civ.Fire}

	c.Use(fx.Creature, fx.DragonEvolution, fx.Triplebreaker, fx.When(fx.AttackConfirmed, fx.ManaBurnX(2)))
}

// BruiserDragon ...
func BruiserDragon(c *match.Card) {

	c.Name = "Bruiser Dragon"
	c.Power = 5000
	c.Civs = []string{civ.Fire}
	c.Family = []string{family.ArmoredDragon}
	c.ManaCost = 5
	c.ManaRequirement = []string{civ.Fire}

	c.Use(fx.Creature, fx.When(fx.Destroyed, func(card *match.Card, ctx *match.Context) {
		fx.SelectBackside(
			card.Player,
			ctx.Match,
			card.Player,
			match.SHIELDZONE,
			fmt.Sprintf("%s's effect: Put 1 of your shields into your graveyard.", card.Name),
			1,
			1,
			false,
		).Map(func(x *match.Card) {
			ctx.Match.MoveCard(x, match.GRAVEYARD, card)
			ctx.Match.ReportActionInChat(card.Player, fmt.Sprintf("%s's effect: One of your shields was put into graveyard", card.Name))
		})
	}))

}
