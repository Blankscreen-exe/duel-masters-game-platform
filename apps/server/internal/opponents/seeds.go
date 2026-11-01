package opponents

import (
	"fmt"
	"log/slog"
	"slices"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/computer"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/decks"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/users"
)

// startingFormat is the format the starting opponents keep to: every card the
// game can play, so a player may bring any deck.
const startingFormat = "Open"

// BaseCardTypes are the card types a deck may have unless its rules say more:
// those the game plays today.
var BaseCardTypes = []string{"Creature", "Evolution Creature", "Spell"}

// startingOpponent is a computer opponent as the site begins with it: its
// deck by card name, decks.MaxCopies of each.
type startingOpponent struct {
	name             string
	description      string
	style            string
	maxCivilizations int
	cards            []string
}

// startingOpponents are the computer opponents a site begins with, in order:
// one for each theme.
var startingOpponents = []startingOpponent{
	{
		name:             "Grimwhisper",
		description:      "Empties your hand before you can use it.",
		style:            "discard",
		maxCivilizations: 1,
		cards: []string{
			"Ghost Touch", "Horrid Worm", "Cloned Nightmare", "Windmill Mutant", "Gigabalza",
			"Locomotiver", "Masked Horror, Shadow of Scorn", "Jagila, the Hidden Pillager", "Zero Nemesis, Shadow of Panic", "Gigavrand",
		},
	},
	{
		name:             "Blazerush",
		description:      "Floods the battle zone with small creatures and attacks every turn.",
		style:            "rush",
		maxCivilizations: 1,
		cards: []string{
			"Deadly Fighter Braid Claw", "Brawler Zyler", "Immortal Baron, Vorg", "Mini Titan Gett", "Crimson Hammer",
			"Explosive Dude Joe", "Pyrofighter Magnus", "Rikabu, the Dismantler", "Fatal Attacker Horvath", "Twin-Cannon Skyterror",
		},
	},
	{
		name:             "Aegis",
		description:      "Hides behind blockers and strikes when you cannot fight back.",
		style:            "wall",
		maxCivilizations: 1,
		cards: []string{
			"La Ura Giga, Sky Guardian", "Emerald Grass", "Iocant, the Oracle", "Senatine Jade Tree", "Bex, the Oracle",
			"Dia Nork, Moonlight Guardian", "Engbelt, the Spydroid", "Szubs Kin, Twilight Guardian", "Syforce, Aurora Elemental", "Sasha, Channeler of Suns",
		},
	},
	{
		name:             "Stormbinder",
		description:      "Taps your creatures so nothing stands in its way.",
		style:            "tapper",
		maxCivilizations: 2,
		cards: []string{
			"Solar Ray", "Aqua Trickster", "Miele, Vizier of Lightning", "Larba Geer, the Immaculate", "Mizoy, the Oracle",
			"Moonlight Flash", "Flohdani, the Spydroid", "Rom, Vizier of Tendrils", "Holy Awe", "Craze Valkyrie, the Drastic",
		},
	},
	{
		name:             "Tidecaller",
		description:      "Washes your creatures back to your hand to keep you behind.",
		style:            "bounce",
		maxCivilizations: 1,
		cards: []string{
			"Spiral Gate", "Stained Glass", "Wave Lance", "Emperor Maroll", "Unicorn Fish",
			"Cloned Spiral", "Crystal Paladin", "Saucer-Head Shark", "Aqua Surfer", "King Mazelan",
		},
	},
	{
		name:             "Titangrove",
		description:      "Builds mana fast, then plays the biggest creatures it can.",
		style:            "giant",
		maxCivilizations: 2,
		cards: []string{
			"Poisonous Mushroom", "Faerie Life", "Bronze-Arm Tribe", "Ultimate Force", "Bolshack Dragon",
			"Carnival Totem", "Roaring Great-Horn", "Gatling Skyterror", "Dawn Giant", "Bodacious Giant",
		},
	},
}

// StartingOpponent is a computer opponent a site begins with: its name, play
// style and deck by card name, decks.MaxCopies of each.
type StartingOpponent struct {
	Name  string
	Style string
	Cards []string
}

// StartingOpponents returns the computer opponents a site begins with, in
// order, such as for tests that play them.
func StartingOpponents() []StartingOpponent {
	list := make([]StartingOpponent, 0, len(startingOpponents))
	for _, opponent := range startingOpponents {
		list = append(list, StartingOpponent{Name: opponent.name, Style: opponent.style, Cards: slices.Clone(opponent.cards)})
	}
	return list
}

// Seeds are the computer opponents' starting data, for store.RunSeeds. An
// opponent naming a card the card catalogue does not have, as another card
// database may not, or whose name an account already has, is left out and
// logged.
func Seeds(logger *slog.Logger) []*gormigrate.Migration {
	return []*gormigrate.Migration{{
		ID: "0001_add_computer_opponents",
		Migrate: func(tx *gorm.DB) error {
			return addOpponents(tx, startingOpponents, logger)
		},
	}, {
		ID: "0002_add_live_opponent",
		Migrate: func(tx *gorm.DB) error {
			return addLiveOpponent(tx, logger)
		},
	}}
}

// catalogueCard is a card's id and name, as the seed finds cards by name.
type catalogueCard struct {
	ID   string
	Name string
}

// addOpponents adds computer opponents, in order, with their accounts,
// finding their cards by name in the card catalogue.
func addOpponents(tx *gorm.DB, starting []startingOpponent, logger *slog.Logger) error {
	var formatID string
	if err := tx.Table("formats").Select("id").Where("name = ?", startingFormat).Scan(&formatID).Error; err != nil {
		return fmt.Errorf("finding the %s format: %w", startingFormat, err)
	}
	if formatID == "" {
		logger.Warn("leaving out the computer opponents, as the format they keep to is gone", "format", startingFormat)
		return nil
	}

	var names []string
	for _, opponent := range starting {
		names = append(names, opponent.cards...)
	}
	slices.Sort(names)
	names = slices.Compact(names)

	var found []catalogueCard
	if err := tx.Table("cards").Select("id, name").Where("name IN ?", names).Find(&found).Error; err != nil {
		return fmt.Errorf("finding the computer opponents' cards: %w", err)
	}
	ids := make(map[string]string, len(found))
	for _, card := range found {
		ids[card.Name] = card.ID
	}

	now := tx.NowFunc()
	for order, opponent := range starting {
		if _, ok := computer.StyleNamed(opponent.style); !ok {
			return fmt.Errorf("computer opponent %q has an unknown style %q", opponent.name, opponent.style)
		}

		var missing []string
		for _, name := range opponent.cards {
			if _, known := ids[name]; !known {
				missing = append(missing, name)
			}
		}
		if len(missing) > 0 {
			logger.Warn("leaving out a computer opponent whose cards the card database does not have", "opponent", opponent.name, "missing", missing)
			continue
		}

		taken, err := users.TakenIn(tx, opponent.name)
		if err != nil {
			return err
		}
		if taken {
			logger.Warn("leaving out a computer opponent whose name an account already has", "opponent", opponent.name)
			continue
		}
		account, err := users.CreateComputerIn(tx, opponent.name)
		if err != nil {
			return err
		}

		id, err := uuid.NewV7()
		if err != nil {
			return fmt.Errorf("making a computer opponent id: %w", err)
		}
		row := opponentRow{
			ID:               id.String(),
			UserID:           account.ID,
			Description:      opponent.description,
			Style:            opponent.style,
			Strength:         string(computer.Normal),
			FormatID:         formatID,
			MaxCivilizations: opponent.maxCivilizations,
			CardTypes:        BaseCardTypes,
			Enabled:          true,
			SortOrder:        order + 1,
			CreatedAt:        now,
			UpdatedAt:        now,
		}
		deck := make([]decks.Entry, 0, len(opponent.cards))
		for _, name := range opponent.cards {
			deck = append(deck, decks.Entry{CardID: ids[name], Count: decks.MaxCopies})
		}
		if err := insert(tx, row, deck); err != nil {
			return fmt.Errorf("adding computer opponent %q: %w", opponent.name, err)
		}
	}

	return nil
}
