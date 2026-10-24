package dm09

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// AcidRefluxTheFleshboiler ...
func AcidRefluxTheFleshboiler(c *match.Card) {

	c.Name = "Acid Reflux, the Fleshboiler"
	c.Power = 3000
	c.Civs = []string{civ.Darkness}
	c.Family = []string{family.DevilMask}
	c.ManaCost = 5
	c.ManaRequirement = []string{civ.Darkness}

	c.Use(fx.Creature, fx.Blocker(), fx.CantAttackCreatures, fx.CantAttackPlayers, fx.Slayer)

}
