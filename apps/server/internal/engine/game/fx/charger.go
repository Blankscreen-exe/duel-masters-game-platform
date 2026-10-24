package fx

import (
	"fmt"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

func Charger(card *match.Card, ctx *match.Context) {
	// After you cast this spell, put it into your mana zone instead of your graveyard.
	if event, ok := ctx.Event.(*match.SpellResolved); ok && event.CardID == card.ID {
		card.Player.MoveCard(card.ID, match.HAND, match.MANAZONE, card.ID)
		card.Tapped = false
		ctx.Match.ReportActionInChat(card.Player, fmt.Sprintf("%s was put in the mana zone instead of your graveyard", card.Name))
	}
}
