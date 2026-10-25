package chat_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/chat"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/clocktest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/store/storetest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/users"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/validation"
)

// heard keeps the messages a listener is told about.
type heard struct {
	mu       sync.Mutex
	messages []chat.Message
}

func (h *heard) MessagePosted(message chat.Message) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.messages = append(h.messages, message)
}

type fixture struct {
	chat   *chat.Service
	heard  *heard
	clock  *clocktest.Clock
	shobu  users.User
	kokujo users.User
}

func newFixture(t *testing.T, db *gorm.DB) fixture {
	t.Helper()

	ctx := context.Background()
	accounts := users.NewService(db, storetest.Logger())
	shobu, err := accounts.Create(ctx, "Shobu", "hash")
	require.NoError(t, err)
	kokujo, err := accounts.Create(ctx, "Kokujo", "hash")
	require.NoError(t, err)

	listener := &heard{}
	c := clocktest.New(time.Now())

	return fixture{chat: chat.NewService(db, listener, c.Now), heard: listener, clock: c, shobu: shobu, kokujo: kokujo}
}

func texts(messages []chat.Message) []string {
	out := make([]string, 0, len(messages))
	for _, message := range messages {
		out = append(out, message.Username+": "+message.Text)
	}
	return out
}

func TestPostingAMessage(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		f := newFixture(t, db)

		posted, err := f.chat.Post(ctx, f.shobu, "  Anyone up\nfor a duel?  ")
		require.NoError(t, err)

		assert.NotEmpty(t, posted.ID)
		assert.Equal(t, f.shobu.ID, posted.UserID)
		assert.Equal(t, "Shobu", posted.Username)
		assert.Equal(t, "Anyone up for a duel?", posted.Text, "one line, trimmed")
		assert.False(t, posted.PostedAt.IsZero())
		assert.Equal(t, []chat.Message{posted}, f.heard.messages, "everyone watching hears it")

		recent, err := f.chat.Recent(ctx)
		require.NoError(t, err)
		require.Len(t, recent, 1)
		assert.Equal(t, posted.ID, recent[0].ID)
		assert.Equal(t, "Shobu", recent[0].Username)
		assert.True(t, posted.PostedAt.Equal(recent[0].PostedAt))
	})
}

func TestRecentShowsTheLatestMessagesOldestFirst(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		f := newFixture(t, db)

		for i := range chat.HistoryLength + 5 {
			author := f.shobu
			if i%2 == 1 {
				author = f.kokujo
			}
			_, err := f.chat.Post(ctx, author, fmt.Sprintf("message %d", i))
			require.NoError(t, err)
			// Past the posting limit, as players taking turns would be.
			f.clock.Advance(5 * time.Second)
		}

		recent, err := f.chat.Recent(ctx)
		require.NoError(t, err)
		require.Len(t, recent, chat.HistoryLength)
		assert.Equal(t, "message 5", recent[0].Text, "the oldest shown")
		assert.Equal(t, fmt.Sprintf("message %d", chat.HistoryLength+4), recent[len(recent)-1].Text, "the newest last")
		assert.Equal(t, "Kokujo", recent[0].Username)
	})
}

func TestADeletedMessageIsNotShown(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		f := newFixture(t, db)

		kept, err := f.chat.Post(ctx, f.shobu, "hello")
		require.NoError(t, err)
		removed, err := f.chat.Post(ctx, f.kokujo, "spam")
		require.NoError(t, err)
		require.NoError(t, db.Table("lobby_chat_messages").Where("id = ?", removed.ID).Update("deleted_at", time.Now()).Error)

		recent, err := f.chat.Recent(ctx)
		require.NoError(t, err)
		assert.Equal(t, []string{"Shobu: hello"}, texts(recent))
		assert.Equal(t, kept.ID, recent[0].ID)
	})
}

func TestAMessageMustBeOneLineOfText(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		f := newFixture(t, db)

		for name, test := range map[string]struct {
			text   string
			reason string
		}{
			"empty":          {"", "must not be empty"},
			"only spaces":    {" \n\t ", "must not be empty"},
			"too long":       {strings.Repeat("é", chat.MaxLength+1), "at most 300"},
			"not text":       {"hi \xff", "valid text"},
			"a control code": {"hi\x07there", "control characters"},
		} {
			_, err := f.chat.Post(ctx, f.shobu, test.text)
			invalid, ok := validation.As(err)
			require.True(t, ok, "%s: got %v", name, err)
			assert.Equal(t, "text", invalid.Field, name)
			assert.Contains(t, invalid.Reason, test.reason, name)
		}

		_, err := f.chat.Post(ctx, f.shobu, strings.Repeat("é", chat.MaxLength))
		assert.NoError(t, err, "exactly the longest message")

		recent, err := f.chat.Recent(ctx)
		require.NoError(t, err)
		assert.Len(t, recent, 1, "nothing invalid was saved")
		assert.Len(t, f.heard.messages, 1, "nor passed on")
	})
}

func TestPostingTooFastIsRefused(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		f := newFixture(t, db)

		for i := range 5 {
			_, err := f.chat.Post(ctx, f.shobu, fmt.Sprintf("spam %d", i))
			require.NoError(t, err)
		}

		f.clock.Advance(4 * time.Second)
		_, err := f.chat.Post(ctx, f.shobu, "one more")
		var tooFast *chat.TooFastError
		require.True(t, errors.As(err, &tooFast), "got %v", err)
		assert.Equal(t, 6*time.Second, tooFast.RetryAfter)

		_, err = f.chat.Post(ctx, f.kokujo, "my turn")
		assert.NoError(t, err, "the limit is per player")

		f.clock.Advance(6 * time.Second)
		_, err = f.chat.Post(ctx, f.shobu, "calm now")
		assert.NoError(t, err, "allowed again once the window has passed")
	})
}
