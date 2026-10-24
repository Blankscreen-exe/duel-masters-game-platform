package dm12

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// TerradragonArqueDelacerna ...
func TerradragonArqueDelacerna(c *match.Card) {

	c.Name = "Terradragon Arque Delacerna"
	c.Power = 6000
	c.Civs = []string{civ.Nature}
	c.Family = []string{family.EarthDragon}
	c.ManaCost = 8
	c.ManaRequirement = []string{civ.Nature}

	c.Use(fx.Creature, fx.Doublebreaker, fx.PutIntoBattleZoneInsteadOfDiscard)

}
