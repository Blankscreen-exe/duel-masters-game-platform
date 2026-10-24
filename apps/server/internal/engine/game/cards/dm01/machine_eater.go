package dm01

import (
	"fmt"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/civ"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/family"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/fx"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// ArtisanPicora ...
func ArtisanPicora(c *match.Card) {

	c.Name = "Artisan Picora"
	c.Power = 2000
	c.Civs = []string{civ.Fire}
	c.Family = []string{family.MachineEater}
	c.ManaCost = 1
	c.ManaRequirement = []string{civ.Fire}

	c.Use(fx.Creature, fx.When(fx.Summoned, func(card *match.Card, ctx *match.Context) {

		fx.Select(
			card.Player,
			ctx.Match,
			card.Player,
			match.MANAZONE,
			"Artisan Picora: Choose 1 card from your mana zone that will be sent to your graveyard",
			1,
			1,
			false,
		).Map(func(x *match.Card) {
			x.Player.MoveCard(x.ID, match.MANAZONE, match.GRAVEYARD, card.ID)
			ctx.Match.ReportActionInChat(card.Player, fmt.Sprintf("%s was sent to %s's graveyard by Artisan Picora", x.Name, x.Player.Username()))
		})

	}))

}

// NomadHeroGigio ...
func NomadHeroGigio(c *match.Card) {

	c.Name = "Nomad Hero Gigio"
	c.Power = 3000
	c.Civs = []string{civ.Fire}
	c.Family = []string{family.MachineEater}
	c.ManaCost = 5
	c.ManaRequirement = []string{civ.Fire}

	c.Use(fx.Creature, fx.AttackUntapped)

}
