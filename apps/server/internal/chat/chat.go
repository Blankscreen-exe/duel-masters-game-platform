// Package chat keeps the lobby chat: one chat for the whole site, which
// everyone can read and players can write in. Messages are kept in the
// database, so the chat survives restarts and admins can moderate it.
package chat

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/ratelimit"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/users"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/validation"
)

// Chat limits.
const (
	// MaxLength is the longest message, in characters.
	MaxLength = 300
	// HistoryLength is how many of the latest messages are shown.
	HistoryLength = 100
	// postsAllowed messages may be posted per postWindow, per player.
	postsAllowed = 5
	postWindow   = 10 * time.Second
)

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
	posts    *ratelimit.Limiter
}

// NewService returns the chat, telling listener about every new message.
// now is the clock; pass time.Now outside tests.
func NewService(db *gorm.DB, listener Listener, now func() time.Time) *Service {
	return &Service{db: db, listener: listener, posts: ratelimit.New(postsAllowed, postWindow, now)}
}

// Post adds a message from a player. Line breaks and tabs become spaces, and
// surrounding spaces are trimmed. It returns a *validation.Error for text
// breaking the rules and a *TooFastError when the player posts too fast.
func (s *Service) Post(ctx context.Context, author users.User, text string) (Message, error) {
	text, err := normalize(text)
	if err != nil {
		return Message{}, err
	}

	if allowed, retryAfter := s.posts.Take(author.ID); !allowed {
		return Message{}, &TooFastError{RetryAfter: retryAfter}
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
