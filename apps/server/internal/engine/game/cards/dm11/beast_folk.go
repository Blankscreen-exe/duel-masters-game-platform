package dm11

import (
	"fmt"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/cnd"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// QuillspikeRumbler ...
func QuillspikeRumbler(c *match.Card) {

	c.Name = "Quillspike Rumbler"
	c.Power = 3000
	c.Civs = []string{civ.Nature}
	c.Family = []string{family.BeastFolk}
	c.ManaCost = 4
	c.ManaRequirement = []string{civ.Nature}

	c.Use(fx.Creature, func(card *match.Card, ctx *match.Context) {
		event, ok := ctx.Event.(*match.AttackConfirmed)

		if !ok || event.CardID != card.ID || !event.Creature || card.Zone != match.BATTLEZONE {
			return
		}

		// Added before the battle captures the attacker's power, and cleared
		// along with the card's other conditions at the end of the turn.
		card.AddCondition(cnd.PowerAmplifier, 3000, card.ID)
		ctx.Match.ReportActionInChat(card.Player, fmt.Sprintf("%s gets +3000 power until the end of the turn", card.Name))
	})

}
