package dm07

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

func SiriGloryElemental(c *match.Card) {

	c.Name = "Siri, Glory Elemental"
	c.Power = 7000
	c.Civs = []string{civ.Light}
	c.Family = []string{family.AngelCommand}
	c.ManaCost = 6
	c.ManaRequirement = []string{civ.Light}

	c.Use(fx.Creature, fx.Doublebreaker,
		fx.WhenAll([]func(*match.Card, *match.Context) bool{fx.EndOfMyTurnCreatureBZ, fx.IDontHaveShields}, fx.MayUntapSelf),
		fx.When(fx.InTheBattlezone, fx.BlockerWhenNoShields),
	)

}
