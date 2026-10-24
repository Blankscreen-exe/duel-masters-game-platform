package dm08

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// QuixoticHeroSwineSnout ...
func QuixoticHeroSwineSnout(c *match.Card) {

	c.Name = "Quixotic Hero Swine Snout"
	c.Power = 1000
	c.Civs = []string{civ.Nature}
	c.Family = []string{family.BeastFolk}
	c.ManaCost = 2
	c.ManaRequirement = []string{civ.Nature}

	powerBoost := 0

	c.PowerModifier = func(m *match.Match, attacking bool) int { return powerBoost }

	c.Use(fx.Creature,
		fx.When(fx.AnotherCreatureSummoned, func(card *match.Card, ctx *match.Context) { powerBoost += 3000 }),
		fx.When(fx.EndOfTurn, func(card *match.Card, ctx *match.Context) { powerBoost = 0 }),
	)
}
