package dm06

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

func SteamStar(c *match.Card) {

	c.Name = "Steam Star"
	c.Power = 1000
	c.Civs = []string{civ.Water}
	c.Family = []string{family.CyberVirus}
	c.ManaCost = 2
	c.ManaRequirement = []string{civ.Water}

	c.Use(fx.Creature)
}

func RippleLotusQ(c *match.Card) {

	c.Name = "Ripple Lotus Q"
	c.Power = 2000
	c.Civs = []string{civ.Water}
	c.Family = []string{family.CyberVirus, family.Survivor}
	c.ManaCost = 6
	c.ManaRequirement = []string{civ.Water}

	c.Use(fx.Creature, fx.Survivor, fx.When(fx.MySurvivorSummoned, fx.MayTapOpCreature))

}
