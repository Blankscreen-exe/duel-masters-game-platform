package fx

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/cnd"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

func Stealth(card *match.Card, ctx *match.Context, civ string) {
	if _, ok := ctx.Event.(*match.UntapStep); ok {
		card.AddCondition(cnd.Stealth, civ, card.ID)
	}
}

func FireStealth(card *match.Card, ctx *match.Context) {
	Stealth(card, ctx, civ.Fire)
}

func LightStealth(card *match.Card, ctx *match.Context) {
	Stealth(card, ctx, civ.Light)
}

func DarknessStealth(card *match.Card, ctx *match.Context) {
	Stealth(card, ctx, civ.Darkness)
}

func NatureStealth(card *match.Card, ctx *match.Context) {
	Stealth(card, ctx, civ.Nature)
}

func WaterStealth(card *match.Card, ctx *match.Context) {
	Stealth(card, ctx, civ.Water)
}
