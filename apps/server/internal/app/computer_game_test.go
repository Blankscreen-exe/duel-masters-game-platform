package app

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/cards/cardstest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/config"
)

const testSiteOrigin = "http://site.test"

// site talks to a running server as a browser does.
type site struct {
	t       *testing.T
	address string
	session *http.Cookie
}

// call makes a request and decodes the answer into into, if any, returning its
// status.
func (s *site) call(method, path string, body, into any) int {
	s.t.Helper()

	data, err := json.Marshal(body)
	require.NoError(s.t, err)
	request, err := http.NewRequest(method, "http://"+s.address+"/api"+path, bytes.NewReader(data))
	require.NoError(s.t, err)
	request.Header.Set("Content-Type", "application/json")
	if s.session != nil {
		request.AddCookie(s.session)
	}

	response, err := http.DefaultClient.Do(request)
	require.NoError(s.t, err)
	defer func() { _ = response.Body.Close() }()

	for _, cookie := range response.Cookies() {
		s.session = cookie
	}
	if into != nil {
		require.NoError(s.t, json.NewDecoder(response.Body).Decode(into))
	}
	return response.StatusCode
}

// The whole server, with the real card database and its seeds: the starting
// opponents are offered, one takes its seat, answers the coin toss and plays
// its turn, and the game ends and closes cleanly with the server.
func TestAPlayerPlaysAStartingOpponentOnTheWholeServer(t *testing.T) {
	cfg, err := config.Load([]string{
		"DATA_DIR=" + filepath.Join(t.TempDir(), "data"),
		"CARD_DATABASE_DIR=" + cardstest.RealDatabase(t),
		"SITE_ORIGINS=" + testSiteOrigin,
	})
	require.NoError(t, err)
	app, err := New(context.Background(), cfg, quietLogger())
	require.NoError(t, err)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	served := make(chan error, 1)
	go func() { served <- app.Serve(ctx, listener) }()
	defer func() {
		cancel()
		select {
		case err := <-served:
			assert.NoError(t, err, "the server shuts down cleanly")
		case <-time.After(shutdownTimeout + 5*time.Second):
			t.Error("the server did not shut down")
		}
	}()

	browser := &site{t: t, address: listener.Addr().String()}
	require.Equal(t, http.StatusCreated, browser.call(http.MethodPost, "/auth/signup", map[string]string{"username": "Shobu", "email": "shobu@example.test", "password": "a long enough password"}, nil))

	var offered []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	require.Equal(t, http.StatusOK, browser.call(http.MethodGet, "/computer-opponents", nil, &offered))
	require.Len(t, offered, 6)
	assert.Equal(t, "Grimwhisper", offered[0].Name)

	var starters []struct {
		ID string `json:"id"`
	}
	require.Equal(t, http.StatusOK, browser.call(http.MethodGet, "/starter-decks", nil, &starters))
	require.NotEmpty(t, starters)

	var started struct {
		GameID string `json:"gameId"`
	}
	require.Equal(t, http.StatusCreated, browser.call(http.MethodPost, "/computer-opponents/"+offered[0].ID+"/games", map[string]string{"deckId": starters[0].ID}, &started))

	header := http.Header{"Origin": {testSiteOrigin}, "Cookie": {browser.session.String()}}
	socket, _, err := websocket.DefaultDialer.Dial("ws://"+browser.address+"/api/games/"+started.GameID+"/socket", header)
	require.NoError(t, err)
	defer func() { _ = socket.Close() }()
	require.NoError(t, socket.WriteJSON(map[string]any{"header": "join_match"}))

	// The player passes every turn of theirs, until the computer has done
	// something on the board, then gives up.
	require.NoError(t, socket.SetReadDeadline(time.Now().Add(time.Minute)))
	ended := false
	for !ended {
		var message struct {
			Header string `json:"header"`
			State  struct {
				MyTurn   bool `json:"myTurn"`
				Opponent struct {
					Manazone []json.RawMessage `json:"manazone"`
				} `json:"opponent"`
			} `json:"state"`
			Practice bool `json:"practice"`
			Counted  bool `json:"counted"`
		}
		require.NoError(t, socket.ReadJSON(&message), "the game goes on")

		switch message.Header {
		case "choose_toss":
			require.NoError(t, socket.WriteJSON(map[string]any{"header": "toss_chosen", "prediction": 1}))
		case "choose_turn":
			require.NoError(t, socket.WriteJSON(map[string]any{"header": "turn_chosen", "player": 1}))
		case "state_update":
			switch {
			case len(message.State.Opponent.Manazone) > 0:
				require.NoError(t, socket.WriteJSON(map[string]any{"header": "resign"}))
			case message.State.MyTurn:
				require.NoError(t, socket.WriteJSON(map[string]any{"header": "end_turn"}))
			}
		case "duel_finished":
			assert.True(t, message.Practice)
			assert.False(t, message.Counted)
			ended = true
		}
	}

	require.Eventually(t, func() bool {
		return browser.call(http.MethodGet, "/matches/"+started.GameID, nil, nil) == http.StatusOK
	}, 10*time.Second, 20*time.Millisecond, "the game is kept in the history")
}
