package dm06

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

func GrinningAxeTheMonstrosity(c *match.Card) {

	c.Name = "Grinning Axe, the Monstrosity"
	c.Power = 1000
	c.Civs = []string{civ.Darkness}
	c.Family = []string{family.DevilMask}
	c.ManaCost = 3
	c.ManaRequirement = []string{civ.Darkness}

	c.Use(fx.Creature, fx.Slayer)
}

func SkullcutterSwarmLeader(c *match.Card) {

	c.Name = "Skullcutter, Swarm Leader"
	c.Power = 4000
	c.Civs = []string{civ.Darkness}
	c.Family = []string{family.DevilMask}
	c.ManaCost = 4
	c.ManaRequirement = []string{civ.Darkness}

	c.Use(fx.Creature, fx.When(fx.EndOfMyTurn, func(card *match.Card, ctx *match.Context) {

		if card.Zone != match.BATTLEZONE {
			return
		}

		if len(fx.Find(card.Player, match.BATTLEZONE)) == 1 {

			ctx.Match.Destroy(card, card, match.DestroyedByMiscAbility)
		}

	}))
}
