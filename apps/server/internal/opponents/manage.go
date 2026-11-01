package opponents

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/adminlog"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/computer"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/decks"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/formats"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/users"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/validation"
)

const (
	// DescriptionMaxLength is the longest description, in characters.
	DescriptionMaxLength = 200
	// MaxCivilizationsLimit is the most a deck's rules can ask for: every
	// civilization there is.
	MaxCivilizationsLimit = formats.TotalCivilizations
)

// CardTypes are every card type a deck's rules may allow, as the card database
// names them, including those the game cannot play yet: a deck can only hold
// them once it can.
var CardTypes = []string{"Creature", "Evolution Creature", "Spell", "Castle", "Cross Gear", "Evolution Cross Gear"}

// UnplayableError is returned for making or keeping an opponent offered with a
// deck that breaks its rules. Players are never offered one; retire it to work
// on its deck.
type UnplayableError struct {
	Name     string
	Problems []decks.Problem
}

func (e *UnplayableError) Error() string {
	return fmt.Sprintf("%s's deck cannot be played: %s", e.Name, e.Problems[0].Message())
}

// Input is a computer opponent as an admin saves it. Its deck may be unfinished
// while it is retired.
type Input struct {
	Name        string
	Description string
	Style       string
	Strength    computer.Strength
	FormatID    string
	Rules       decks.Rules
	Cards       []decks.Entry
}

// All returns every opponent, retired or not, in order, for the admin panel.
// The live opponent is not among them: it builds its own deck, and a setting
// switches it on.
func (s *Service) All(ctx context.Context) ([]Opponent, error) {
	return s.read(ctx, s.db.WithContext(ctx).Where("style <> ?", LiveStyle))
}

// ByID returns an opponent, retired or not, for the admin panel. It returns
// ErrNotFound for one that does not exist.
func (s *Service) ByID(ctx context.Context, id string) (Opponent, error) {
	found, err := s.read(ctx, s.db.WithContext(ctx).Where("id = ? AND style <> ?", id, LiveStyle))
	if err != nil {
		return Opponent{}, err
	}
	if len(found) == 0 {
		return Opponent{}, ErrNotFound
	}
	return found[0], nil
}

// Create adds an opponent with an account of its own, last in order. It starts
// retired, to be offered once its deck is ready. It returns a
// *validation.Error for input breaking the rules, users.ErrUsernameTaken for a
// name an account has, and users.ErrNotAdmin for an account no longer an admin.
func (s *Service) Create(ctx context.Context, actor users.User, input Input) (Opponent, error) {
	input, err := s.normalize(ctx, input)
	if err != nil {
		return Opponent{}, err
	}
	id, err := uuid.NewV7()
	if err != nil {
		return Opponent{}, fmt.Errorf("making a computer opponent id: %w", err)
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		admin, err := users.LockAdminIn(tx, actor.ID, tx.NowFunc())
		if err != nil {
			return err
		}

		taken, err := users.TakenIn(tx, input.Name)
		if err != nil {
			return err
		}
		if taken {
			return users.ErrUsernameTaken
		}
		account, err := users.CreateComputerIn(tx, input.Name)
		if err != nil {
			return err
		}

		var last int
		if err := tx.Model(&opponentRow{}).Select("COALESCE(MAX(sort_order), 0)").Scan(&last).Error; err != nil {
			return fmt.Errorf("finding the last computer opponent: %w", err)
		}
		now := tx.NowFunc()
		row := opponentRow{
			ID:               id.String(),
			UserID:           account.ID,
			Description:      input.Description,
			Style:            input.Style,
			Strength:         string(input.Strength),
			FormatID:         input.FormatID,
			MaxCivilizations: input.Rules.MaxCivilizations,
			CardTypes:        input.Rules.CardTypes,
			SortOrder:        last + 1,
			CreatedAt:        now,
			UpdatedAt:        now,
		}
		if err := insert(tx, row, input.Cards); err != nil {
			return err
		}
		return adminlog.RecordIn(tx, admin, adminlog.ComputerOpponentAdded, account.Username, "")
	})
	if err != nil {
		return Opponent{}, err
	}
	return s.ByID(ctx, id.String())
}

// Update changes an opponent. Games already running carry on as they started.
// It returns what Create returns, ErrNotFound for an opponent that does not
// exist, and an *UnplayableError for an offered one's deck breaking its rules.
func (s *Service) Update(ctx context.Context, actor users.User, id string, input Input) (Opponent, error) {
	current, err := s.ByID(ctx, id)
	if err != nil {
		return Opponent{}, err
	}
	input, err = s.normalize(ctx, input)
	if err != nil {
		return Opponent{}, err
	}

	if current.Enabled {
		changed := current
		changed.Name, changed.FormatID, changed.Rules, changed.Cards = input.Name, input.FormatID, input.Rules, input.Cards
		if err := s.ensurePlayable(ctx, changed); err != nil {
			return Opponent{}, err
		}
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		admin, err := users.LockAdminIn(tx, actor.ID, tx.NowFunc())
		if err != nil {
			return err
		}
		if err := users.RenameComputerIn(tx, current.UserID, input.Name); err != nil {
			return err
		}

		err = tx.Model(&opponentRow{ID: id}).
			Select("description", "style", "strength", "format_id", "max_civilizations", "card_types", "updated_at").
			Updates(opponentRow{
				Description:      input.Description,
				Style:            input.Style,
				Strength:         string(input.Strength),
				FormatID:         input.FormatID,
				MaxCivilizations: input.Rules.MaxCivilizations,
				CardTypes:        input.Rules.CardTypes,
				UpdatedAt:        tx.NowFunc(),
			}).Error
		if err != nil {
			return fmt.Errorf("changing computer opponent %q: %w", current.Name, err)
		}
		if err := replaceCards(tx, id, current.Name, input.Cards); err != nil {
			return err
		}

		details := ""
		if current.Name != input.Name {
			details = "Was called " + current.Name
		}
		return adminlog.RecordIn(tx, admin, adminlog.ComputerOpponentChanged, input.Name, details)
	})
	if err != nil {
		return Opponent{}, err
	}
	return s.ByID(ctx, id)
}

// SetEnabled offers an opponent to players, or retires it: running games carry
// on, and a rematch being decided against it ends when the player agrees. It
// returns ErrNotFound for one that does not exist, an *UnplayableError for
// offering one whose deck breaks its rules, and users.ErrNotAdmin for an account
// no longer an admin. Asking for what already holds records nothing.
func (s *Service) SetEnabled(ctx context.Context, actor users.User, id string, enabled bool) (Opponent, error) {
	current, err := s.ByID(ctx, id)
	if err != nil {
		return Opponent{}, err
	}
	if enabled {
		if err := s.ensurePlayable(ctx, current); err != nil {
			return Opponent{}, err
		}
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		admin, err := users.LockAdminIn(tx, actor.ID, tx.NowFunc())
		if err != nil || current.Enabled == enabled {
			return err
		}

		err = tx.Model(&opponentRow{ID: id}).Select("enabled", "updated_at").
			Updates(opponentRow{Enabled: enabled, UpdatedAt: tx.NowFunc()}).Error
		if err != nil {
			return fmt.Errorf("changing whether %q is offered: %w", current.Name, err)
		}

		action := adminlog.ComputerOpponentRetired
		if enabled {
			action = adminlog.ComputerOpponentOffered
		}
		return adminlog.RecordIn(tx, admin, action, current.Name, "")
	})
	if err != nil {
		return Opponent{}, err
	}
	return s.ByID(ctx, id)
}

// ensurePlayable returns an *UnplayableError for an opponent whose deck breaks
// its rules.
func (s *Service) ensurePlayable(ctx context.Context, opponent Opponent) error {
	problems, err := s.Check(ctx, opponent)
	if err != nil {
		return err
	}
	if len(problems) > 0 {
		return &UnplayableError{Name: opponent.Name, Problems: problems}
	}
	return nil
}

// normalize trims an opponent's input and checks it, returning a
// *validation.Error for anything breaking the rules. It does not check that the
// deck is whole, as a retired opponent's need not be.
func (s *Service) normalize(ctx context.Context, input Input) (Input, error) {
	name, _, err := users.NormalizeUsername(input.Name)
	if err != nil {
		return Input{}, err
	}
	input.Name = name

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

	if _, known := computer.StyleNamed(input.Style); !known {
		return Input{}, validation.New("style", "must be one of the play styles")
	}
	if input.Strength != computer.Easy && input.Strength != computer.Normal {
		return Input{}, validation.New("strength", "must be easy or normal")
	}
	if _, err := s.formats.ByID(ctx, input.FormatID); err != nil {
		if errors.Is(err, formats.ErrNotFound) {
			return Input{}, validation.New("formatId", "must be one of the formats")
		}
		return Input{}, err
	}

	if input.Rules.MaxCivilizations < 0 || input.Rules.MaxCivilizations > MaxCivilizationsLimit {
		return Input{}, validation.New("maxCivilizations", fmt.Sprintf("must be from 0, for any number, to %d", MaxCivilizationsLimit))
	}
	if len(input.Rules.CardTypes) == 0 {
		return Input{}, validation.New("cardTypes", "must allow at least one card type")
	}
	allowed := make([]string, 0, len(CardTypes))
	for _, cardType := range CardTypes {
		if slices.Contains(input.Rules.CardTypes, cardType) {
			allowed = append(allowed, cardType)
		}
	}
	if len(allowed) != len(slices.Compact(slices.Sorted(slices.Values(input.Rules.CardTypes)))) {
		return Input{}, validation.New("cardTypes", "must be card types the game knows")
	}
	input.Rules.CardTypes = allowed

	if err := decks.CheckForSaving(ctx, s.catalogue, input.Cards); err != nil {
		return Input{}, err
	}
	return input, nil
}

// insert adds an opponent's row and deck.
func insert(tx *gorm.DB, row opponentRow, cards []decks.Entry) error {
	if err := tx.Create(&row).Error; err != nil {
		return fmt.Errorf("adding a computer opponent: %w", err)
	}
	return insertCards(tx, row.ID, cards)
}

// replaceCards replaces an opponent's deck.
func replaceCards(tx *gorm.DB, opponentID, name string, cards []decks.Entry) error {
	if err := tx.Where("computer_opponent_id = ?", opponentID).Delete(&opponentCardRow{}).Error; err != nil {
		return fmt.Errorf("replacing the deck of computer opponent %q: %w", name, err)
	}
	return insertCards(tx, opponentID, cards)
}

// insertCards adds an opponent's deck, which may be empty.
func insertCards(tx *gorm.DB, opponentID string, cards []decks.Entry) error {
	if len(cards) == 0 {
		return nil
	}

	entries := make([]opponentCardRow, 0, len(cards))
	for _, entry := range cards {
		entries = append(entries, opponentCardRow{ComputerOpponentID: opponentID, CardID: entry.CardID, Count: entry.Count})
	}
	if err := tx.Create(&entries).Error; err != nil {
		return fmt.Errorf("adding a computer opponent's deck: %w", err)
	}
	return nil
}
