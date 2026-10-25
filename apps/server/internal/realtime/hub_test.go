package realtime

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testOrigin  = "http://site.test"
	testTimeout = 5 * time.Second
)

// presenceLog keeps the players whose presence changed, as the hub reports
// them.
type presenceLog struct {
	mu      sync.Mutex
	changed []string
}

func (p *presenceLog) PresenceChanged(viewer Viewer) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.changed = append(p.changed, viewer.Username)
}

func (p *presenceLog) names() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]string(nil), p.changed...)
}

// testServer serves the hub over HTTP, taking the viewer from the request's
// "user" and "name" query parameters.
func testServer(t *testing.T, hub *Hub) *httptest.Server {
	t.Helper()

	upgrader := NewUpgrader([]string{testOrigin})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		_ = hub.Serve(conn, Viewer{UserID: r.URL.Query().Get("user"), Username: r.URL.Query().Get("name")})
	}))
	t.Cleanup(server.Close)

	return server
}

func newHub(t *testing.T, offlineAfter time.Duration) (*Hub, *presenceLog) {
	t.Helper()

	presence := &presenceLog{}
	hub := NewHub(Options{OfflineAfter: offlineAfter, Presence: presence, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	t.Cleanup(func() { _ = hub.Close(context.Background()) })

	return hub, presence
}

// dial opens a connection as a viewer; an empty user is a visitor.
func dial(t *testing.T, server *httptest.Server, user, name string) *websocket.Conn {
	t.Helper()

	url := "ws" + strings.TrimPrefix(server.URL, "http") + "/?user=" + user + "&name=" + name
	conn, response, err := websocket.DefaultDialer.Dial(url, http.Header{"Origin": {testOrigin}})
	require.NoError(t, err)
	_ = response.Body.Close()
	t.Cleanup(func() { _ = conn.Close() })

	return conn
}

// receive reads the next message, failing the test if none comes.
func receive(t *testing.T, conn *websocket.Conn) map[string]any {
	t.Helper()

	require.NoError(t, conn.SetReadDeadline(time.Now().Add(testTimeout)))
	_, data, err := conn.ReadMessage()
	require.NoError(t, err)

	var message map[string]any
	require.NoError(t, json.Unmarshal(data, &message))
	return message
}

// nothingArrives checks that no message arrives for a while.
func nothingArrives(t *testing.T, conn *websocket.Conn) {
	t.Helper()

	require.NoError(t, conn.SetReadDeadline(time.Now().Add(150*time.Millisecond)))
	_, data, err := conn.ReadMessage()
	require.Error(t, err, "expected nothing, got %s", data)
}

func waitUntil(t *testing.T, condition func() bool, message string) {
	t.Helper()
	require.Eventually(t, condition, testTimeout, 5*time.Millisecond, message)
}

func TestBroadcastReachesEveryConnection(t *testing.T) {
	hub, _ := newHub(t, time.Minute)
	server := testServer(t, hub)

	visitor := dial(t, server, "", "")
	shobu := dial(t, server, "u1", "Shobu")
	waitUntil(t, func() bool { return hub.IsOnline("u1") }, "Shobu should be online")
	waitUntil(t, func() bool { return hub.Connections() == 2 }, "both connections should be open")

	hub.Broadcast(map[string]any{"type": "hello"})

	assert.Equal(t, "hello", receive(t, visitor)["type"])
	assert.Equal(t, "hello", receive(t, shobu)["type"])
}

func TestSendToReachesOnlyThatPlayersTabs(t *testing.T) {
	hub, _ := newHub(t, time.Minute)
	server := testServer(t, hub)

	firstTab := dial(t, server, "u1", "Shobu")
	secondTab := dial(t, server, "u1", "Shobu")
	other := dial(t, server, "u2", "Kokujo")
	visitor := dial(t, server, "", "")
	waitUntil(t, func() bool { return hub.Connections() == 4 }, "every connection should be open")

	hub.SendTo("u1", map[string]any{"type": "your_game_started"})

	assert.Equal(t, "your_game_started", receive(t, firstTab)["type"])
	assert.Equal(t, "your_game_started", receive(t, secondTab)["type"])
	nothingArrives(t, other)
	nothingArrives(t, visitor)
}

func TestAPlayerIsOnlineUntilTheirLastTabHasBeenClosedForAWhile(t *testing.T) {
	hub, presence := newHub(t, 150*time.Millisecond)
	server := testServer(t, hub)

	firstTab := dial(t, server, "u1", "Shobu")
	waitUntil(t, func() bool { return hub.IsOnline("u1") }, "Shobu should be online")
	secondTab := dial(t, server, "u1", "Shobu")
	waitUntil(t, func() bool { return hub.Connections() == 2 }, "both tabs should be open")
	assert.Equal(t, []string{"Shobu"}, presence.names(), "one notice for the first tab only")
	assert.Equal(t, []Viewer{{UserID: "u1", Username: "Shobu"}}, hub.Online())

	require.NoError(t, firstTab.Close())
	waitUntil(t, func() bool { return hub.Connections() == 1 }, "the first tab should be gone")
	assert.Never(t, func() bool { return !hub.IsOnline("u1") }, 300*time.Millisecond, 10*time.Millisecond,
		"still online in the second tab")

	require.NoError(t, secondTab.Close())
	waitUntil(t, func() bool { return hub.Connections() == 0 }, "the second tab should be gone")
	assert.True(t, hub.IsOnline("u1"), "online for a moment after the last tab closes")

	waitUntil(t, func() bool { return !hub.IsOnline("u1") }, "Shobu should go offline")
	assert.Equal(t, []string{"Shobu", "Shobu"}, presence.names())
	assert.Empty(t, hub.Online())
}

func TestReloadingAPageDoesNotCountAsLeaving(t *testing.T) {
	hub, presence := newHub(t, 300*time.Millisecond)
	server := testServer(t, hub)

	tab := dial(t, server, "u1", "Shobu")
	waitUntil(t, func() bool { return hub.IsOnline("u1") }, "Shobu should be online")
	require.NoError(t, tab.Close())
	waitUntil(t, func() bool { return hub.Connections() == 0 }, "the tab should be gone")

	dial(t, server, "u1", "Shobu")
	assert.Never(t, func() bool { return !hub.IsOnline("u1") }, 500*time.Millisecond, 10*time.Millisecond)
	assert.Equal(t, []string{"Shobu"}, presence.names(), "no notice for the reload")
}

func TestVisitorsAreNotPlayers(t *testing.T) {
	hub, presence := newHub(t, time.Minute)
	server := testServer(t, hub)

	dial(t, server, "", "")
	waitUntil(t, func() bool { return hub.Connections() == 1 }, "the visitor should be connected")

	assert.Empty(t, hub.Online())
	assert.Empty(t, presence.names())
}

func TestASlowConnectionIsDroppedRatherThanHoldingUpOthers(t *testing.T) {
	hub, _ := newHub(t, time.Minute)
	server := testServer(t, hub)

	stuck := dial(t, server, "u1", "Stuck")
	healthy := dial(t, server, "u2", "Healthy")
	waitUntil(t, func() bool { return hub.Connections() == 2 }, "both should be connected")

	// The stuck tab never reads. Large messages fill its network buffers,
	// then its queue, and it is dropped; the healthy tab gets every one.
	big := map[string]any{"type": "big", "padding": strings.Repeat("x", 256*1024)}
	for i := 0; i < 4*sendQueue && hub.Connections() == 2; i++ {
		hub.Broadcast(big)
		assert.Equal(t, "big", receive(t, healthy)["type"])
	}

	waitUntil(t, func() bool { return hub.Connections() == 1 }, "the stuck connection should be dropped")
	_ = stuck
}

func TestClosingTheHubEndsEveryConnection(t *testing.T) {
	hub, presence := newHub(t, time.Minute)
	server := testServer(t, hub)

	conn := dial(t, server, "u1", "Shobu")
	waitUntil(t, func() bool { return hub.Connections() == 1 }, "should be connected")

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	require.NoError(t, hub.Close(ctx))

	require.NoError(t, conn.SetReadDeadline(time.Now().Add(testTimeout)))
	_, _, err := conn.ReadMessage()
	assert.Error(t, err, "the connection should be closed")
	assert.Equal(t, []string{"Shobu"}, presence.names(), "stopping the server is not players leaving")

	// A connection arriving afterwards is refused.
	late, response, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), http.Header{"Origin": {testOrigin}})
	if err == nil {
		_ = response.Body.Close()
		require.NoError(t, late.SetReadDeadline(time.Now().Add(testTimeout)))
		_, _, err = late.ReadMessage()
		_ = late.Close()
	}
	assert.Error(t, err)
}

func TestTheUpgraderAcceptsOnlyTheSitesPages(t *testing.T) {
	upgrader := NewUpgrader([]string{"https://duelmasters.example/", "http://localhost:3000"})

	for origin, allowed := range map[string]bool{
		"https://duelmasters.example": true,
		"HTTPS://DuelMasters.Example": true,
		"http://localhost:3000":       true,
		"https://evil.example":        false,
		"http://duelmasters.example":  false,
		"http://localhost:3001":       false,
		"":                            false,
		"null":                        false,
	} {
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		if origin != "" {
			request.Header.Set("Origin", origin)
		}
		assert.Equal(t, allowed, upgrader.CheckOrigin(request), "%q", origin)
	}
}
