package games

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/chat"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/config"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/settings"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/store/storetest"
)

// queuedConnection hands out the messages queued on it, one per read, and
// ends reading once closed.
type queuedConnection struct {
	messages chan []byte
	closed   chan struct{}
}

func newQueuedConnection(count int) *queuedConnection {
	c := &queuedConnection{messages: make(chan []byte, count), closed: make(chan struct{})}
	for i := range count {
		c.messages <- fmt.Appendf(nil, "message %d", i)
	}
	return c
}

func (c *queuedConnection) SetReadLimit(int64)                {}
func (c *queuedConnection) SetReadDeadline(time.Time) error   { return nil }
func (c *queuedConnection) SetPongHandler(func(string) error) {}
func (c *queuedConnection) SetWriteDeadline(time.Time) error  { return nil }
func (c *queuedConnection) WriteMessage(int, []byte) error    { return nil }
func (c *queuedConnection) WriteJSON(any) error               { return nil }

func (c *queuedConnection) ReadMessage() (int, []byte, error) {
	select {
	case message := <-c.messages:
		return 1, message, nil
	case <-c.closed:
		return 0, nil, errors.New("closed")
	}
}

func (c *queuedConnection) Close() error {
	select {
	case <-c.closed:
	default:
		close(c.closed)
	}
	return nil
}

// paceSettings returns the settings, with the pace set as given.
func paceSettings(t *testing.T, perSecond, burst int) *settings.Service {
	t.Helper()

	ctx := context.Background()
	db := storetest.Open(t, config.SQLite)
	siteSettings, err := settings.NewService(ctx, db, storetest.Logger(), slices.Concat(Settings(), chat.Settings())...)
	require.NoError(t, err)
	setPace(t, siteSettings, perSecond, burst)
	return siteSettings
}

func setPace(t *testing.T, siteSettings *settings.Service, perSecond, burst int) {
	t.Helper()

	ctx := context.Background()
	_, err := siteSettings.Set(ctx, MessagesPerSecond.Key, fmt.Sprint(perSecond))
	require.NoError(t, err)
	_, err = siteSettings.Set(ctx, MessageBurst.Key, fmt.Sprint(burst))
	require.NoError(t, err)
}

// readAll reads count messages and returns them with how long it took.
func readAll(t *testing.T, conn *pacedConnection, count int) ([]string, time.Duration) {
	t.Helper()

	start := time.Now()
	read := make([]string, 0, count)
	for range count {
		_, data, err := conn.ReadMessage()
		require.NoError(t, err)
		read = append(read, string(data))
	}
	return read, time.Since(start)
}

func TestMessagesPastTheBurstWaitForThePace(t *testing.T) {
	conn := newPacedConnection(newQueuedConnection(9), paceSettings(t, 20, 3))
	defer conn.Close()

	read, took := readAll(t, conn, 9)

	want := make([]string, 0, 9)
	for i := range 9 {
		want = append(want, fmt.Sprintf("message %d", i))
	}
	assert.Equal(t, want, read, "every message arrives, in order")
	// After a burst of 3, the other 6 come at 20 a second: 300ms at least.
	assert.GreaterOrEqual(t, took, 250*time.Millisecond, "the messages past the burst waited")
}

func TestABurstPassesAtOnce(t *testing.T) {
	conn := newPacedConnection(newQueuedConnection(5), paceSettings(t, 1, 5))
	defer conn.Close()

	_, took := readAll(t, conn, 5)
	assert.Less(t, took, 500*time.Millisecond, "a burst within the limit does not wait")
}

func TestClosingEndsAWaitForThePace(t *testing.T) {
	conn := newPacedConnection(newQueuedConnection(2), paceSettings(t, 1, 1))
	readAll(t, conn, 1)

	returned := make(chan error, 1)
	go func() {
		_, _, err := conn.ReadMessage()
		returned <- err
	}()

	time.Sleep(50 * time.Millisecond)
	start := time.Now()
	require.NoError(t, conn.Close())

	select {
	case err := <-returned:
		assert.Error(t, err, "a read cut short by closing reports it")
		assert.Less(t, time.Since(start), 500*time.Millisecond)
	case <-time.After(2 * time.Second):
		t.Fatal("closing did not end the wait")
	}
}

func TestAChangedPaceAppliesToTheNextMessage(t *testing.T) {
	siteSettings := paceSettings(t, 1, 1)
	conn := newPacedConnection(newQueuedConnection(11), siteSettings)
	defer conn.Close()
	readAll(t, conn, 1)

	// At the old pace the next ten would take ten seconds.
	setPace(t, siteSettings, 100, 200)
	_, took := readAll(t, conn, 10)
	assert.Less(t, took, 2*time.Second, "the new pace applied")
}
