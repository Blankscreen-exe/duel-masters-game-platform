package cards

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"gorm.io/gorm"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/paging"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/store"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/validation"
)

// Search limits.
const (
	DefaultPageSize = 60
	MaxPageSize     = 100
	MaxQueryLength  = 100
)

// ErrNotFound is returned for a card, picture or set icon the catalogue does
// not have.
var ErrNotFound = errors.New("not in the card catalogue")

// Filter narrows a search. Empty lists and nil costs do not narrow it. A card
// matches a list when it has any of the listed values.
type Filter struct {
	// Query is looked for in the name and rules text, ignoring case and
	// accents.
	Query         string
	Civilizations []string
	Types         []string
	Races         []string
	Keywords      []string
	// Sets matches cards with a printing in any of them.
	Sets []string
	// MinCost and MaxCost bound the mana cost, inclusive.
	MinCost *int
	MaxCost *int
	// PlayableOnly keeps the cards the engine can play.
	PlayableOnly bool
	// Page counts from 1; 0 means the first page. PageSize is at most
	// MaxPageSize; 0 means DefaultPageSize.
	Page     int
	PageSize int
}

// Page is one page of search results, in name order.
type Page struct {
	Cards    []Card
	Total    int
	Page     int
	PageSize int
}

// Set is a set cards were printed in.
type Set struct {
	Code    string
	HasIcon bool
}

// FilterValues are the values a search can filter by, as the catalogue holds
// them.
type FilterValues struct {
	Civilizations []string
	Types         []string
	Races         []string
	Keywords      []string
	// Sets are in release order.
	Sets []Set
}

// Catalogue reads the card catalogue. It is safe for concurrent use.
type Catalogue struct {
	db *gorm.DB
	// dir is the card database's folder, where pictures and icons are.
	dir string
}

// NewCatalogue reads the catalogue from db, with pictures and icons from the
// card database in dir.
func NewCatalogue(db *gorm.DB, dir string) *Catalogue {
	return &Catalogue{db: db, dir: dir}
}

// Search returns one page of the cards matching filter. It returns a
// *validation.Error for a filter breaking the limits.
func (c *Catalogue) Search(ctx context.Context, filter Filter) (Page, error) {
	if err := normalizeFilter(&filter); err != nil {
		return Page{}, err
	}

	query := c.db.WithContext(ctx).Model(&cardRow{}).Where("in_database = ?", true)
	if filter.Query != "" {
		query = query.Where(store.Contains("search_text", searchText(filter.Query)))
	}
	if len(filter.Civilizations) > 0 {
		query = query.Where("id IN (?)", c.db.Model(&civilizationRow{}).Select("card_id").Where("civilization IN ?", filter.Civilizations))
	}
	if len(filter.Types) > 0 {
		query = query.Where("type IN ?", filter.Types)
	}
	if len(filter.Races) > 0 {
		query = query.Where("id IN (?)", c.db.Model(&raceRow{}).Select("card_id").Where("race IN ?", filter.Races))
	}
	if len(filter.Keywords) > 0 {
		query = query.Where("id IN (?)", c.db.Model(&keywordRow{}).Select("card_id").Where("keyword IN ?", filter.Keywords))
	}
	if len(filter.Sets) > 0 {
		query = query.Where("id IN (?)", c.db.Model(&printingRow{}).Select("card_id").Where("set_code IN ?", filter.Sets))
	}
	if filter.MinCost != nil {
		query = query.Where("cost >= ?", *filter.MinCost)
	}
	if filter.MaxCost != nil {
		query = query.Where("cost <= ?", *filter.MaxCost)
	}
	if filter.PlayableOnly {
		query = query.Where("engine_id IS NOT NULL")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return Page{}, fmt.Errorf("counting cards: %w", err)
	}

	var rows []cardRow
	err := query.Order("sort_order").Limit(filter.PageSize).Offset((filter.Page - 1) * filter.PageSize).Find(&rows).Error
	if err != nil {
		return Page{}, fmt.Errorf("searching cards: %w", err)
	}

	cards, err := c.withDetails(ctx, rows)
	if err != nil {
		return Page{}, err
	}

	return Page{Cards: cards, Total: int(total), Page: filter.Page, PageSize: filter.PageSize}, nil
}

// normalizeFilter fills in the default page and page size and checks the
// limits.
func normalizeFilter(filter *Filter) error {
	filter.Query = strings.TrimSpace(filter.Query)

	if utf8.RuneCountInString(filter.Query) > MaxQueryLength {
		return validation.New("query", fmt.Sprintf("must be at most %d characters", MaxQueryLength))
	}

	page := paging.Request{Page: filter.Page, PageSize: filter.PageSize}
	if err := page.Normalize(DefaultPageSize, MaxPageSize); err != nil {
		return err
	}
	filter.Page, filter.PageSize = page.Page, page.PageSize

	switch {
	case filter.MinCost != nil && *filter.MinCost < 0:
		return validation.New("minCost", "must be 0 or more")
	case filter.MaxCost != nil && *filter.MaxCost < 0:
		return validation.New("maxCost", "must be 0 or more")
	case filter.MinCost != nil && filter.MaxCost != nil && *filter.MinCost > *filter.MaxCost:
		return validation.New("minCost", "must not be above maxCost")
	}

	return nil
}

// Card returns a card still in the card database by its id. It returns
// ErrNotFound for any other id.
func (c *Catalogue) Card(ctx context.Context, id string) (Card, error) {
	var row cardRow
	err := c.db.WithContext(ctx).Where("id = ? AND in_database = ?", id, true).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Card{}, ErrNotFound
	}
	if err != nil {
		return Card{}, fmt.Errorf("finding card %q: %w", id, err)
	}

	cards, err := c.withDetails(ctx, []cardRow{row})
	if err != nil {
		return Card{}, err
	}

	return cards[0], nil
}

// Cards returns the cards still in the card database among ids, by id. Ids of
// other cards are left out.
func (c *Catalogue) Cards(ctx context.Context, ids []string) (map[string]Card, error) {
	found := make(map[string]Card, len(ids))
	if len(ids) == 0 {
		return found, nil
	}

	var rows []cardRow
	if err := c.db.WithContext(ctx).Where("id IN ? AND in_database = ?", unique(ids), true).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("finding cards: %w", err)
	}

	cards, err := c.withDetails(ctx, rows)
	if err != nil {
		return nil, err
	}
	for _, card := range cards {
		found[card.ID] = card
	}

	return found, nil
}

// ByNames finds cards still in the card database by name, as players type
// them: exactly, or else ignoring case, accents, spaces and punctuation, the
// way engine cards are linked. It returns the cards by the names given; a name
// matching no card, or several cards once case and punctuation are ignored, is
// left out.
func (c *Catalogue) ByNames(ctx context.Context, names []string) (map[string]Card, error) {
	found := make(map[string]Card, len(names))
	if len(names) == 0 {
		return found, nil
	}

	var rows []cardRow
	if err := c.db.WithContext(ctx).Select("id", "name").Where("in_database = ?", true).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("listing card names: %w", err)
	}

	byName := make(map[string]string, len(rows))
	byKey := make(map[string][]string, len(rows))
	for _, row := range rows {
		byName[row.Name] = row.ID
		key := nameKey(row.Name)
		byKey[key] = append(byKey[key], row.ID)
	}

	ids := make(map[string]string, len(names))
	for _, name := range names {
		if id, ok := byName[name]; ok {
			ids[name] = id
		} else if matches := byKey[nameKey(name)]; len(matches) == 1 && nameKey(name) != "" {
			ids[name] = matches[0]
		}
	}

	matched := make([]string, 0, len(ids))
	for _, id := range ids {
		matched = append(matched, id)
	}
	cards, err := c.Cards(ctx, matched)
	if err != nil {
		return nil, err
	}

	for name, id := range ids {
		found[name] = cards[id]
	}

	return found, nil
}

// withDetails turns rows into cards with their lists, in the rows' order.
func (c *Catalogue) withDetails(ctx context.Context, rows []cardRow) ([]Card, error) {
	cards := make([]Card, len(rows))
	index := make(map[string]int, len(rows))
	ids := make([]string, len(rows))
	for i, row := range rows {
		ids[i] = row.ID
		index[row.ID] = i
		cards[i] = Card{
			ID:            row.ID,
			Name:          row.Name,
			Type:          row.Type,
			Cost:          row.Cost,
			Power:         row.Power,
			PowerText:     row.PowerText,
			RulesText:     row.RulesText,
			Civilizations: []string{},
			Races:         []string{},
			Keywords:      []string{},
			Categories:    []string{},
			Abilities:     []Ability{},
			Printings:     []Printing{},
		}
		if row.EngineID != nil {
			cards[i].EngineID = *row.EngineID
		}
	}
	if len(rows) == 0 {
		return cards, nil
	}

	db := c.db.WithContext(ctx)

	var printings []printingRow
	if err := db.Where("card_id IN ?", ids).Order("card_id, sort_order").Find(&printings).Error; err != nil {
		return nil, fmt.Errorf("reading printings: %w", err)
	}
	for _, p := range printings {
		card := &cards[index[p.CardID]]
		card.Printings = append(card.Printings, Printing{
			ID:              p.ID,
			SetCode:         p.SetCode,
			CollectorNumber: p.CollectorNumber,
			Rarity:          p.Rarity,
			RarityCode:      p.RarityCode,
			ImagePath:       p.ImagePath,
		})
	}

	var civilizations []civilizationRow
	if err := db.Where("card_id IN ?", ids).Order("card_id, civilization").Find(&civilizations).Error; err != nil {
		return nil, fmt.Errorf("reading civilizations: %w", err)
	}
	for _, r := range civilizations {
		card := &cards[index[r.CardID]]
		card.Civilizations = append(card.Civilizations, r.Civilization)
	}

	var races []raceRow
	if err := db.Where("card_id IN ?", ids).Order("card_id, sort_order").Find(&races).Error; err != nil {
		return nil, fmt.Errorf("reading races: %w", err)
	}
	for _, r := range races {
		card := &cards[index[r.CardID]]
		card.Races = append(card.Races, r.Race)
	}

	var keywords []keywordRow
	if err := db.Where("card_id IN ?", ids).Order("card_id, keyword").Find(&keywords).Error; err != nil {
		return nil, fmt.Errorf("reading keywords: %w", err)
	}
	for _, r := range keywords {
		card := &cards[index[r.CardID]]
		card.Keywords = append(card.Keywords, r.Keyword)
	}

	var categories []categoryRow
	if err := db.Where("card_id IN ?", ids).Order("card_id, category").Find(&categories).Error; err != nil {
		return nil, fmt.Errorf("reading categories: %w", err)
	}
	for _, r := range categories {
		card := &cards[index[r.CardID]]
		card.Categories = append(card.Categories, r.Category)
	}

	var abilities []abilityRow
	if err := db.Where("card_id IN ?", ids).Order("card_id, sort_order").Find(&abilities).Error; err != nil {
		return nil, fmt.Errorf("reading abilities: %w", err)
	}
	for _, r := range abilities {
		card := &cards[index[r.CardID]]
		card.Abilities = append(card.Abilities, Ability{Text: r.Text, Reminder: r.Reminder})
	}

	return cards, nil
}

// FilterValues returns the values searches can filter by, from the cards still
// in the card database.
func (c *Catalogue) FilterValues(ctx context.Context) (FilterValues, error) {
	db := c.db.WithContext(ctx)
	// Built afresh for each use: a GORM query keeps state once run.
	inDatabase := func() *gorm.DB { return db.Model(&cardRow{}).Select("id").Where("in_database = ?", true) }
	values := FilterValues{}

	for _, list := range []struct {
		into   *[]string
		column string
		query  *gorm.DB
	}{
		{&values.Civilizations, "civilization", db.Model(&civilizationRow{}).Where("card_id IN (?)", inDatabase())},
		{&values.Types, "type", db.Model(&cardRow{}).Where("in_database = ?", true)},
		{&values.Races, "race", db.Model(&raceRow{}).Where("card_id IN (?)", inDatabase())},
		{&values.Keywords, "keyword", db.Model(&keywordRow{}).Where("card_id IN (?)", inDatabase())},
	} {
		*list.into = []string{}
		if err := list.query.Distinct().Order(list.column).Pluck(list.column, list.into).Error; err != nil {
			return FilterValues{}, fmt.Errorf("listing %s values: %w", list.column, err)
		}
	}

	var sets []setRow
	if err := db.Order("sort_order").Find(&sets).Error; err != nil {
		return FilterValues{}, fmt.Errorf("listing sets: %w", err)
	}
	values.Sets = make([]Set, 0, len(sets))
	for _, set := range sets {
		values.Sets = append(values.Sets, Set{Code: set.Code, HasIcon: set.IconPath != ""})
	}

	return values, nil
}

// ImageFile returns the file holding a picture, by printing id or by engine
// id, the latter giving the card's first printing. It returns ErrNotFound when
// there is no such printing, card or picture.
func (c *Catalogue) ImageFile(ctx context.Context, key string) (string, error) {
	var printing printingRow
	db := c.db.WithContext(ctx)

	err := db.Where("id = ?", key).Take(&printing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = db.Where("card_id = (?)", db.Model(&cardRow{}).Select("id").Where("engine_id = ?", key)).
			Order("sort_order").Take(&printing).Error
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("finding picture %q: %w", key, err)
	}

	return c.file(dataFolder, printing.ImagePath)
}

// SetIconFile returns the file holding a set's icon. It returns ErrNotFound
// when there is no such set or icon.
func (c *Catalogue) SetIconFile(ctx context.Context, code string) (string, error) {
	var set setRow
	err := c.db.WithContext(ctx).Where("code = ?", code).Take(&set).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("finding set %q: %w", code, err)
	}

	return c.file("", set.IconPath)
}

// file is the path of a file inside the card database's folder, or
// ErrNotFound for an empty or unsafe relative path.
func (c *Catalogue) file(folder, relative string) (string, error) {
	if relative == "" || !insideDataFolder(relative) {
		return "", ErrNotFound
	}

	return filepath.Join(c.dir, folder, filepath.FromSlash(relative)), nil
}
