// Package formats keeps the formats games are played in: the sets each allows
// and the cards it bans or limits. Admins manage them; whether a deck keeps to
// a format is decided by package decks.
package formats

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"gorm.io/gorm"
)

// ErrNotFound is returned for a format that does not exist.
var ErrNotFound = errors.New("format not found")

// TotalCivilizations is how many civilizations the game has, the most a deck can
// hold.
const TotalCivilizations = 5

// Format is a set of deck-building rules on top of the game's own.
type Format struct {
	ID          string
	Name        string
	Description string
	// AllSets allows every set, including sets added to the card database
	// later. Otherwise only Sets are allowed.
	AllSets bool
	// Sets are the allowed set codes, in no particular order; empty when
	// AllSets.
	Sets []string
	// Limits are the banned and limited cards: the most copies of each card a
	// deck may have, 0 for a banned card.
	Limits map[string]int
	// MinCivilizations and MaxCivilizations are the fewest and the most
	// civilizations a deck's cards may have together, 0 leaving either open. A
	// card of two civilizations counts as both.
	MinCivilizations int
	MaxCivilizations int
	// Retired formats cannot be chosen for new games, and decks are no longer
	// checked against them; games already in them keep them.
	Retired bool
}

// AllowsSet reports whether cards printed in the set may be used.
func (f Format) AllowsSet(code string) bool {
	return f.AllSets || slices.Contains(f.Sets, code)
}

// Limit returns the most copies of a card the format allows, and false when
// the format leaves the card to the game's own limit.
func (f Format) Limit(cardID string) (int, bool) {
	limit, limited := f.Limits[cardID]
	return limit, limited
}

// The formats' rows, as this package reads them.

type formatRow struct {
	ID          string `gorm:"primaryKey"`
	Name        string
	Description string
	AllSets     bool
	SortOrder   int
	RetiredAt   *time.Time
	// See Format.
	MinCivilizations int
	MaxCivilizations int
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (formatRow) TableName() string { return "formats" }

type setRow struct {
	FormatID string `gorm:"primaryKey"`
	SetCode  string `gorm:"primaryKey"`
}

func (setRow) TableName() string { return "format_sets" }

type cardRow struct {
	FormatID  string `gorm:"primaryKey"`
	CardID    string `gorm:"primaryKey"`
	MaxCopies int
}

func (cardRow) TableName() string { return "format_cards" }

// Service reads the formats. It is safe for concurrent use.
type Service struct {
	db *gorm.DB
}

// NewService returns a service using db.
func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// List returns the formats players can choose, in the order admins set.
func (s *Service) List(ctx context.Context) ([]Format, error) {
	return s.read(ctx, s.db.WithContext(ctx).Where("retired_at IS NULL"))
}

// All returns every format, retired or not, in the order admins set.
func (s *Service) All(ctx context.Context) ([]Format, error) {
	return s.read(ctx, s.db.WithContext(ctx))
}

func (s *Service) read(ctx context.Context, query *gorm.DB) ([]Format, error) {
	var rows []formatRow
	if err := query.Order("sort_order, name").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("listing formats: %w", err)
	}

	return s.withRules(ctx, rows)
}

// ByID returns a format, retired or not. It returns ErrNotFound if there is
// none.
func (s *Service) ByID(ctx context.Context, id string) (Format, error) {
	var row formatRow
	err := s.db.WithContext(ctx).Where("id = ?", id).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Format{}, ErrNotFound
	}
	if err != nil {
		return Format{}, fmt.Errorf("finding format %q: %w", id, err)
	}

	found, err := s.withRules(ctx, []formatRow{row})
	if err != nil {
		return Format{}, err
	}

	return found[0], nil
}

// withRules turns rows into formats with their sets and card limits, in the
// rows' order.
func (s *Service) withRules(ctx context.Context, rows []formatRow) ([]Format, error) {
	formats := make([]Format, len(rows))
	index := make(map[string]int, len(rows))
	ids := make([]string, len(rows))
	for i, row := range rows {
		ids[i] = row.ID
		index[row.ID] = i
		formats[i] = Format{
			ID:          row.ID,
			Name:        row.Name,
			Description: row.Description,
			AllSets:     row.AllSets,
			Sets:        []string{},
			Limits:      map[string]int{},
			Retired:     row.RetiredAt != nil,

			MinCivilizations: row.MinCivilizations,
			MaxCivilizations: row.MaxCivilizations,
		}
	}
	if len(rows) == 0 {
		return formats, nil
	}

	db := s.db.WithContext(ctx)

	var sets []setRow
	if err := db.Where("format_id IN ?", ids).Order("format_id, set_code").Find(&sets).Error; err != nil {
		return nil, fmt.Errorf("reading format sets: %w", err)
	}
	for _, set := range sets {
		format := &formats[index[set.FormatID]]
		format.Sets = append(format.Sets, set.SetCode)
	}

	var limits []cardRow
	if err := db.Where("format_id IN ?", ids).Find(&limits).Error; err != nil {
		return nil, fmt.Errorf("reading format card limits: %w", err)
	}
	for _, limit := range limits {
		formats[index[limit.FormatID]].Limits[limit.CardID] = limit.MaxCopies
	}

	return formats, nil
}
