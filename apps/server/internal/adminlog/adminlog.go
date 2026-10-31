// Package adminlog keeps the admin log: what admins have done in the admin
// panel, to what, and when. An action is recorded in the same transaction as
// the change it describes, so the log never misses a change nor lists one
// that did not happen.
package adminlog

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/paging"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/users"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/validation"
)

// Admin log limits.
const (
	DefaultPageSize = 50
	MaxPageSize     = 100
)

// Action is what an admin did.
type Action string

// The actions the log records. They are part of the API: clients show and
// filter by them, so they never change.
const (
	UserBanned         Action = "user_banned"
	UserUnbanned       Action = "user_unbanned"
	UserMuted          Action = "user_muted"
	UserUnmuted        Action = "user_unmuted"
	RoleChanged        Action = "role_changed"
	PasswordReset      Action = "password_reset"
	PlayerLetIn        Action = "player_let_in"
	SettingChanged     Action = "setting_changed"
	SettingReset       Action = "setting_reset"
	ChatMessageDeleted Action = "chat_message_deleted"
	GameStopped        Action = "game_stopped"

	ComputerOpponentAdded   Action = "computer_opponent_added"
	ComputerOpponentChanged Action = "computer_opponent_changed"
	ComputerOpponentOffered Action = "computer_opponent_offered"
	ComputerOpponentRetired Action = "computer_opponent_retired"

	FormatAdded      Action = "format_added"
	FormatChanged    Action = "format_changed"
	FormatRetired    Action = "format_retired"
	FormatRestored   Action = "format_restored"
	FormatsReordered Action = "formats_reordered"

	MailSettingsChanged Action = "mail_settings_changed"
	MailSettingsCleared Action = "mail_settings_cleared"
)

// Actions lists every action, in the order the admin panel offers them.
func Actions() []Action {
	return []Action{
		UserBanned, UserUnbanned, UserMuted, UserUnmuted, RoleChanged, PasswordReset, PlayerLetIn,
		SettingChanged, SettingReset, ChatMessageDeleted, GameStopped,
		ComputerOpponentAdded, ComputerOpponentChanged, ComputerOpponentOffered, ComputerOpponentRetired,
		FormatAdded, FormatChanged, FormatRetired, FormatRestored, FormatsReordered,
		MailSettingsChanged, MailSettingsCleared,
	}
}

// Entry is one action in the log.
type Entry struct {
	ID string
	// AdminID is empty once the admin's account is gone; AdminName keeps
	// their name as it was.
	AdminID   string
	AdminName string
	Action    Action
	// Subject is what the action was done to, such as a player's name.
	Subject string
	// Details say how, such as a ban's length and reason; empty when there is
	// nothing to add.
	Details string
	At      time.Time
}

// row is a row of admin_actions.
type row struct {
	ID        string `gorm:"primaryKey"`
	AdminID   *string
	AdminName string
	Action    Action
	Subject   string
	Details   string
	CreatedAt time.Time
}

func (row) TableName() string { return "admin_actions" }

// RecordIn adds an action to the log inside the caller's transaction, the
// one making the change.
func RecordIn(tx *gorm.DB, admin users.User, action Action, subject, details string) error {
	id, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("making an admin log id: %w", err)
	}

	adminID := admin.ID
	entry := row{ID: id.String(), AdminID: &adminID, AdminName: admin.Username, Action: action, Subject: subject, Details: details, CreatedAt: tx.NowFunc()}
	if err := tx.Create(&entry).Error; err != nil {
		return fmt.Errorf("recording %s in the admin log: %w", action, err)
	}
	return nil
}

// Query asks for one page of the log.
type Query struct {
	// Action keeps the entries of one action; empty for every action.
	Action Action
	paging.Request
}

// Page is one page of the log, the latest first.
type Page struct {
	Entries  []Entry
	Total    int
	Page     int
	PageSize int
}

// Service reads the admin log. It is safe for concurrent use.
type Service struct {
	db *gorm.DB
}

// NewService returns a service using db.
func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// Record adds an action that changes nothing in the database, such as
// stopping a game, to the log.
func (s *Service) Record(ctx context.Context, admin users.User, action Action, subject, details string) error {
	return RecordIn(s.db.WithContext(ctx), admin, action, subject, details)
}

// List returns one page of the log, the latest first. It returns a
// *validation.Error for an unknown action or a page breaking the limits.
func (s *Service) List(ctx context.Context, query Query) (Page, error) {
	if query.Action != "" && !slices.Contains(Actions(), query.Action) {
		return Page{}, validation.New("action", "must be one of the admin log's actions")
	}
	if err := query.Normalize(DefaultPageSize, MaxPageSize); err != nil {
		return Page{}, err
	}

	matching := s.db.WithContext(ctx).Model(&row{})
	if query.Action != "" {
		matching = matching.Where("action = ?", query.Action)
	}

	var total int64
	if err := matching.Count(&total).Error; err != nil {
		return Page{}, fmt.Errorf("counting the admin log: %w", err)
	}

	// Ids are made in time order, so they order actions made at the same
	// moment.
	var rows []row
	err := matching.Order("created_at DESC").Order("id DESC").Limit(query.PageSize).Offset(query.Offset()).Find(&rows).Error
	if err != nil {
		return Page{}, fmt.Errorf("reading the admin log: %w", err)
	}

	entries := make([]Entry, 0, len(rows))
	for _, r := range rows {
		entry := Entry{ID: r.ID, AdminName: r.AdminName, Action: r.Action, Subject: r.Subject, Details: r.Details, At: r.CreatedAt}
		if r.AdminID != nil {
			entry.AdminID = *r.AdminID
		}
		entries = append(entries, entry)
	}
	return Page{Entries: entries, Total: int(total), Page: query.Page, PageSize: query.PageSize}, nil
}
