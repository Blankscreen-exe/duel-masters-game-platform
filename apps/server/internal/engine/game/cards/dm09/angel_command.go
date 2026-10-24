package dm09

import (
	"fmt"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// MihailCelestialElemental ...
func MihailCelestialElemental(c *match.Card) {

	c.Name = "Mihail, Celestial Elemental"
	c.Power = 4000
	c.Civs = []string{civ.Light}
	c.Family = []string{family.AngelCommand}
	c.ManaCost = 8
	c.ManaRequirement = []string{civ.Light}

	c.Use(fx.Creature, func(card *match.Card, ctx *match.Context) {
		if event, ok := ctx.Event.(*match.CreatureDestroyed); ok && card.Zone == match.BATTLEZONE {
			if event.Card.ID != card.ID {
				ctx.InterruptFlow()
				ctx.Match.ReportActionInChat(event.Card.Player, fmt.Sprintf("%s stayed in the battlezone instead of being destroyed due to %s's effect.", event.Card.Name, card.Name))
			}
		}
	})

}
