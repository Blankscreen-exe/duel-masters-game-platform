package dm02

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// GeneralDarkFiend ...
func GeneralDarkFiend(c *match.Card) {

	c.Name = "General Dark Fiend"
	c.Power = 6000
	c.Civs = []string{civ.Darkness}
	c.Family = []string{family.DarkLord}
	c.ManaCost = 5
	c.ManaRequirement = []string{civ.Darkness}

	c.Use(fx.Creature, fx.Doublebreaker, fx.When(fx.Attacking, func(card *match.Card, ctx *match.Context) {

		shields := fx.Find(
			card.Player,
			match.SHIELDZONE,
		)

		if len(shields) < 1 {
			return
		}

		ctx.Match.MoveCard(shields[0], match.GRAVEYARD, card)

	}))

}
