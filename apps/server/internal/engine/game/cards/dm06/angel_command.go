package dm06

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

func GarielElementalOfSunbeams(c *match.Card) {

	c.Name = "Gariel, Elemental of Sunbeams"
	c.Power = 7500
	c.Civs = []string{civ.Light}
	c.Family = []string{family.AngelCommand}
	c.ManaCost = 5
	c.ManaRequirement = []string{civ.Light}

	spellcast := false

	c.Use(fx.Creature, fx.Doublebreaker,
		fx.When(fx.IHaveCastASpell, func(card *match.Card, ctx *match.Context) { spellcast = true }),
		func(card *match.Card, ctx *match.Context) {

			// reset spellcast flag at the end of the turn
			if _, ok := ctx.Event.(*match.EndStep); ok {
				spellcast = false
			}

			if event, ok := ctx.Event.(*match.PlayCardEvent); ok {
				if event.CardID == card.ID {
					if !spellcast {
						ctx.InterruptFlow()
						ctx.Match.WarnPlayer(card.Player, "You can summon this creature only if you have cast a spell this turn.")
						return
					}
				}

			}

		})

}
