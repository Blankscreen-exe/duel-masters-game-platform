package decks

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/cards"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/settings"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/validation"
)

// NameMaxLength is the longest deck name, in characters.
const NameMaxLength = 40

// MaxPerPlayer is the admin setting for how many decks a player can have.
var MaxPerPlayer = settings.Number{
	Key:         "decks.max_per_player",
	Group:       "Decks",
	Description: "The most decks one player can have. Players already over it keep their decks but cannot add more.",
	Default:     50,
	Min:         1,
	Max:         500,
}

// Settings are the settings this feature declares.
func Settings() []settings.Definition {
	return []settings.Definition{MaxPerPlayer}
}

// ErrNotFound is returned for a deck that does not exist or belongs to
// another player: players cannot tell the two apart.
var ErrNotFound = errors.New("deck not found")

// TooManyDecksError is returned when a player already has as many decks as
// allowed.
type TooManyDecksError struct {
	Limit int64
}

func (e *TooManyDecksError) Error() string {
	return fmt.Sprintf("a player can have at most %d decks", e.Limit)
}

// Deck is one of a player's decks, or a starter deck.
type Deck struct {
	ID string
	// UserID is the player whose deck it is; empty for a starter deck.
	UserID string
	// Starter is whether it is a starter deck, which every player may play.
	Starter bool
	Name    string
	// CoverCardID is the card the player chose to show for the deck, or empty
	// to let the site pick one (see Cover).
	CoverCardID string
	// Cards are by card id.
	Cards     []Entry
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Input is a deck as a player saves it. A deck may be saved unfinished, with
// fewer than Size cards, but never with cards it could not be played with.
type Input struct {
	Name        string
	CoverCardID string
	Cards       []Entry
}

// The decks' rows, as this package reads and writes them.

type deckRow struct {
	ID          string `gorm:"primaryKey"`
	UserID      string
	Name        string
	CoverCardID *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (deckRow) TableName() string { return "decks" }

type deckCardRow struct {
	DeckID string `gorm:"primaryKey"`
	CardID string `gorm:"primaryKey"`
	Count  int
}

func (deckCardRow) TableName() string { return "deck_cards" }

// Service keeps the players' decks. It is safe for concurrent use.
type Service struct {
	db        *gorm.DB
	catalogue *cards.Catalogue
	settings  *settings.Service
	logger    *slog.Logger

	// createMu makes counting a player's decks and adding one a single step,
	// so two decks created at the same moment cannot pass the limit. A lock in
	// memory is enough, since one server runs per database.
	createMu sync.Mutex
}

// NewService returns a service using db, checking cards against catalogue.
func NewService(db *gorm.DB, catalogue *cards.Catalogue, siteSettings *settings.Service, logger *slog.Logger) *Service {
	return &Service{db: db, catalogue: catalogue, settings: siteSettings, logger: logger}
}

// List returns a player's decks, the most recently changed first. Decks
// changed within the same tick of the clock come the newest first: ids are
// UUIDv7s, which sort by when they were made.
func (s *Service) List(ctx context.Context, userID string) ([]Deck, error) {
	var rows []deckRow
	if err := s.db.WithContext(ctx).Where("user_id = ?", userID).Order("updated_at DESC, id DESC").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("listing decks: %w", err)
	}

	return withCards(s.db.WithContext(ctx), rows)
}

// Get returns one of a player's decks. It returns ErrNotFound for a deck that
// does not exist or is another player's.
func (s *Service) Get(ctx context.Context, userID, deckID string) (Deck, error) {
	return s.get(s.db.WithContext(ctx), userID, deckID)
}

// get finds a deck through db, which may be a transaction.
func (s *Service) get(db *gorm.DB, userID, deckID string) (Deck, error) {
	var row deckRow
	err := db.Where("id = ? AND user_id = ?", deckID, userID).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Deck{}, ErrNotFound
	}
	if err != nil {
		return Deck{}, fmt.Errorf("finding deck %q: %w", deckID, err)
	}

	found, err := withCards(db, []deckRow{row})
	if err != nil {
		return Deck{}, err
	}

	return found[0], nil
}

// Create adds a deck for a player. It returns a *validation.Error for a deck
// breaking the rules and a *TooManyDecksError when the player has as many
// decks as allowed.
func (s *Service) Create(ctx context.Context, userID string, input Input) (Deck, error) {
	input, err := s.check(ctx, input)
	if err != nil {
		return Deck{}, err
	}

	id, err := uuid.NewV7()
	if err != nil {
		return Deck{}, fmt.Errorf("making a deck id: %w", err)
	}

	s.createMu.Lock()
	defer s.createMu.Unlock()

	limit := s.settings.Number(MaxPerPlayer)
	var deck Deck
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing int64
		if err := tx.Model(&deckRow{}).Where("user_id = ?", userID).Count(&existing).Error; err != nil {
			return fmt.Errorf("counting decks: %w", err)
		}
		if existing >= limit {
			return &TooManyDecksError{Limit: limit}
		}

		row := deckRow{ID: id.String(), UserID: userID, Name: input.Name, CoverCardID: optional(input.CoverCardID)}
		if err := tx.Create(&row).Error; err != nil {
			return fmt.Errorf("creating a deck: %w", err)
		}
		if err := writeCards(tx, row.ID, input.Cards); err != nil {
			return err
		}

		deck, err = s.get(tx, userID, row.ID)
		return err
	})
	if err != nil {
		return Deck{}, err
	}

	return deck, nil
}

// Update replaces a player's deck with input. It returns ErrNotFound for a
// deck that does not exist or is another player's, and a *validation.Error
// for a deck breaking the rules.
func (s *Service) Update(ctx context.Context, userID, deckID string, input Input) (Deck, error) {
	input, err := s.check(ctx, input)
	if err != nil {
		return Deck{}, err
	}

	var deck Deck
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&deckRow{}).Where("id = ? AND user_id = ?", deckID, userID).Updates(map[string]any{
			"name":          input.Name,
			"cover_card_id": optional(input.CoverCardID),
			"updated_at":    tx.NowFunc(),
		})
		if result.Error != nil {
			return fmt.Errorf("changing deck %q: %w", deckID, result.Error)
		}
		if result.RowsAffected == 0 {
			return ErrNotFound
		}

		if err := tx.Where("deck_id = ?", deckID).Delete(&deckCardRow{}).Error; err != nil {
			return fmt.Errorf("clearing deck %q: %w", deckID, err)
		}
		if err := writeCards(tx, deckID, input.Cards); err != nil {
			return err
		}

		deck, err = s.get(tx, userID, deckID)
		return err
	})
	if err != nil {
		return Deck{}, err
	}

	return deck, nil
}

// Delete removes a player's deck. It returns ErrNotFound for a deck that does
// not exist or is another player's.
func (s *Service) Delete(ctx context.Context, userID, deckID string) error {
	result := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", deckID, userID).Delete(&deckRow{})
	if result.Error != nil {
		return fmt.Errorf("deleting deck %q: %w", deckID, result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// check trims the name and checks a deck against the rules for saving one.
func (s *Service) check(ctx context.Context, input Input) (Input, error) {
	input.Name = strings.TrimSpace(input.Name)
	switch {
	case !utf8.ValidString(input.Name):
		return Input{}, validation.New("name", "must be valid text")
	case input.Name == "":
		return Input{}, validation.New("name", "must not be empty")
	case utf8.RuneCountInString(input.Name) > NameMaxLength:
		return Input{}, validation.New("name", fmt.Sprintf("must be at most %d characters", NameMaxLength))
	}

	ids := make([]string, 0, len(input.Cards))
	for _, entry := range input.Cards {
		if slices.Contains(ids, entry.CardID) {
			return Input{}, validation.New("cards", "must list each card once")
		}
		ids = append(ids, entry.CardID)
	}
	if total := Count(input.Cards); total > Size {
		return Input{}, validation.New("cards", fmt.Sprintf("must be at most %d, not %d", Size, total))
	}

	known, err := s.catalogue.Cards(ctx, ids)
	if err != nil {
		return Input{}, err
	}
	for _, entry := range input.Cards {
		card, found := known[entry.CardID]
		switch {
		case !found:
			return Input{}, validation.New("cards", fmt.Sprintf("include %q, which is not a card", entry.CardID))
		case !card.Playable():
			return Input{}, validation.New("cards", fmt.Sprintf("include %s, which cannot be played yet", card.Name))
		case entry.Count < 1 || entry.Count > MaxCopies:
			return Input{}, validation.New("cards", fmt.Sprintf("must have 1 to %d copies of %s, not %d", MaxCopies, card.Name, entry.Count))
		}
	}

	if input.CoverCardID != "" && !slices.Contains(ids, input.CoverCardID) {
		return Input{}, validation.New("coverCardId", "must be one of the deck's cards")
	}

	return input, nil
}

// writeCards adds a deck's cards.
func writeCards(tx *gorm.DB, deckID string, entries []Entry) error {
	if len(entries) == 0 {
		return nil
	}

	rows := make([]deckCardRow, 0, len(entries))
	for _, entry := range entries {
		rows = append(rows, deckCardRow{DeckID: deckID, CardID: entry.CardID, Count: entry.Count})
	}
	if err := tx.Create(&rows).Error; err != nil {
		return fmt.Errorf("writing the cards of deck %q: %w", deckID, err)
	}

	return nil
}

// withCards turns rows into decks with their cards, in the rows' order,
// reading through db, which may be a transaction.
func withCards(db *gorm.DB, rows []deckRow) ([]Deck, error) {
	decks := make([]Deck, len(rows))
	ids := make([]string, len(rows))
	for i, row := range rows {
		ids[i] = row.ID
		decks[i] = Deck{ID: row.ID, UserID: row.UserID, Name: row.Name, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
		if row.CoverCardID != nil {
			decks[i].CoverCardID = *row.CoverCardID
		}
	}
	if len(rows) == 0 {
		return decks, nil
	}

	var entries []deckCardRow
	if err := db.Where("deck_id IN ?", ids).Order("deck_id, card_id").Find(&entries).Error; err != nil {
		return nil, fmt.Errorf("reading deck cards: %w", err)
	}

	return fillCards(decks, entries), nil
}

// fillCards gives each deck its entries, by deck id, and every deck an empty
// list at least.
func fillCards(decks []Deck, entries []deckCardRow) []Deck {
	index := make(map[string]int, len(decks))
	for i := range decks {
		index[decks[i].ID] = i
		decks[i].Cards = []Entry{}
	}
	for _, entry := range entries {
		deck := &decks[index[entry.DeckID]]
		deck.Cards = append(deck.Cards, Entry{CardID: entry.CardID, Count: entry.Count})
	}
	return decks
}

// Imported is a deck list read from text: the cards found, and the lines that
// could not be used, with why.
type Imported struct {
	Cards    []Entry
	Problems []LineProblem
}

// LineProblem is a line of a deck list that could not be used, or not fully.
type LineProblem struct {
	Number int
	Text   string
	Reason string
}

// Import reads a deck list as ParseText does and finds its cards. A card
// listed on several lines is counted once with all its copies. Cards that
// cannot be played, and copies beyond MaxCopies, are left out and reported. It
// returns a *validation.Error for text beyond the limits.
func (s *Service) Import(ctx context.Context, text string) (Imported, error) {
	lines, err := ParseText(text)
	if err != nil {
		return Imported{}, err
	}

	names := make([]string, 0, 2*len(lines))
	for _, line := range lines {
		names = append(names, line.Name, line.Text)
	}
	found, err := s.catalogue.ByNames(ctx, names)
	if err != nil {
		return Imported{}, err
	}

	imported := Imported{Cards: []Entry{}, Problems: []LineProblem{}}
	position := map[string]int{}
	report := func(line Line, reason string) {
		imported.Problems = append(imported.Problems, LineProblem{Number: line.Number, Text: line.Text, Reason: reason})
	}

	for _, line := range lines {
		card, named := found[line.Name]
		count := line.Count
		// The whole line as a name, for cards such as "750 Man", when reading
		// it as a count and a name finds no card or an impossible count.
		if whole, ok := found[line.Text]; ok && line.Text != line.Name && (!named || count < 1 || count > Size) {
			card, named, count = whole, true, 1
		}

		switch {
		case !named:
			report(line, fmt.Sprintf("No card is called %q.", line.Name))
			continue
		case count < 1 || count > Size:
			report(line, fmt.Sprintf("The count must be between 1 and %d.", Size))
			continue
		case !card.Playable():
			report(line, fmt.Sprintf("%s cannot be played yet.", card.Name))
			continue
		}

		at, listed := position[card.ID]
		if !listed {
			at = len(imported.Cards)
			position[card.ID] = at
			imported.Cards = append(imported.Cards, Entry{CardID: card.ID})
		}

		entry := &imported.Cards[at]
		kept := min(entry.Count+count, MaxCopies)
		if left := entry.Count + count - kept; left > 0 {
			report(line, fmt.Sprintf("A deck can have at most %d copies of %s; %d left out.", MaxCopies, card.Name, left))
		}
		entry.Count = kept
	}

	return imported, nil
}

// optional stores an empty value as NULL.
func optional(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
