package opponents

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/computer"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/decks"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/formats"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/users"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/validation"
)

// LiveStyle is the stored style of the live opponent, who plays from outside
// the server. It is not one of the computer player's styles, so an admin cannot
// give it to another opponent.
const LiveStyle = "live"

const (
	liveName        = "Claude"
	liveDescription = "Playing live from Claude Code on this computer, with a deck it built itself. It takes its time."
)

// Live returns the live opponent, whether or not it has a deck yet. It returns
// ErrNotFound on a site without one, as when another account had its name.
func (s *Service) Live(ctx context.Context) (Opponent, error) {
	found, err := s.read(ctx, s.db.WithContext(ctx).Where("style = ?", LiveStyle))
	if err != nil {
		return Opponent{}, err
	}
	if len(found) == 0 {
		return Opponent{}, ErrNotFound
	}
	return found[0], nil
}

// SaveLiveDeck gives the live opponent a deck in a format, which players must
// then bring a deck legal in too. It returns what is wrong with a deck that
// cannot be played, which is then not saved, a *validation.Error for input
// breaking the rules, and ErrNotFound as Live does. Games already running carry
// on as they started.
func (s *Service) SaveLiveDeck(ctx context.Context, formatID string, cards []decks.Entry) ([]decks.Problem, error) {
	live, err := s.Live(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := s.formats.ByID(ctx, formatID); errors.Is(err, formats.ErrNotFound) {
		return nil, validation.New("formatId", "must be one of the formats")
	} else if err != nil {
		return nil, err
	}
	if err := decks.CheckForSaving(ctx, s.catalogue, cards); err != nil {
		return nil, err
	}

	live.FormatID, live.Cards = formatID, slices.Clone(cards)
	problems, err := s.Check(ctx, live)
	if err != nil || len(problems) > 0 {
		return problems, err
	}

	return nil, s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Model(&opponentRow{ID: live.ID}).Select("format_id", "enabled", "updated_at").
			Updates(opponentRow{FormatID: formatID, Enabled: true, UpdatedAt: tx.NowFunc()}).Error
		if err != nil {
			return fmt.Errorf("changing the deck of %q: %w", live.Name, err)
		}
		return replaceCards(tx, live.ID, live.Name, cards)
	})
}

// addLiveOpponent adds the live opponent with an account of its own, last in
// order and with no deck: it is offered once it has built one and a setting has
// switched it on.
func addLiveOpponent(tx *gorm.DB, logger *slog.Logger) error {
	var formatID string
	if err := tx.Table("formats").Select("id").Where("name = ?", startingFormat).Scan(&formatID).Error; err != nil {
		return fmt.Errorf("finding the %s format: %w", startingFormat, err)
	}
	if formatID == "" {
		logger.Warn("leaving out the live opponent, as the format it starts in is gone", "format", startingFormat)
		return nil
	}

	taken, err := users.TakenIn(tx, liveName)
	if err != nil {
		return err
	}
	if taken {
		logger.Warn("leaving out the live opponent, as an account already has its name", "name", liveName)
		return nil
	}
	account, err := users.CreateComputerIn(tx, liveName)
	if err != nil {
		return err
	}

	var last int
	if err := tx.Table("computer_opponents").Select("COALESCE(MAX(sort_order), 0)").Scan(&last).Error; err != nil {
		return fmt.Errorf("finding the last computer opponent: %w", err)
	}
	id, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("making a computer opponent id: %w", err)
	}
	now := tx.NowFunc()
	return insert(tx, opponentRow{
		ID:          id.String(),
		UserID:      account.ID,
		Description: liveDescription,
		Style:       LiveStyle,
		Strength:    string(computer.Normal),
		FormatID:    formatID,
		CardTypes:   BaseCardTypes,
		SortOrder:   last + 1,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil)
}
