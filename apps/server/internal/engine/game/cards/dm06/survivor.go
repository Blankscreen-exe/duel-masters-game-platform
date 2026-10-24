package dm06

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

func QTronicHypermind(c *match.Card) {

	c.Name = "Q-tronic Hypermind"
	c.Power = 8000
	c.Civs = []string{civ.Water}
	c.Family = []string{family.Survivor}
	c.ManaCost = 8
	c.ManaRequirement = []string{civ.Water}

	c.Use(fx.Creature, fx.Doublebreaker, fx.Evolution, fx.Survivor, fx.When(fx.Summoned, func(card *match.Card, ctx *match.Context) {

		mysurvivors := len(fx.FindFilter(card.Player, match.BATTLEZONE, func(card *match.Card) bool { return card.HasFamily(family.Survivor) }))
		enemysurvivors := len(fx.FindFilter(ctx.Match.Opponent(card.Player), match.BATTLEZONE, func(card *match.Card) bool { return card.HasFamily(family.Survivor) }))
		toDraw := mysurvivors + enemysurvivors

		if toDraw < 1 {
			return
		}

		fx.DrawUpto(card, ctx, toDraw)

	}))
}

func QTronicGargantua(c *match.Card) {

	c.Name = "Q-tronic Gargantua"
	c.Power = 9000
	c.Civs = []string{civ.Fire}
	c.Family = []string{family.Survivor}
	c.ManaCost = 6
	c.ManaRequirement = []string{civ.Fire}

	// Crew breaker - Survivor
	c.Use(
		fx.Creature,
		fx.Evolution,
		fx.Survivor,
		fx.CrewBreaker(fx.CountOtherOwnCreaturesWithFamily([]string{family.Survivor})),
	)
}
