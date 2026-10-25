package decks

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Starter decks are ready-made decks every player may play without building
// one, or copy into their own decks to change. The first in order is every
// player's default.

// starterRow is a row of starter_decks.
type starterRow struct {
	ID          string `gorm:"primaryKey"`
	Name        string
	CoverCardID *string
	SortOrder   int
	Enabled     bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (starterRow) TableName() string { return "starter_decks" }

// starterCardRow is a row of starter_deck_cards.
type starterCardRow struct {
	StarterDeckID string `gorm:"primaryKey"`
	CardID        string `gorm:"primaryKey"`
	Count         int
}

func (starterCardRow) TableName() string { return "starter_deck_cards" }

// Starters returns the starter decks players are offered, the default first.
func (s *Service) Starters(ctx context.Context) ([]Deck, error) {
	var rows []starterRow
	if err := s.db.WithContext(ctx).Where("enabled = ?", true).Order("sort_order, name").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("listing the starter decks: %w", err)
	}
	return withStarterCards(s.db.WithContext(ctx), rows)
}

// Starter returns a starter deck players are offered. It returns ErrNotFound
// for one that does not exist or is not offered.
func (s *Service) Starter(ctx context.Context, deckID string) (Deck, error) {
	var row starterRow
	err := s.db.WithContext(ctx).Where("id = ? AND enabled = ?", deckID, true).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Deck{}, ErrNotFound
	}
	if err != nil {
		return Deck{}, fmt.Errorf("finding starter deck %q: %w", deckID, err)
	}

	found, err := withStarterCards(s.db.WithContext(ctx), []starterRow{row})
	if err != nil {
		return Deck{}, err
	}
	return found[0], nil
}

// Playable returns a deck a player may play: one of their own, or a starter
// deck. It returns ErrNotFound for any other.
func (s *Service) Playable(ctx context.Context, userID, deckID string) (Deck, error) {
	deck, err := s.Get(ctx, userID, deckID)
	if errors.Is(err, ErrNotFound) {
		return s.Starter(ctx, deckID)
	}
	return deck, err
}

// CopyStarter adds a copy of a starter deck to a player's own decks, for
// them to change. It returns ErrNotFound for a starter deck not offered, and
// the errors Create returns.
func (s *Service) CopyStarter(ctx context.Context, userID, deckID string) (Deck, error) {
	starter, err := s.Starter(ctx, deckID)
	if err != nil {
		return Deck{}, err
	}
	return s.Create(ctx, userID, Input{Name: starter.Name, CoverCardID: starter.CoverCardID, Cards: starter.Cards})
}

// withStarterCards turns starter deck rows into decks with their cards, in
// the rows' order.
func withStarterCards(db *gorm.DB, rows []starterRow) ([]Deck, error) {
	decks := make([]Deck, len(rows))
	ids := make([]string, len(rows))
	for i, row := range rows {
		ids[i] = row.ID
		decks[i] = Deck{ID: row.ID, Name: row.Name, Starter: true, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
		if row.CoverCardID != nil {
			decks[i].CoverCardID = *row.CoverCardID
		}
	}
	if len(rows) == 0 {
		return decks, nil
	}

	var entries []deckCardRow
	err := db.Model(&starterCardRow{}).Select("starter_deck_id AS deck_id, card_id, count").
		Where("starter_deck_id IN ?", ids).Order("starter_deck_id, card_id").Scan(&entries).Error
	if err != nil {
		return nil, fmt.Errorf("reading starter deck cards: %w", err)
	}
	return fillCards(decks, entries), nil
}

// starterList is a starter deck as the site begins with it: its cards by
// name, four copies of each.
type starterList struct {
	name  string
	cover string
	cards []string
}

// startingStarters are the starter decks a site begins with: one for each
// civilization, of cards from the first set, DM-01, so they keep to every
// format. Fire's is first, every player's default.
var startingStarters = []starterList{
	{
		name:  "Fire: Bolshack Rush",
		cover: "Bolshack Dragon",
		cards: []string{
			"Deadly Fighter Braid Claw", "Brawler Zyler", "Crimson Hammer", "Fire Sweeper Burning Hellion",
			"Super Explosive Volcanodon", "Meteosaur", "Draglide", "Tornado Flame", "Bolshack Dragon", "Gatling Skyterror",
		},
	},
	{
		name:  "Water: Depthcon Tide",
		cover: "King Depthcon",
		cards: []string{
			"Hunter Fish", "Spiral Gate", "Aqua Hulcus", "Candy Drop", "Phantom Fish",
			"Brain Serum", "Faerie Child", "Revolver Fish", "Aqua Knight", "King Depthcon",
		},
	},
	{
		name:  "Nature: Great-Horn Stampede",
		cover: "Roaring Great-Horn",
		cards: []string{
			"Poisonous Mushroom", "Bronze-Arm Tribe", "Golden Wing Striker", "Dimension Gate", "Forest Hornet",
			"Coiling Vines", "Deathblade Beetle", "Tri-horn Shepherd", "Natural Snare", "Roaring Great-Horn",
		},
	},
	{
		name:  "Light: Urth's Guardians",
		cover: "Urth, Purifying Elemental",
		cards: []string{
			"Reusol, the Oracle", "Solar Ray", "Iere, Vizier of Bullets", "Senatine Jade Tree", "Miele, Vizier of Lightning",
			"Lok, Vizier of Hunting", "Dia Nork, Moonlight Guardian", "Lah, Purification Enforcer", "Holy Awe", "Urth, Purifying Elemental",
		},
	},
	{
		name:  "Darkness: Zagaan's Shadow",
		cover: "Zagaan, Knight of Darkness",
		cards: []string{
			"Bloody Squito", "Ghost Touch", "Writhing Bone Ghoul", "Bone Spider", "Bone Assassin, the Ripper",
			"Dark Raven, Shadow of Grief", "Death Smoke", "Masked Horror, Shadow of Scorn", "Terror Pit", "Zagaan, Knight of Darkness",
		},
	},
}

// Seeds are the decks feature's starting data, for store.RunSeeds. A
// starter deck naming a card the card catalogue does not have, as another
// card database may not, is left out and logged.
func Seeds(logger *slog.Logger) []*gormigrate.Migration {
	return []*gormigrate.Migration{{
		ID: "0001_add_starter_decks",
		Migrate: func(tx *gorm.DB) error {
			return addStarters(tx, startingStarters, logger)
		},
	}}
}

// catalogueCard is a card's id and name, as the seed finds cards by name.
type catalogueCard struct {
	ID   string
	Name string
}

// addStarters adds starter decks, in order, finding their cards by name in
// the card catalogue. A deck naming a card the catalogue does not have is
// left out, so a starter deck is never added without its cards.
func addStarters(tx *gorm.DB, lists []starterList, logger *slog.Logger) error {
	var names []string
	for _, list := range lists {
		names = append(names, list.cover)
		names = append(names, list.cards...)
	}
	slices.Sort(names)
	names = slices.Compact(names)

	var found []catalogueCard
	if err := tx.Table("cards").Select("id, name").Where("name IN ?", names).Find(&found).Error; err != nil {
		return fmt.Errorf("finding the starter decks' cards: %w", err)
	}
	ids := make(map[string]string, len(found))
	for _, card := range found {
		ids[card.Name] = card.ID
	}

	now := tx.NowFunc()
	for order, list := range lists {
		var missing []string
		for _, name := range append([]string{list.cover}, list.cards...) {
			if _, known := ids[name]; !known {
				missing = append(missing, name)
			}
		}
		if len(missing) > 0 {
			logger.Warn("leaving out a starter deck whose cards the card database does not have", "deck", list.name, "missing", missing)
			continue
		}

		id, err := uuid.NewV7()
		if err != nil {
			return fmt.Errorf("making a starter deck id: %w", err)
		}
		cover := ids[list.cover]
		row := starterRow{ID: id.String(), Name: list.name, CoverCardID: &cover, SortOrder: order + 1, Enabled: true, CreatedAt: now, UpdatedAt: now}
		if err := tx.Create(&row).Error; err != nil {
			return fmt.Errorf("adding starter deck %q: %w", list.name, err)
		}

		entries := make([]starterCardRow, 0, len(list.cards))
		for _, name := range list.cards {
			entries = append(entries, starterCardRow{StarterDeckID: row.ID, CardID: ids[name], Count: MaxCopies})
		}
		if err := tx.Create(&entries).Error; err != nil {
			return fmt.Errorf("adding the cards of starter deck %q: %w", list.name, err)
		}
	}

	return nil
}
