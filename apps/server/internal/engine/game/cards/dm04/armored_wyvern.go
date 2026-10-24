package dm04

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/cnd"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// ChaoticSkyterror ...
func ChaoticSkyterror(c *match.Card) {

	c.Name = "Chaotic Skyterror"
	c.Power = 4000
	c.Civs = []string{civ.Fire}
	c.Family = []string{family.ArmoredWyvern}
	c.ManaCost = 5
	c.ManaRequirement = []string{civ.Fire}

	c.Use(fx.Creature, func(card *match.Card, ctx *match.Context) {

		if card.Zone != match.BATTLEZONE {
			return
		}

		if event, ok := ctx.Event.(*match.AttackCreature); ok {
			chaoticSkyterrorSpecial(card, ctx, event.CardID)
		}

		if event, ok := ctx.Event.(*match.AttackPlayer); ok {
			chaoticSkyterrorSpecial(card, ctx, event.CardID)
		}

	})

}

func chaoticSkyterrorSpecial(card *match.Card, ctx *match.Context, cardID string) {

	p := ctx.Match.CurrentPlayer()

	creature, err := p.Player.GetCard(cardID, match.BATTLEZONE)

	if err != nil {
		return
	}

	if !creature.HasFamily(family.DemonCommand) {
		return
	}

	if ctx.Match.IsPlayerTurn(card.Player) {
		creature.AddUniqueSourceCondition(cnd.DoubleBreaker, true, card.ID)
		creature.AddUniqueSourceCondition(cnd.PowerAttacker, 4000, card.ID)
	}
}
