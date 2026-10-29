// Package chat keeps the lobby chat: one chat for the whole site, which
// everyone can read and players can write in. Messages are kept in the
// database, so the chat survives restarts and admins can moderate it.
package chat

import (
	"context"
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/ratelimit"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/settings"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/users"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/validation"
)

// Chat limits.
const (
	// MaxLength is the longest message, in characters.
	MaxLength = 300
	// HistoryLength is how many of the latest messages are shown.
	HistoryLength = 100
)

// PostsAllowed and PostWindow are the admin settings for how fast a player
// may chat: at most PostsAllowed messages in each PostWindow. They apply to
// the lobby chat and, each counted on its own, to the chat in games.
var (
	PostsAllowed = settings.Number{
		Key:         "chat.posts_allowed",
		Group:       "Chat",
		Description: "How many messages a player may post in the time below, in the lobby chat and, counted separately, in the chat of their games. Past it they are told to wait.",
		Default:     5,
		Min:         1,
		Max:         100,
	}
	PostWindow = settings.Number{
		Key:         "chat.post_window_seconds",
		Group:       "Chat",
		Description: "The time, in seconds, the number of messages above is counted over.",
		Default:     10,
		Min:         1,
		Max:         600,
	}
)

// Settings are the settings this feature declares.
func Settings() []settings.Definition {
	return []settings.Definition{PostsAllowed, PostWindow}
}

// PostLimits returns how fast a player may chat, as the settings say at the
// moment it is called, for a ratelimit.NewLive limiter.
func PostLimits(siteSettings *settings.Service) func() ratelimit.Limits {
	return func() ratelimit.Limits {
		return ratelimit.Limits{
			Attempts: int(siteSettings.Number(PostsAllowed)),
			Window:   time.Duration(siteSettings.Number(PostWindow)) * time.Second,
		}
	}
}

// Message is a message in the chat.
type Message struct {
	ID       string
	UserID   string
	Username string
	Text     string
	PostedAt time.Time
}

// TooFastError is returned to a player posting faster than allowed.
type TooFastError struct {
	RetryAfter time.Duration
}

func (e *TooFastError) Error() string {
	return fmt.Sprintf("posting too fast; wait %s", e.RetryAfter.Round(time.Second))
}

// Seconds is how many whole seconds the player must wait, rounded up.
func (e *TooFastError) Seconds() int {
	return int(math.Ceil(e.RetryAfter.Seconds()))
}

// Refusal tells the player, in a sentence, to wait before posting again.
func (e *TooFastError) Refusal() string {
	seconds := e.Seconds()
	unit := "seconds"
	if seconds == 1 {
		unit = "second"
	}
	return fmt.Sprintf("You are posting too fast. Wait %d %s.", seconds, unit)
}

// MutedError is returned to a player an admin has muted.
type MutedError struct {
	Mute users.Penalty
}

func (e *MutedError) Error() string {
	if e.Mute.Until == nil {
		return "muted"
	}
	return fmt.Sprintf("muted until %s", e.Mute.Until.Format(time.RFC3339))
}

// Refusal tells the player, in a sentence, that they are muted and until
// when.
func (e *MutedError) Refusal() string {
	if e.Mute.Until == nil {
		return "An admin has muted you in the chat."
	}
	return "An admin has muted you in the chat until " + e.Mute.Until.UTC().Format(time.RFC1123) + "."
}

// Refusal is what to tell a player whose post Gate.Check refused, or an
// empty string for an error it did not return.
func Refusal(err error) string {
	var muted *MutedError
	if errors.As(err, &muted) {
		return muted.Refusal()
	}
	var tooFast *TooFastError
	if errors.As(err, &tooFast) {
		return tooFast.Refusal()
	}
	return ""
}

// Gate decides whether a player may post in a chat now: not muted by an
// admin, and no faster than the settings allow. Each chat keeps its own
// gate, so each counts its posts on its own. It is safe for concurrent use.
type Gate struct {
	posts *ratelimit.Limiter
	now   func() time.Time
}

// NewGate returns a gate following the settings as they change. now is the
// clock; pass time.Now outside tests.
func NewGate(siteSettings *settings.Service, now func() time.Time) *Gate {
	return &Gate{posts: ratelimit.NewLive(PostLimits(siteSettings), now), now: now}
}

// Check returns a *MutedError for a muted author and a *TooFastError for one
// posting too fast. A post it allows counts towards the limit; a refused one
// does not.
func (g *Gate) Check(author users.User) error {
	if mute, muted := author.MuteAt(g.now()); muted {
		return &MutedError{Mute: mute}
	}
	if allowed, retryAfter := g.posts.Take(author.ID); !allowed {
		return &TooFastError{RetryAfter: retryAfter}
	}
	return nil
}

// Listener hears about new messages, to pass them on to everyone watching
// the chat. It is called after the message is saved and must not block.
type Listener interface {
	MessagePosted(Message)
}

// messageRow is a row of lobby_chat_messages.
type messageRow struct {
	ID        string `gorm:"primaryKey"`
	UserID    string
	Text      string
	PostedAt  time.Time
	DeletedAt *time.Time
}

func (messageRow) TableName() string { return "lobby_chat_messages" }

// shownRow is a message as the chat shows it, with its author's name.
type shownRow struct {
	ID       string
	UserID   string
	Text     string
	PostedAt time.Time
	Username string
}

// Service reads and writes the chat. It is safe for concurrent use.
type Service struct {
	db       *gorm.DB
	listener Listener
	gate     *Gate
}

// NewService returns the chat, telling listener about every new message and
// letting each player post as fast as the settings allow. now is the clock;
// pass time.Now outside tests.
func NewService(db *gorm.DB, listener Listener, siteSettings *settings.Service, now func() time.Time) *Service {
	return &Service{db: db, listener: listener, gate: NewGate(siteSettings, now)}
}

// Post adds a message from a player. Line breaks and tabs become spaces, and
// surrounding spaces are trimmed. It returns a *validation.Error for text
// breaking the rules, a *MutedError for a muted player and a *TooFastError
// when the player posts too fast.
func (s *Service) Post(ctx context.Context, author users.User, text string) (Message, error) {
	text, err := normalize(text)
	if err != nil {
		return Message{}, err
	}

	if err := s.gate.Check(author); err != nil {
		return Message{}, err
	}

	id, err := uuid.NewV7()
	if err != nil {
		return Message{}, fmt.Errorf("making a chat message id: %w", err)
	}

	db := s.db.WithContext(ctx)
	row := messageRow{ID: id.String(), UserID: author.ID, Text: text, PostedAt: db.NowFunc()}
	if err := db.Create(&row).Error; err != nil {
		return Message{}, fmt.Errorf("saving a chat message: %w", err)
	}

	message := Message{ID: row.ID, UserID: author.ID, Username: author.Username, Text: text, PostedAt: row.PostedAt}
	s.listener.MessagePosted(message)

	return message, nil
}

// Recent returns the latest messages still shown, oldest first.
func (s *Service) Recent(ctx context.Context) ([]Message, error) {
	var rows []shownRow
	err := s.db.WithContext(ctx).
		Model(&messageRow{}).
		Select("lobby_chat_messages.id, lobby_chat_messages.user_id, lobby_chat_messages.text, lobby_chat_messages.posted_at, users.username").
		Joins("JOIN users ON users.id = lobby_chat_messages.user_id").
		Where("lobby_chat_messages.deleted_at IS NULL").
		Order("lobby_chat_messages.posted_at DESC, lobby_chat_messages.id DESC").
		Limit(HistoryLength).
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("reading the chat: %w", err)
	}

	messages := make([]Message, 0, len(rows))
	for _, row := range rows {
		messages = append(messages, Message{ID: row.ID, UserID: row.UserID, Username: row.Username, Text: row.Text, PostedAt: row.PostedAt})
	}
	slices.Reverse(messages)

	return messages, nil
}

// normalize makes a message one line of printable text and checks its length.
func normalize(text string) (string, error) {
	if !utf8.ValidString(text) {
		return "", validation.New("text", "must be valid text")
	}

	var line strings.Builder
	for _, r := range text {
		switch {
		case r == '\n' || r == '\r' || r == '\t':
			line.WriteRune(' ')
		case unicode.IsControl(r):
			return "", validation.New("text", "must not contain control characters")
		default:
			line.WriteRune(r)
		}
	}

	normalized := strings.TrimSpace(line.String())
	switch {
	case normalized == "":
		return "", validation.New("text", "must not be empty")
	case utf8.RuneCountInString(normalized) > MaxLength:
		return "", validation.New("text", fmt.Sprintf("must be at most %d characters", MaxLength))
	}

	return normalized, nil
}
