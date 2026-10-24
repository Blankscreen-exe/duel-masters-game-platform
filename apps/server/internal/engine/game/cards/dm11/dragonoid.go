package dm11

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/cnd"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// SapianTarkFlameDervish ...
func SapianTarkFlameDervish(c *match.Card) {

	c.Name = "Sapian Tark, Flame Dervish"
	c.Power = 2000
	c.Civs = []string{civ.Fire}
	c.Family = []string{family.Dragonoid}
	c.ManaCost = 3
	c.ManaRequirement = []string{civ.Fire}

	c.PowerModifier = fx.WaveStrikerPower(c, 4000)

	c.Use(fx.Creature, fx.WaveStriker, fx.WaveStrikerGrant(cnd.AttackUntapped, true))

}
