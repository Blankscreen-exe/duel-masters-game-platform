package dm07

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

func CrathLadeMercilessKing(c *match.Card) {

	c.Name = "Crath Lade, Merciless King"
	c.Power = 4000
	c.Civs = []string{civ.Darkness}
	c.Family = []string{family.DarkLord}
	c.ManaCost = 8
	c.ManaRequirement = []string{civ.Darkness}
	c.TapAbility = fx.OpponentDiscards2RandomCards

	c.Use(fx.Creature, fx.TapAbility)
}
