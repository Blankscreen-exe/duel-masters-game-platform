package formats

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/adminlog"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/cards"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/users"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/validation"
)

const (
	// NameMaxLength is the longest name, in characters.
	NameMaxLength = 40
	// DescriptionMaxLength is the longest description, in characters.
	DescriptionMaxLength = 200
	// MaxLimit is the most copies a limit may allow; any more is no limit, as
	// the game itself allows only a few.
	MaxLimit = 2
)

var (
	// ErrNameTaken is returned for a name another format has, in any case.
	ErrNameTaken = errors.New("a format already has this name")
	// ErrLastFormat is returned for retiring the only format players can still
	// choose, which would leave them unable to start a game.
	ErrLastFormat = errors.New("the last format cannot be retired")
)

// Input is a format as an admin saves it.
type Input struct {
	Name        string
	Description string
	// AllSets allows every set; otherwise at least one of Sets is needed.
	AllSets bool
	Sets    []string
	// Limits are the most copies of each card a deck may have, 0 for banned.
	Limits map[string]int
}

// Manager lets admins change the formats. It is safe for concurrent use.
type Manager struct {
	db        *gorm.DB
	formats   *Service
	catalogue *cards.Catalogue
}

// NewManager returns a manager using db, checking sets and cards against the
// catalogue.
func NewManager(db *gorm.DB, formats *Service, catalogue *cards.Catalogue) *Manager {
	return &Manager{db: db, formats: formats, catalogue: catalogue}
}

// Create adds a format, last in order. It returns a *validation.Error for input
// breaking the rules, ErrNameTaken for a name another format has, and
// users.ErrNotAdmin for an account no longer an admin.
func (m *Manager) Create(ctx context.Context, actor users.User, input Input) (Format, error) {
	input, err := m.normalize(ctx, input)
	if err != nil {
		return Format{}, err
	}
	id, err := uuid.NewV7()
	if err != nil {
		return Format{}, fmt.Errorf("making a format id: %w", err)
	}

	err = m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		admin, err := users.LockAdminIn(tx, actor.ID, tx.NowFunc())
		if err != nil {
			return err
		}
		if err := ensureNameFree(tx, input.Name, ""); err != nil {
			return err
		}

		var last int
		if err := tx.Model(&formatRow{}).Select("COALESCE(MAX(sort_order), 0)").Scan(&last).Error; err != nil {
			return fmt.Errorf("finding the last format: %w", err)
		}
		now := tx.NowFunc()
		row := formatRow{
			ID:          id.String(),
			Name:        input.Name,
			Description: input.Description,
			AllSets:     input.AllSets,
			SortOrder:   last + 1,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if err := tx.Create(&row).Error; err != nil {
			return nameError(err, "adding a format")
		}
		if err := insertRules(tx, row.ID, input); err != nil {
			return err
		}
		return adminlog.RecordIn(tx, admin, adminlog.FormatAdded, input.Name, "")
	})
	if err != nil {
		return Format{}, err
	}
	return m.formats.ByID(ctx, id.String())
}

// Update changes a format. Games already running carry on under the rules they
// started with; everything else follows the new rules at once. It returns what
// Create returns, and ErrNotFound for a format that does not exist.
func (m *Manager) Update(ctx context.Context, actor users.User, id string, input Input) (Format, error) {
	current, err := m.formats.ByID(ctx, id)
	if err != nil {
		return Format{}, err
	}
	input, err = m.normalize(ctx, input)
	if err != nil {
		return Format{}, err
	}

	err = m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		admin, err := users.LockAdminIn(tx, actor.ID, tx.NowFunc())
		if err != nil {
			return err
		}
		if err := ensureNameFree(tx, input.Name, id); err != nil {
			return err
		}

		err = tx.Model(&formatRow{ID: id}).
			Select("name", "description", "all_sets", "updated_at").
			Updates(formatRow{Name: input.Name, Description: input.Description, AllSets: input.AllSets, UpdatedAt: tx.NowFunc()}).Error
		if err != nil {
			return nameError(err, fmt.Sprintf("changing format %q", current.Name))
		}
		if err := tx.Where("format_id = ?", id).Delete(&setRow{}).Error; err != nil {
			return fmt.Errorf("replacing the sets of format %q: %w", current.Name, err)
		}
		if err := tx.Where("format_id = ?", id).Delete(&cardRow{}).Error; err != nil {
			return fmt.Errorf("replacing the limits of format %q: %w", current.Name, err)
		}
		if err := insertRules(tx, id, input); err != nil {
			return err
		}

		details := ""
		if current.Name != input.Name {
			details = "Was called " + current.Name
		}
		return adminlog.RecordIn(tx, admin, adminlog.FormatChanged, input.Name, details)
	})
	if err != nil {
		return Format{}, err
	}
	return m.formats.ByID(ctx, id)
}

// SetRetired retires a format or brings it back. Once retired it cannot be
// chosen for new games and decks are not checked against it, while games
// running, games already open and past games keep it. It returns ErrNotFound for
// a format that does not exist, ErrLastFormat for the only one left to choose,
// and users.ErrNotAdmin for an account no longer an admin. Asking for what
// already holds records nothing.
func (m *Manager) SetRetired(ctx context.Context, actor users.User, id string, retired bool) (Format, error) {
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		admin, err := users.LockAdminIn(tx, actor.ID, tx.NowFunc())
		if err != nil {
			return err
		}

		var row formatRow
		err = tx.Where("id = ?", id).Take(&row).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("finding format %q: %w", id, err)
		}
		if (row.RetiredAt != nil) == retired {
			return nil
		}

		action := adminlog.FormatRestored
		now := tx.NowFunc()
		var retiredAt *time.Time
		if retired {
			var others int64
			if err := tx.Model(&formatRow{}).Where("retired_at IS NULL AND id <> ?", id).Count(&others).Error; err != nil {
				return fmt.Errorf("counting the formats left: %w", err)
			}
			if others == 0 {
				return ErrLastFormat
			}
			action, retiredAt = adminlog.FormatRetired, &now
		}

		err = tx.Model(&formatRow{ID: id}).Select("retired_at", "updated_at").
			Updates(formatRow{RetiredAt: retiredAt, UpdatedAt: now}).Error
		if err != nil {
			return fmt.Errorf("changing whether %q is retired: %w", row.Name, err)
		}
		return adminlog.RecordIn(tx, admin, action, row.Name, "")
	})
	if err != nil {
		return Format{}, err
	}
	return m.formats.ByID(ctx, id)
}

// Reorder puts every format, retired or not, in the order given; the first
// players can choose is what the site picks for them to begin with. It returns a
// *validation.Error for ids that are not each format once, and
// users.ErrNotAdmin for an account no longer an admin.
func (m *Manager) Reorder(ctx context.Context, actor users.User, ids []string) ([]Format, error) {
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		admin, err := users.LockAdminIn(tx, actor.ID, tx.NowFunc())
		if err != nil {
			return err
		}

		var rows []formatRow
		if err := tx.Find(&rows).Error; err != nil {
			return fmt.Errorf("reading the formats: %w", err)
		}
		names := make(map[string]string, len(rows))
		for _, row := range rows {
			names[row.ID] = row.Name
		}
		if len(ids) != len(rows) || len(slices.Compact(slices.Sorted(slices.Values(ids)))) != len(rows) {
			return validation.New("formatIds", "must name each format once")
		}

		ordered := make([]string, 0, len(ids))
		for i, id := range ids {
			name, known := names[id]
			if !known {
				return validation.New("formatIds", "must name each format once")
			}
			ordered = append(ordered, name)

			err := tx.Model(&formatRow{ID: id}).Select("sort_order").Updates(formatRow{SortOrder: i + 1}).Error
			if err != nil {
				return fmt.Errorf("ordering format %q: %w", name, err)
			}
		}
		return adminlog.RecordIn(tx, admin, adminlog.FormatsReordered, "Formats", strings.Join(ordered, ", "))
	})
	if err != nil {
		return nil, err
	}
	return m.formats.All(ctx)
}

// normalize trims a format's input and checks it, returning a *validation.Error
// for anything breaking the rules.
func (m *Manager) normalize(ctx context.Context, input Input) (Input, error) {
	input.Name = strings.TrimSpace(input.Name)
	switch {
	case !utf8.ValidString(input.Name):
		return Input{}, validation.New("name", "must be valid text")
	case input.Name == "":
		return Input{}, validation.New("name", "must not be empty")
	case strings.ContainsFunc(input.Name, unicode.IsControl):
		return Input{}, validation.New("name", "must be one line of text")
	case utf8.RuneCountInString(input.Name) > NameMaxLength:
		return Input{}, validation.New("name", fmt.Sprintf("must be at most %d characters", NameMaxLength))
	}

	input.Description = strings.TrimSpace(input.Description)
	switch {
	case !utf8.ValidString(input.Description):
		return Input{}, validation.New("description", "must be valid text")
	case input.Description == "":
		return Input{}, validation.New("description", "must not be empty")
	case strings.ContainsFunc(input.Description, unicode.IsControl):
		return Input{}, validation.New("description", "must be one line of text")
	case utf8.RuneCountInString(input.Description) > DescriptionMaxLength:
		return Input{}, validation.New("description", fmt.Sprintf("must be at most %d characters", DescriptionMaxLength))
	}

	if err := m.checkSets(ctx, &input); err != nil {
		return Input{}, err
	}
	if err := m.checkLimits(ctx, input); err != nil {
		return Input{}, err
	}
	return input, nil
}

// checkSets checks the sets are ones the card database has, and leaves each once
// in order; a format allowing every set lists none.
func (m *Manager) checkSets(ctx context.Context, input *Input) error {
	if input.AllSets {
		input.Sets = nil
		return nil
	}
	if len(input.Sets) == 0 {
		return validation.New("sets", "must allow at least one set")
	}

	values, err := m.catalogue.FilterValues(ctx)
	if err != nil {
		return err
	}
	known := make(map[string]bool, len(values.Sets))
	for _, set := range values.Sets {
		known[set.Code] = true
	}
	for _, code := range input.Sets {
		if !known[code] {
			return validation.New("sets", fmt.Sprintf("must be sets the card database has, not %q", code))
		}
	}
	input.Sets = slices.Compact(slices.Sorted(slices.Values(input.Sets)))
	return nil
}

// checkLimits checks each limit is within bounds and on a card the card database
// has.
func (m *Manager) checkLimits(ctx context.Context, input Input) error {
	ids := make([]string, 0, len(input.Limits))
	for id, limit := range input.Limits {
		if limit < 0 || limit > MaxLimit {
			return validation.New("limits", fmt.Sprintf("must allow from 0 to %d copies", MaxLimit))
		}
		ids = append(ids, id)
	}

	known, err := m.catalogue.Cards(ctx, ids)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if _, found := known[id]; !found {
			return validation.New("limits", fmt.Sprintf("must be on cards the card database has, not %q", id))
		}
	}
	return nil
}

// ensureNameFree returns ErrNameTaken for a name, in any case, of a format other
// than the one with id except.
func ensureNameFree(tx *gorm.DB, name, except string) error {
	var rows []formatRow
	if err := tx.Select("id", "name").Find(&rows).Error; err != nil {
		return fmt.Errorf("reading the formats' names: %w", err)
	}
	for _, row := range rows {
		if row.ID != except && strings.EqualFold(row.Name, name) {
			return ErrNameTaken
		}
	}
	return nil
}

// nameError is ErrNameTaken when two admins took the same name at once, which
// only the database's unique index can catch; otherwise err, described.
func nameError(err error, doing string) error {
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return ErrNameTaken
	}
	return fmt.Errorf("%s: %w", doing, err)
}

// insertRules adds a format's sets and limits.
func insertRules(tx *gorm.DB, id string, input Input) error {
	if len(input.Sets) > 0 {
		sets := make([]setRow, 0, len(input.Sets))
		for _, code := range input.Sets {
			sets = append(sets, setRow{FormatID: id, SetCode: code})
		}
		if err := tx.Create(&sets).Error; err != nil {
			return fmt.Errorf("adding a format's sets: %w", err)
		}
	}

	if len(input.Limits) > 0 {
		limits := make([]cardRow, 0, len(input.Limits))
		for cardID, limit := range input.Limits {
			limits = append(limits, cardRow{FormatID: id, CardID: cardID, MaxCopies: limit})
		}
		if err := tx.Create(&limits).Error; err != nil {
			return fmt.Errorf("adding a format's limits: %w", err)
		}
	}
	return nil
}
