package dm06

import (
	"fmt"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/cnd"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

func GraveWormQ(c *match.Card) {

	c.Name = "Grave Worm Q"
	c.Power = 3000
	c.Civs = []string{civ.Darkness}
	c.Family = []string{family.ParasiteWorm, family.Survivor}
	c.ManaCost = 5
	c.ManaRequirement = []string{civ.Darkness}

	c.Use(fx.Creature, fx.Survivor, fx.When(fx.MySurvivorSummoned, func(card *match.Card, ctx *match.Context) {
		fx.SelectFilter(
			card.Player,
			ctx.Match,
			card.Player,
			match.GRAVEYARD,
			"You may return a survivor from your graveyard to your hand",
			1,
			1,
			true,
			func(x *match.Card) bool { return x.HasFamily(family.Survivor) && x.HasCondition(cnd.Creature) },
			true,
		).Map(func(x *match.Card) {
			card.Player.MoveCard(x.ID, match.GRAVEYARD, match.HAND, card.ID)
			ctx.Match.ReportActionInChat(card.Player, fmt.Sprintf("%s was moved to %s's hand from their graveyard by Grave Worm Q's survivor ability", x.Name, card.Player.Username()))
		})

	}))

}

func TentacleWorm(c *match.Card) {

	c.Name = "Tentacle Worm"
	c.Power = 3000
	c.Civs = []string{civ.Darkness}
	c.Family = []string{family.ParasiteWorm}
	c.ManaCost = 4
	c.ManaRequirement = []string{civ.Darkness}

	c.Use(fx.Creature)

}
