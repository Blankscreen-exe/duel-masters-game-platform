package fx

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/cnd"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// Adds the tap ability condition every turn
func TapAbility(card *match.Card, ctx *match.Context) {

	if _, ok := ctx.Event.(*match.UntapStep); ok {
		card.AddUniqueSourceCondition(cnd.TapAbility, card.TapAbility, card.ID)
	}

}
