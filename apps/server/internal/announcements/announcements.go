// Package announcements keeps what admins tell every player. Everyone can read
// them; an important one also pops up in the lobby until each player closes it.
// The site calls them notifications; here that word would be confused with the
// short messages a page shows, such as "Deck saved".
package announcements

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/adminlog"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/paging"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/users"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/validation"
)

// Limits.
const (
	TitleMaxLength = 100
	BodyMaxLength  = 2000

	DefaultPageSize = 20
	MaxPageSize     = 100

	// maxPopups bounds the popups one player is given at once. Those beyond it
	// come once the first are closed.
	maxPopups = 20
)

// ErrNotFound is returned when no announcement has the id.
var ErrNotFound = errors.New("announcement not found")

// Announcement is one thing an admin has told every player.
type Announcement struct {
	ID        string `gorm:"primaryKey"`
	Title     string
	Body      string
	Important bool
	// AnnouncedAt is when it last went out to players: when it was posted, made
	// important, or an admin asked to show it again.
	AnnouncedAt time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (Announcement) TableName() string { return "announcements" }

// dismissal is a player having closed an announcement's popup.
type dismissal struct {
	AnnouncementID string `gorm:"primaryKey"`
	UserID         string `gorm:"primaryKey"`
	DismissedAt    time.Time
}

func (dismissal) TableName() string { return "announcement_dismissals" }

// Input is an announcement as an admin writes it.
type Input struct {
	Title     string
	Body      string
	Important bool
	// ShowAgain, on changing an important announcement, pops it up again for
	// players who have closed it. Without it, a correction stays quiet.
	ShowAgain bool
}

// Page is one page of the announcements, the latest first.
type Page struct {
	Announcements []Announcement
	Total         int
	Page          int
	PageSize      int
}

// ForPlayer is what a player has yet to see.
type ForPlayer struct {
	// Unread counts what has gone out since they last looked.
	Unread int
	// Popups are the important announcements they have yet to close, the oldest
	// first.
	Popups []Announcement
}

// Listener is told when players should look again.
type Listener interface {
	AnnouncementsChanged()
}

// Service keeps the announcements. It is safe for concurrent use.
type Service struct {
	db       *gorm.DB
	accounts *users.Service
	listener Listener
	now      func() time.Time
	logger   *slog.Logger
}

// NewService returns the service. now is the clock; pass time.Now outside tests.
func NewService(db *gorm.DB, accounts *users.Service, listener Listener, now func() time.Time, logger *slog.Logger) *Service {
	return &Service{db: db, accounts: accounts, listener: listener, now: now, logger: logger}
}

// timestamp is the current time as the database stores it: UTC, to the
// microsecond.
func (s *Service) timestamp() time.Time {
	return s.now().UTC().Truncate(time.Microsecond)
}

// List returns one page of the announcements, the latest first. It returns a
// *validation.Error for a page breaking the limits.
func (s *Service) List(ctx context.Context, request paging.Request) (Page, error) {
	if err := request.Normalize(DefaultPageSize, MaxPageSize); err != nil {
		return Page{}, err
	}

	var total int64
	if err := s.db.WithContext(ctx).Model(&Announcement{}).Count(&total).Error; err != nil {
		return Page{}, fmt.Errorf("counting announcements: %w", err)
	}

	var found []Announcement
	err := s.db.WithContext(ctx).Order("announced_at DESC, id DESC").
		Limit(request.PageSize).Offset(request.Offset()).Find(&found).Error
	if err != nil {
		return Page{}, fmt.Errorf("listing announcements: %w", err)
	}

	return Page{Announcements: found, Total: int(total), Page: request.Page, PageSize: request.PageSize}, nil
}

// For returns what a player has yet to see. Nothing from before they joined
// counts: a new player has no pile of old news to get through, though they can
// read it all.
func (s *Service) For(ctx context.Context, player users.User) (ForPlayer, error) {
	since := player.CreatedAt
	if player.AnnouncementsSeenAt != nil && player.AnnouncementsSeenAt.After(since) {
		since = *player.AnnouncementsSeenAt
	}

	var unread int64
	err := s.db.WithContext(ctx).Model(&Announcement{}).Where("announced_at > ?", since).Count(&unread).Error
	if err != nil {
		return ForPlayer{}, fmt.Errorf("counting unread announcements: %w", err)
	}

	dismissed := s.db.WithContext(ctx).Model(&dismissal{}).Select("announcement_id").Where("user_id = ?", player.ID)
	popups := []Announcement{}
	err = s.db.WithContext(ctx).
		Where("important = ? AND announced_at > ?", true, player.CreatedAt).
		Where("id NOT IN (?)", dismissed).
		Order("announced_at, id").Limit(maxPopups).Find(&popups).Error
	if err != nil {
		return ForPlayer{}, fmt.Errorf("finding a player's popups: %w", err)
	}

	return ForPlayer{Unread: int(unread), Popups: popups}, nil
}

// MarkSeen notes that a player has looked at the announcements just now.
func (s *Service) MarkSeen(ctx context.Context, player users.User) error {
	return s.accounts.SetAnnouncementsSeen(ctx, player.ID, s.timestamp())
}

// Dismiss notes that a player has closed an announcement's popup, so it does not
// come back. Closing it twice is not an error. It returns ErrNotFound when there
// is no such announcement.
func (s *Service) Dismiss(ctx context.Context, player users.User, id string) error {
	if err := s.db.WithContext(ctx).Where("id = ?", id).Take(&Announcement{}).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return fmt.Errorf("finding an announcement: %w", err)
	}

	row := dismissal{AnnouncementID: id, UserID: player.ID, DismissedAt: s.timestamp()}
	if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error; err != nil {
		return fmt.Errorf("noting a closed popup: %w", err)
	}

	return nil
}

// normalize checks an admin's text and returns it tidied.
func normalize(input Input) (Input, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Body = strings.TrimSpace(strings.ReplaceAll(input.Body, "\r\n", "\n"))

	switch {
	case input.Title == "":
		return Input{}, validation.New("title", "is required")
	case utf8.RuneCountInString(input.Title) > TitleMaxLength:
		return Input{}, validation.New("title", fmt.Sprintf("must be at most %d characters", TitleMaxLength))
	case strings.ContainsAny(input.Title, "\r\n"):
		return Input{}, validation.New("title", "must be one line")
	case input.Body == "":
		return Input{}, validation.New("body", "is required")
	case utf8.RuneCountInString(input.Body) > BodyMaxLength:
		return Input{}, validation.New("body", fmt.Sprintf("must be at most %d characters", BodyMaxLength))
	}

	return input, nil
}

// importance is how the admin log describes an announcement.
func importance(important bool) string {
	if important {
		return "Important"
	}
	return ""
}

// Create posts an announcement. It returns a *validation.Error for text breaking
// the rules and users.ErrNotAdmin for an account no longer an admin.
func (s *Service) Create(ctx context.Context, actor users.User, input Input) (Announcement, error) {
	input, err := normalize(input)
	if err != nil {
		return Announcement{}, err
	}
	id, err := uuid.NewV7()
	if err != nil {
		return Announcement{}, fmt.Errorf("making an announcement id: %w", err)
	}

	now := s.timestamp()
	row := Announcement{ID: id.String(), Title: input.Title, Body: input.Body, Important: input.Important, AnnouncedAt: now, CreatedAt: now, UpdatedAt: now}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		admin, err := users.LockAdminIn(tx, actor.ID, now)
		if err != nil {
			return err
		}
		if err := tx.Create(&row).Error; err != nil {
			return fmt.Errorf("adding an announcement: %w", err)
		}
		return adminlog.RecordIn(tx, admin, adminlog.AnnouncementAdded, row.Title, importance(row.Important))
	})
	if err != nil {
		return Announcement{}, err
	}

	s.listener.AnnouncementsChanged()
	return row, nil
}

// Update changes an announcement. Making one important sends it out, as does
// asking to show an important one again; those pop it up even for players who
// closed it before. Anything else, such as fixing a typo, stays quiet. It returns
// a *validation.Error for text breaking the rules, ErrNotFound when there is no
// such announcement, and users.ErrNotAdmin for an account no longer an admin.
func (s *Service) Update(ctx context.Context, actor users.User, id string, input Input) (Announcement, error) {
	input, err := normalize(input)
	if err != nil {
		return Announcement{}, err
	}

	now := s.timestamp()
	var row Announcement
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		admin, err := users.LockAdminIn(tx, actor.ID, now)
		if err != nil {
			return err
		}
		if err := tx.Where("id = ?", id).Take(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return fmt.Errorf("finding an announcement: %w", err)
		}

		sendAgain := input.Important && (input.ShowAgain || !row.Important)
		row.Title, row.Body, row.Important, row.UpdatedAt = input.Title, input.Body, input.Important, now
		details := importance(row.Important)
		if sendAgain {
			row.AnnouncedAt = now
			details = "Important, shown to everyone again"
			if err := tx.Where("announcement_id = ?", id).Delete(&dismissal{}).Error; err != nil {
				return fmt.Errorf("forgetting closed popups: %w", err)
			}
		}

		if err := tx.Save(&row).Error; err != nil {
			return fmt.Errorf("changing an announcement: %w", err)
		}
		return adminlog.RecordIn(tx, admin, adminlog.AnnouncementChanged, row.Title, details)
	})
	if err != nil {
		return Announcement{}, err
	}

	s.listener.AnnouncementsChanged()
	return row, nil
}

// Delete removes an announcement for good; the admin log keeps its title. It
// returns ErrNotFound when there is no such announcement, and users.ErrNotAdmin
// for an account no longer an admin.
func (s *Service) Delete(ctx context.Context, actor users.User, id string) error {
	now := s.timestamp()
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		admin, err := users.LockAdminIn(tx, actor.ID, now)
		if err != nil {
			return err
		}

		var row Announcement
		if err := tx.Where("id = ?", id).Take(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return fmt.Errorf("finding an announcement: %w", err)
		}
		if err := tx.Delete(&row).Error; err != nil {
			return fmt.Errorf("deleting an announcement: %w", err)
		}
		return adminlog.RecordIn(tx, admin, adminlog.AnnouncementDeleted, row.Title, "")
	})
	if err != nil {
		return err
	}

	s.listener.AnnouncementsChanged()
	return nil
}
