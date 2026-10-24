package fx

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/cnd"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// ShieldsSelectionEffect adds the HasShieldsSelectionEffect condition in UntapStep
func ShieldsSelectionEffect(card *match.Card, ctx *match.Context) {

	if _, ok := ctx.Event.(*match.UntapStep); ok {

		card.AddCondition(cnd.HasShieldsSelectionEffect, nil, nil)

	}

}
