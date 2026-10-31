// Package opponents keeps the computer opponents: the personalities players
// can practise against, each with its own play style, deck and the rules its
// deck keeps to. A computer opponent plays under an account of its own, whose
// name is its name, so its games are recorded like anyone's.
package opponents

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"time"

	"gorm.io/gorm"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/cards"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/computer"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/decks"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/formats"
)

// ErrNotFound is returned for an opponent that does not exist or is not
// offered.
var ErrNotFound = errors.New("computer opponent not found")

// Opponent is a computer opponent.
type Opponent struct {
	ID string
	// UserID is its account, which its games are recorded under.
	UserID string
	// Name is its account's name.
	Name        string
	Description string
	// Style is the name of how it plays (computer.StyleNamed).
	Style    string
	Strength computer.Strength
	// FormatID is the format its deck keeps to, and a player's deck must.
	FormatID string
	// Rules are what its deck keeps to beyond its format's.
	Rules decks.Rules
	// Enabled is whether players are offered it.
	Enabled bool
	// Cards are its deck, by card id.
	Cards []decks.Entry
}

// Deck is its deck as a game takes it.
func (o Opponent) Deck() decks.Deck {
	return decks.Deck{ID: o.ID, Name: o.Name, Cards: slices.Clone(o.Cards)}
}

// opponentRow is a row of computer_opponents, with its account's name.
type opponentRow struct {
	ID               string `gorm:"primaryKey"`
	UserID           string
	Description      string
	Style            string
	Strength         string
	FormatID         string
	MaxCivilizations int
	CardTypes        []string `gorm:"serializer:json"`
	Enabled          bool
	SortOrder        int
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (opponentRow) TableName() string { return "computer_opponents" }

// opponentCardRow is a row of computer_opponent_cards.
type opponentCardRow struct {
	ComputerOpponentID string `gorm:"primaryKey"`
	CardID             string `gorm:"primaryKey"`
	Count              int
}

func (opponentCardRow) TableName() string { return "computer_opponent_cards" }

// accountName is an opponent's account's name.
type accountName struct {
	ID       string
	Username string
}

// Service reads the computer opponents. It is safe for concurrent use.
type Service struct {
	db        *gorm.DB
	catalogue *cards.Catalogue
	formats   *formats.Service
}

// NewService returns the computer opponents, checking their decks against the
// catalogue's cards and the formats.
func NewService(db *gorm.DB, catalogue *cards.Catalogue, formatService *formats.Service) *Service {
	return &Service{db: db, catalogue: catalogue, formats: formatService}
}

// List returns the opponents players are offered, in order.
func (s *Service) List(ctx context.Context) ([]Opponent, error) {
	return s.read(ctx, s.db.WithContext(ctx).Where("enabled = ?", true))
}

// Get returns an opponent players are offered. It returns ErrNotFound for
// any other.
func (s *Service) Get(ctx context.Context, id string) (Opponent, error) {
	found, err := s.read(ctx, s.db.WithContext(ctx).Where("id = ? AND enabled = ?", id, true))
	if err != nil {
		return Opponent{}, err
	}
	if len(found) == 0 {
		return Opponent{}, ErrNotFound
	}
	return found[0], nil
}

// ByUser returns the opponent playing under an account, if it is offered. It
// returns ErrNotFound for any other.
func (s *Service) ByUser(ctx context.Context, userID string) (Opponent, error) {
	found, err := s.read(ctx, s.db.WithContext(ctx).Where("user_id = ? AND enabled = ?", userID, true))
	if err != nil {
		return Opponent{}, err
	}
	if len(found) == 0 {
		return Opponent{}, ErrNotFound
	}
	return found[0], nil
}

// Check lists every way an opponent's deck breaks its format's rules or its
// own, or that its format is retired, or nothing for a deck it can play.
func (s *Service) Check(ctx context.Context, opponent Opponent) ([]decks.Problem, error) {
	format, err := s.formats.ByID(ctx, opponent.FormatID)
	if err != nil {
		return nil, fmt.Errorf("finding %s's format: %w", opponent.Name, err)
	}

	ids := make([]string, 0, len(opponent.Cards))
	for _, entry := range opponent.Cards {
		ids = append(ids, entry.CardID)
	}
	known, err := s.catalogue.Cards(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("reading %s's cards: %w", opponent.Name, err)
	}

	var retired []decks.Problem
	if format.Retired {
		retired = []decks.Problem{{Kind: decks.ProblemFormatRetired}}
	}
	return slices.Concat(retired, decks.Check(opponent.Cards, known, format), decks.CheckRules(opponent.Cards, known, opponent.Rules)), nil
}

// read reads the opponents the query finds, in order, with their names and
// decks.
func (s *Service) read(ctx context.Context, query *gorm.DB) ([]Opponent, error) {
	var rows []opponentRow
	if err := query.Order("sort_order").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("reading the computer opponents: %w", err)
	}
	if len(rows) == 0 {
		return []Opponent{}, nil
	}

	userIDs := make([]string, 0, len(rows))
	for _, row := range rows {
		userIDs = append(userIDs, row.UserID)
	}
	var accounts []accountName
	if err := s.db.WithContext(ctx).Table("users").Select("id, username").Where("id IN ?", userIDs).Find(&accounts).Error; err != nil {
		return nil, fmt.Errorf("reading the computer opponents' names: %w", err)
	}
	names := make(map[string]string, len(accounts))
	for _, account := range accounts {
		names[account.ID] = account.Username
	}

	found := make([]Opponent, len(rows))
	index := make(map[string]int, len(rows))
	for i, row := range rows {
		index[row.ID] = i
		found[i] = Opponent{
			ID:          row.ID,
			UserID:      row.UserID,
			Name:        names[row.UserID],
			Description: row.Description,
			Style:       row.Style,
			Strength:    computer.Strength(row.Strength),
			FormatID:    row.FormatID,
			Rules:       decks.Rules{MaxCivilizations: row.MaxCivilizations, CardTypes: row.CardTypes},
			Enabled:     row.Enabled,
			Cards:       []decks.Entry{},
		}
	}

	var entries []opponentCardRow
	err := s.db.WithContext(ctx).Where("computer_opponent_id IN ?", slices.Collect(maps.Keys(index))).
		Order("computer_opponent_id, card_id").Find(&entries).Error
	if err != nil {
		return nil, fmt.Errorf("reading the computer opponents' decks: %w", err)
	}
	for _, entry := range entries {
		opponent := &found[index[entry.ComputerOpponentID]]
		opponent.Cards = append(opponent.Cards, decks.Entry{CardID: entry.CardID, Count: entry.Count})
	}

	return found, nil
}
