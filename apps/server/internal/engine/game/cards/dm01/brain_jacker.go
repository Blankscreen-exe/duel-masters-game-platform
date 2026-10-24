package dm01

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// BloodySquito ...
func BloodySquito(c *match.Card) {

	c.Name = "Bloody Squito"
	c.Power = 4000
	c.Civs = []string{civ.Darkness}
	c.Family = []string{family.BrainJacker}
	c.ManaCost = 2
	c.ManaRequirement = []string{civ.Darkness}

	c.Use(fx.Creature, fx.Blocker(), fx.CantAttackPlayers, fx.CantAttackCreatures, fx.Suicide)

}

// DarkClown ...
func DarkClown(c *match.Card) {

	c.Name = "Dark Clown"
	c.Power = 6000
	c.Civs = []string{civ.Darkness}
	c.Family = []string{family.BrainJacker}
	c.ManaCost = 4
	c.ManaRequirement = []string{civ.Darkness}

	c.Use(fx.Creature, fx.Blocker(), fx.CantAttackPlayers, fx.CantAttackCreatures, fx.Suicide)

}
