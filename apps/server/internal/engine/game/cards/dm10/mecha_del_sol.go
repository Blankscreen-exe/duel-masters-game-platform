package dm10

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/cnd"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// RyudmilaChannelerOfSuns ...
func RyudmilaChannelerOfSuns(c *match.Card) {
	c.Name = "Ryudmila, Channeler of Suns"
	c.Power = 2000
	c.Civs = []string{civ.Light}
	c.Family = []string{family.MechaDelSol}
	c.ManaCost = 5
	c.ManaRequirement = []string{civ.Light}

	c.Use(
		fx.Creature,
		fx.ModifyPowers(func(event *match.GetPowerEvent) {
			if c.Zone != match.BATTLEZONE || event.Card != c {
				return
			}

			event.Power += len(fx.FindFilter(
				c.Player,
				match.BATTLEZONE,
				func(card *match.Card) bool {
					return card.ID != c.ID &&
						card.HasCondition(cnd.Creature) &&
						!card.Tapped
				},
			)) * 2000
		}),
		fx.When(fx.WouldBeDestroyed, fx.ShuffleIntoDeckInsteadOfDestruction),
	)
}

// BerochikaChannelerOfSuns ...
func BerochikaChannelerOfSuns(c *match.Card) {

	c.Name = "Berochika, Channeler of Suns"
	c.Power = 5000
	c.Civs = []string{civ.Light}
	c.Family = []string{family.MechaDelSol}
	c.ManaCost = 5
	c.ManaRequirement = []string{civ.Light}

	c.Use(fx.Creature, fx.When(fx.Summoned, func(card *match.Card, ctx *match.Context) {
		myShields, _ := card.Player.Container(match.SHIELDZONE)

		if len(myShields) >= 5 {
			fx.TopCardToShield(card, ctx)
		}
	}))

}
