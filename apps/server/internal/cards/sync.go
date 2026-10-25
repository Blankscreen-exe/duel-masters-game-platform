package cards

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"unicode"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/games"
)

// syncFormat changes whenever Sync starts writing the catalogue differently,
// so the next start copies it again even though the files are unchanged.
const syncFormat = "1"

// insertBatchSize bounds how many rows one INSERT writes.
const insertBatchSize = 500

// catalogueStateID is the id of the one row of card_catalogue_state.
const catalogueStateID = 1

// The catalogue's rows, as package cards reads and writes them.

type setRow struct {
	Code      string `gorm:"primaryKey"`
	SortOrder int
	IconPath  string
}

func (setRow) TableName() string { return "card_sets" }

type cardRow struct {
	ID         string `gorm:"primaryKey"`
	Name       string
	Type       string
	Cost       int
	Power      *int
	PowerText  string
	RulesText  string
	SearchText string
	EngineID   *string
	SortOrder  int
	InDatabase bool
}

func (cardRow) TableName() string { return "cards" }

type printingRow struct {
	ID              string `gorm:"primaryKey"`
	CardID          string
	SetCode         string
	SortOrder       int
	CollectorNumber string
	Rarity          string
	RarityCode      string
	ImagePath       string
}

func (printingRow) TableName() string { return "card_printings" }

type civilizationRow struct {
	CardID       string `gorm:"primaryKey"`
	Civilization string `gorm:"primaryKey"`
}

func (civilizationRow) TableName() string { return "card_civilizations" }

type raceRow struct {
	CardID    string `gorm:"primaryKey"`
	Race      string `gorm:"primaryKey"`
	SortOrder int
}

func (raceRow) TableName() string { return "card_races" }

type keywordRow struct {
	CardID  string `gorm:"primaryKey"`
	Keyword string `gorm:"primaryKey"`
}

func (keywordRow) TableName() string { return "card_keywords" }

type categoryRow struct {
	CardID   string `gorm:"primaryKey"`
	Category string `gorm:"primaryKey"`
}

func (categoryRow) TableName() string { return "card_categories" }

type abilityRow struct {
	CardID    string `gorm:"primaryKey"`
	SortOrder int    `gorm:"primaryKey"`
	Text      string
	Reminder  string
}

func (abilityRow) TableName() string { return "card_abilities" }

type stateRow struct {
	ID          int `gorm:"primaryKey"`
	Fingerprint string
	SyncedAt    time.Time
}

func (stateRow) TableName() string { return "card_catalogue_state" }

// detailTables are the tables holding a card's lists, rewritten whole on every
// copy.
var detailTables = []any{&civilizationRow{}, &raceRow{}, &keywordRow{}, &categoryRow{}, &abilityRow{}, &printingRow{}}

// SyncResult reports what Sync did.
type SyncResult struct {
	// Copied is false when the catalogue was already up to date.
	Copied bool
	Cards  int
	// Playable counts the cards linked to the engine.
	Playable int
	// Problems lists engine cards that could not be linked and printings
	// without a picture; see Build.
	Problems []string
}

// Sync copies the card database into the catalogue's tables, unless the files
// and the engine's cards are unchanged since the last copy. The copy is one
// transaction, so readers see the old catalogue or the new one, never a mix.
// Cards no longer in the card database are kept, marked as gone, so nothing
// referring to them breaks.
func Sync(ctx context.Context, db *gorm.DB, source Source, playable []games.PlayableCard, logger *slog.Logger) (SyncResult, error) {
	fingerprint := catalogueFingerprint(source, playable)

	var state stateRow
	err := db.WithContext(ctx).Take(&state, catalogueStateID).Error
	switch {
	case err == nil && state.Fingerprint == fingerprint:
		return SyncResult{}, nil
	case err != nil && !errors.Is(err, gorm.ErrRecordNotFound):
		return SyncResult{}, fmt.Errorf("reading the card catalogue's state: %w", err)
	}

	cards, problems := Build(source, playable)
	for _, problem := range problems {
		logger.WarnContext(ctx, "card catalogue: "+problem)
	}

	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := writeSets(tx, source.Sets); err != nil {
			return err
		}
		if err := writeCards(tx, cards); err != nil {
			return err
		}

		return tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{"fingerprint", "synced_at"}),
		}).Create(&stateRow{ID: catalogueStateID, Fingerprint: fingerprint, SyncedAt: tx.NowFunc()}).Error
	})
	if err != nil {
		return SyncResult{}, fmt.Errorf("copying the card catalogue: %w", err)
	}

	result := SyncResult{Copied: true, Cards: len(cards), Problems: problems}
	for _, card := range cards {
		if card.Playable() {
			result.Playable++
		}
	}
	logger.InfoContext(ctx, "copied the card catalogue from the card database", "cards", result.Cards, "playable", result.Playable, "problems", len(problems))

	return result, nil
}

// catalogueFingerprint identifies a copy: the card database's content, the
// engine's cards and the way Sync writes them.
func catalogueFingerprint(source Source, playable []games.PlayableCard) string {
	hash := sha256.New()
	fmt.Fprintf(hash, "format %s\nsource %s\n", syncFormat, source.Fingerprint)
	for _, set := range source.Sets {
		fmt.Fprintf(hash, "set %s %s\n", set.Code, set.IconPath)
	}
	for _, card := range playable {
		fmt.Fprintf(hash, "engine %s %s %s\n", card.EngineID, card.Name, card.Set)
	}

	return fmt.Sprintf("%x", hash.Sum(nil))
}

func writeSets(tx *gorm.DB, sets []SourceSet) error {
	if len(sets) == 0 {
		return nil
	}

	rows := make([]setRow, 0, len(sets))
	for i, set := range sets {
		rows = append(rows, setRow{Code: set.Code, SortOrder: i, IconPath: set.IconPath})
	}

	err := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "code"}},
		DoUpdates: clause.AssignmentColumns([]string{"sort_order", "icon_path"}),
	}).CreateInBatches(rows, insertBatchSize).Error
	if err != nil {
		return fmt.Errorf("writing the card sets: %w", err)
	}

	return nil
}

func writeCards(tx *gorm.DB, cards []Card) error {
	// Every card is marked gone and unlinked first, then those still in the
	// card database are written back. Unlinking first lets an engine id move
	// from one card to another without breaking its uniqueness half way.
	err := tx.Session(&gorm.Session{AllowGlobalUpdate: true}).Model(&cardRow{}).
		Updates(map[string]any{"in_database": false, "engine_id": nil}).Error
	if err != nil {
		return fmt.Errorf("marking the cards: %w", err)
	}

	for _, table := range detailTables {
		if err := tx.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(table).Error; err != nil {
			return fmt.Errorf("clearing the cards' details: %w", err)
		}
	}

	rows := cardRows{}
	for i, card := range cards {
		rows.add(card, i)
	}

	if len(rows.cards) > 0 {
		err = tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{"name", "type", "cost", "power", "power_text", "rules_text", "search_text", "engine_id", "sort_order", "in_database"}),
		}).CreateInBatches(rows.cards, insertBatchSize).Error
		if err != nil {
			return fmt.Errorf("writing the cards: %w", err)
		}
	}

	// One after another, stopping at the first failure: the transaction is
	// lost by then anyway.
	for _, write := range []func() error{
		func() error { return createAll(tx, "printings", rows.printings) },
		func() error { return createAll(tx, "civilizations", rows.civilizations) },
		func() error { return createAll(tx, "races", rows.races) },
		func() error { return createAll(tx, "keywords", rows.keywords) },
		func() error { return createAll(tx, "categories", rows.categories) },
		func() error { return createAll(tx, "abilities", rows.abilities) },
	} {
		if err := write(); err != nil {
			return err
		}
	}

	return nil
}

// createAll inserts rows in batches. GORM refuses an empty list, and a card
// database may well have, say, no categories at all.
func createAll[T any](tx *gorm.DB, what string, rows []T) error {
	if len(rows) == 0 {
		return nil
	}
	if err := tx.CreateInBatches(rows, insertBatchSize).Error; err != nil {
		return fmt.Errorf("writing the cards' %s: %w", what, err)
	}

	return nil
}

// cardRows are the rows of every card, gathered for writing in batches.
type cardRows struct {
	cards         []cardRow
	printings     []printingRow
	civilizations []civilizationRow
	races         []raceRow
	keywords      []keywordRow
	categories    []categoryRow
	abilities     []abilityRow
}

func (r *cardRows) add(card Card, sortOrder int) {
	row := cardRow{
		ID:         card.ID,
		Name:       card.Name,
		Type:       card.Type,
		Cost:       card.Cost,
		Power:      card.Power,
		PowerText:  card.PowerText,
		RulesText:  card.RulesText,
		SearchText: searchText(card.Name + "\n" + card.RulesText),
		SortOrder:  sortOrder,
		InDatabase: true,
	}
	if card.EngineID != "" {
		engineID := card.EngineID
		row.EngineID = &engineID
	}
	r.cards = append(r.cards, row)

	for i, printing := range card.Printings {
		r.printings = append(r.printings, printingRow{
			ID:              printing.ID,
			CardID:          card.ID,
			SetCode:         printing.SetCode,
			SortOrder:       i,
			CollectorNumber: printing.CollectorNumber,
			Rarity:          printing.Rarity,
			RarityCode:      printing.RarityCode,
			ImagePath:       printing.ImagePath,
		})
	}
	for _, civilization := range unique(card.Civilizations) {
		r.civilizations = append(r.civilizations, civilizationRow{CardID: card.ID, Civilization: civilization})
	}
	for i, race := range unique(card.Races) {
		r.races = append(r.races, raceRow{CardID: card.ID, Race: race, SortOrder: i})
	}
	for _, keyword := range unique(card.Keywords) {
		r.keywords = append(r.keywords, keywordRow{CardID: card.ID, Keyword: keyword})
	}
	for _, category := range unique(card.Categories) {
		r.categories = append(r.categories, categoryRow{CardID: card.ID, Category: category})
	}
	for i, ability := range card.Abilities {
		r.abilities = append(r.abilities, abilityRow{CardID: card.ID, SortOrder: i, Text: ability.Text, Reminder: ability.Reminder})
	}
}

// unique drops repeated values, keeping the first of each, since a list
// value is part of its table's key.
func unique(values []string) []string {
	seen := make(map[string]bool, len(values))
	kept := make([]string, 0, len(values))
	for _, value := range values {
		if !seen[value] {
			seen[value] = true
			kept = append(kept, value)
		}
	}

	return kept
}

// searchText folds text for searching: lowercase, without accents, with runs
// of spaces made single, so "Überdragon" is found by "uberdragon" on every
// database.
func searchText(text string) string {
	return strings.Join(strings.FieldsFunc(foldAccents(strings.ToLower(text)), unicode.IsSpace), " ")
}
