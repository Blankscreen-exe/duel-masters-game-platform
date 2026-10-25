package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// liveServer runs the API over real HTTP, as live connections need.
func (s testServer) liveServer(t *testing.T) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(s.handler)
	t.Cleanup(server.Close)
	return server
}

// openLive opens the live connection from the site's page, with the session
// cookie when one is given.
func openLive(t *testing.T, server *httptest.Server, session *http.Cookie, origin string) (*websocket.Conn, *http.Response, error) {
	t.Helper()

	header := http.Header{}
	if origin != "" {
		header.Set("Origin", origin)
	}
	if session != nil {
		header.Set("Cookie", session.String())
	}

	conn, response, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http")+"/api/realtime", header)
	if conn != nil {
		t.Cleanup(func() { _ = conn.Close() })
	}
	return conn, response, err
}

func readLive(t *testing.T, conn *websocket.Conn) map[string]any {
	t.Helper()

	require.NoError(t, conn.SetReadDeadline(time.Now().Add(5*time.Second)))
	_, data, err := conn.ReadMessage()
	require.NoError(t, err)

	var message map[string]any
	require.NoError(t, json.Unmarshal(data, &message))
	return message
}

func TestVisitorsAndPlayersOpenTheLiveConnection(t *testing.T) {
	api := newTestServer(t, false)
	server := api.liveServer(t)
	session := api.signUp(t, "Shobu")

	visitor, _, err := openLive(t, server, nil, testSiteOrigin)
	require.NoError(t, err)
	player, _, err := openLive(t, server, session, testSiteOrigin)
	require.NoError(t, err)

	require.Eventually(t, func() bool { return len(api.hub.Online()) == 1 }, 5*time.Second, 5*time.Millisecond)
	assert.Equal(t, "Shobu", api.hub.Online()[0].Username, "the player is known by their session")

	api.hub.Broadcast(map[string]any{"type": "hello"})
	assert.Equal(t, "hello", readLive(t, visitor)["type"])
	assert.Equal(t, "hello", readLive(t, player)["type"])
}

func TestAnotherSitesPageCannotOpenTheLiveConnection(t *testing.T) {
	api := newTestServer(t, false)
	server := api.liveServer(t)
	session := api.signUp(t, "Shobu")

	for _, origin := range []string{"https://evil.example", ""} {
		_, response, err := openLive(t, server, session, origin)
		require.Error(t, err, "%q", origin)
		require.NotNil(t, response)
		assert.Equal(t, http.StatusForbidden, response.StatusCode, "%q", origin)
		_ = response.Body.Close()
	}
	assert.Empty(t, api.hub.Online())
}

func TestTheLiveConnectionIsOnlyAWebsocket(t *testing.T) {
	api := newTestServer(t, false)

	request := httptest.NewRequest(http.MethodGet, "/api/realtime", nil)
	request.Header.Set("Origin", testSiteOrigin)
	recorder := httptest.NewRecorder()
	api.handler.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
}
