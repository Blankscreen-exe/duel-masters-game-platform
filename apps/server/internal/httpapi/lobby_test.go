package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/cards"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/games/gamestest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/httpapi/api"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/lobby"
)

// playServer is the API over real engine cards, served over HTTP so players
// keep live connections and play over game sockets.
type playServer struct {
	testServer
	live *httptest.Server
	// cardIDs are ten real engine cards, enough for a legal deck.
	cardIDs []string
}

func newPlayServer(t *testing.T) playServer {
	t.Helper()

	var ids []string
	server := newTestServerWith(t, false, func(t *testing.T, db *gorm.DB) (*cards.Catalogue, string) {
		catalogue, cardIDs := gamestest.RealCards(t, db)
		ids = cardIDs
		return catalogue, ""
	})

	return playServer{testServer: server, live: server.liveServer(t), cardIDs: ids}
}

// player is someone with an account, the site open in a tab, and a legal
// deck.
type player struct {
	name    string
	session *http.Cookie
	tab     *websocket.Conn
	deckID  string
}

func (s playServer) player(t *testing.T, name string) player {
	t.Helper()

	session := s.signUp(t, name)
	tab, _, err := openLive(t, s.live, session, testSiteOrigin)
	require.NoError(t, err)

	entries := make([]api.DeckEntry, 0, len(s.cardIDs))
	for _, id := range s.cardIDs {
		entries = append(entries, api.DeckEntry{CardId: id, Count: 4})
	}
	deck := s.createDeck(t, session, api.DeckInput{Name: name + "'s deck", Cards: entries})

	require.Eventually(t, func() bool {
		for _, viewer := range s.hub.Online() {
			if viewer.Username == name {
				return true
			}
		}
		return false
	}, 5*time.Second, 5*time.Millisecond, "%s should be online", name)

	return player{name: name, session: session, tab: tab, deckID: deck.Id}
}

// liveMessage reads a tab's live messages until one of the type arrives.
func liveMessage(t *testing.T, tab *websocket.Conn, messageType string) map[string]any {
	t.Helper()
	return liveMessageWhere(t, tab, messageType, func(map[string]any) bool { return true })
}

// liveMessageWhere reads a tab's live messages until one of the type that
// also matches arrives, skipping older ones.
func liveMessageWhere(t *testing.T, tab *websocket.Conn, messageType string, matches func(map[string]any) bool) map[string]any {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for {
		require.NoError(t, tab.SetReadDeadline(deadline))
		_, data, err := tab.ReadMessage()
		require.NoError(t, err, "waiting for %q", messageType)

		var message map[string]any
		require.NoError(t, json.Unmarshal(data, &message))
		if message["type"] == messageType && matches(message) {
			return message
		}
	}
}

// openGameSocket opens a game's websocket as the duel board does.
func (s playServer) openGameSocket(t *testing.T, session *http.Cookie, gameID string) (*websocket.Conn, *http.Response, error) {
	t.Helper()

	header := http.Header{"Origin": {testSiteOrigin}}
	if session != nil {
		header.Set("Cookie", session.String())
	}
	conn, response, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(s.live.URL, "http")+"/api/games/"+gameID+"/socket", header)
	if conn != nil {
		t.Cleanup(func() { _ = conn.Close() })
	}
	return conn, response, err
}

// gameMessage reads a game socket until a message with one of the headers
// arrives.
func gameMessage(t *testing.T, conn *websocket.Conn, headers ...string) map[string]any {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for {
		require.NoError(t, conn.SetReadDeadline(deadline))
		var message map[string]any
		require.NoError(t, conn.ReadJSON(&message), "waiting for %v", headers)
		if slices.Contains(headers, message["header"].(string)) {
			return message
		}
	}
}

func (s playServer) openGame(t *testing.T, host player, input api.OpenGameInput) api.OpenGame {
	t.Helper()

	if input.DeckId == "" {
		input.DeckId = host.deckID
	}
	if input.FormatId == "" {
		open, _ := s.formats(t)
		input.FormatId = open.Id
	}
	response := s.call(http.MethodPost, "/api/lobby/games", input, host.session)
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())
	return decode[api.OpenGame](t, response)
}

func TestTwoPlayersPlayAGameThroughTheSite(t *testing.T) {
	site := newPlayServer(t)
	shobu := site.player(t, "Shobu")
	kokujo := site.player(t, "Kokujo")
	mimi := site.player(t, "Mimi")

	name := "Friday night"
	open := site.openGame(t, shobu, api.OpenGameInput{Name: &name, Listed: true, Spectators: true})
	assert.Equal(t, "Friday night", open.Name)
	assert.Equal(t, "Shobu", open.Host.Username)
	assert.True(t, open.Listed)

	liveMessageWhere(t, kokujo.tab, "lobby", func(message map[string]any) bool {
		return len(message["lobby"].(map[string]any)["openGames"].([]any)) == 1
	})
	lobbyNow := decode[api.Lobby](t, site.call(http.MethodGet, "/api/lobby", nil, kokujo.session))
	require.Len(t, lobbyNow.OpenGames, 1)
	assert.Equal(t, open, lobbyNow.OpenGames[0])

	response := site.call(http.MethodPost, "/api/lobby/games/"+open.Id+"/join", api.JoinGameInput{DeckId: kokujo.deckID}, kokujo.session)
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())
	gameID := decode[api.GameStarted](t, response).GameId

	for _, tab := range []*websocket.Conn{shobu.tab, kokujo.tab} {
		assert.Equal(t, gameID, liveMessage(t, tab, "game_started")["gameId"], "both are told to go to the game")
	}

	game := decode[api.LiveGame](t, site.call(http.MethodGet, "/api/games/"+gameID, nil, shobu.session))
	assert.Equal(t, api.GameRolePlayer, game.Role)
	assert.Equal(t, "Shobu", game.Host.Username)
	assert.Equal(t, "Kokujo's deck", game.Guest.DeckName)
	watching := decode[api.LiveGame](t, site.call(http.MethodGet, "/api/games/"+gameID, nil, mimi.session))
	assert.Equal(t, api.GameRoleSpectator, watching.Role)

	// Both play over real game sockets, as the duel board does.
	host, _, err := site.openGameSocket(t, shobu.session, gameID)
	require.NoError(t, err)
	guest, _, err := site.openGameSocket(t, kokujo.session, gameID)
	require.NoError(t, err)
	for _, conn := range []*websocket.Conn{host, guest} {
		require.NoError(t, conn.WriteJSON(map[string]any{"header": "join_match"}))
	}

	// The host is told either to call the toss or who calls it.
	caller, other := host, guest
	if gameMessage(t, host, "choose_toss", "toss_being_chosen")["header"] == "toss_being_chosen" {
		caller, other = guest, host
	}
	require.NoError(t, caller.WriteJSON(map[string]any{"header": "toss_chosen", "prediction": 1}))
	result := gameMessage(t, other, "toss_result")
	chooser := host
	if result["winner"].(map[string]any)["username"] == "Kokujo" {
		chooser = guest
	}
	gameMessage(t, chooser, "choose_turn")
	require.NoError(t, chooser.WriteJSON(map[string]any{"header": "turn_chosen", "player": 1}))
	gameMessage(t, host, "state_update")

	require.NoError(t, guest.WriteJSON(map[string]any{"header": "resign"}))
	finished := gameMessage(t, host, "duel_finished")
	assert.Equal(t, "surrender", finished["ending"])
	assert.Equal(t, "Shobu", finished["winner"].(map[string]any)["username"])

	require.Eventually(t, func() bool { return len(site.games.Games()) == 0 }, 5*time.Second, 5*time.Millisecond)
}

func TestOpeningAndJoiningAreRefusedWhenTheyCannotWork(t *testing.T) {
	site := newPlayServer(t)
	shobu := site.player(t, "Shobu")
	kokujo := site.player(t, "Kokujo")
	openFormat, _ := site.formats(t)

	response := site.call(http.MethodPost, "/api/lobby/games", api.OpenGameInput{DeckId: shobu.deckID, FormatId: openFormat.Id}, nil)
	assert.Equal(t, http.StatusUnauthorized, response.Code)

	short := site.createDeck(t, shobu.session, api.DeckInput{Name: "Unfinished", Cards: []api.DeckEntry{{CardId: site.cardIDs[0], Count: 4}}})
	response = site.call(http.MethodPost, "/api/lobby/games", api.OpenGameInput{DeckId: short.Id, FormatId: openFormat.Id}, shobu.session)
	require.Equal(t, http.StatusBadRequest, response.Code)
	refused := decode[api.Error](t, response)
	assert.Equal(t, codeDeckNotLegal, refused.Code)
	require.NotNil(t, refused.Problems)
	assert.Equal(t, api.DeckProblemCodeDeckSize, (*refused.Problems)[0].Code)
	assert.Contains(t, refused.Message, "The deck has 4 cards; it needs exactly 40.")

	response = site.call(http.MethodPost, "/api/lobby/games", api.OpenGameInput{DeckId: shobu.deckID, FormatId: "nope"}, shobu.session)
	require.Equal(t, http.StatusBadRequest, response.Code)
	assert.Equal(t, "formatId", *decode[api.Error](t, response).Field)

	open := site.openGame(t, shobu, api.OpenGameInput{Listed: true})
	response = site.call(http.MethodPost, "/api/lobby/games", api.OpenGameInput{DeckId: shobu.deckID, FormatId: openFormat.Id}, shobu.session)
	require.Equal(t, http.StatusConflict, response.Code)
	assert.Equal(t, api.Error{Code: codeBusy, Message: "You are already hosting a game. Cancel it first."}, decode[api.Error](t, response))

	response = site.call(http.MethodPost, "/api/lobby/games/"+open.Id+"/join", api.JoinGameInput{DeckId: shobu.deckID}, shobu.session)
	require.Equal(t, http.StatusConflict, response.Code)
	assert.Equal(t, codeOwnGame, decode[api.Error](t, response).Code)

	response = site.call(http.MethodDelete, "/api/lobby/games/"+open.Id, nil, kokujo.session)
	assert.Equal(t, http.StatusForbidden, response.Code)
	assert.Equal(t, codeNotHost, decode[api.Error](t, response).Code)

	response = site.call(http.MethodDelete, "/api/lobby/games/"+open.Id, nil, shobu.session)
	assert.Equal(t, http.StatusNoContent, response.Code)

	for _, request := range []struct{ method, path string }{
		{http.MethodGet, "/api/lobby/games/" + open.Id},
		{http.MethodDelete, "/api/lobby/games/" + open.Id},
	} {
		response = site.call(request.method, request.path, nil, kokujo.session)
		assert.Equal(t, http.StatusNotFound, response.Code, request.path)
	}
	response = site.call(http.MethodPost, "/api/lobby/games/"+open.Id+"/join", api.JoinGameInput{DeckId: kokujo.deckID}, kokujo.session)
	assert.Equal(t, http.StatusNotFound, response.Code)

	_, err := site.settings.Set(t.Context(), lobby.NewGamesOpen.Key, "false")
	require.NoError(t, err)
	response = site.call(http.MethodPost, "/api/lobby/games", api.OpenGameInput{DeckId: kokujo.deckID, FormatId: openFormat.Id}, kokujo.session)
	assert.Equal(t, http.StatusForbidden, response.Code)
	assert.Equal(t, codeGamesPaused, decode[api.Error](t, response).Code)
	assert.True(t, decode[api.Lobby](t, site.call(http.MethodGet, "/api/lobby", nil, kokujo.session)).Paused)
}

func TestAnInviteOnlyGameIsReachedByItsLink(t *testing.T) {
	site := newPlayServer(t)
	shobu := site.player(t, "Shobu")
	kokujo := site.player(t, "Kokujo")

	open := site.openGame(t, shobu, api.OpenGameInput{Listed: false})

	assert.Empty(t, decode[api.Lobby](t, site.call(http.MethodGet, "/api/lobby", nil, kokujo.session)).OpenGames)
	response := site.call(http.MethodGet, "/api/lobby/games/"+open.Id, nil, kokujo.session)
	require.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, open, decode[api.OpenGame](t, response))
}

func TestQuickMatchThroughTheSite(t *testing.T) {
	site := newPlayServer(t)
	shobu := site.player(t, "Shobu")
	kokujo := site.player(t, "Kokujo")
	openFormat, _ := site.formats(t)

	response := site.call(http.MethodPost, "/api/quick-match", api.QuickMatchInput{FormatId: openFormat.Id, DeckId: shobu.deckID}, shobu.session)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	entered := decode[api.QuickMatchEntered](t, response)
	assert.Nil(t, entered.GameId)
	require.NotNil(t, entered.Ticket)
	assert.Equal(t, "Open", entered.Ticket.FormatName)

	liveMessageWhere(t, kokujo.tab, "quick_match_queue", func(message map[string]any) bool {
		return len(message["waiting"].([]any)) == 1
	})

	status := decode[api.QuickMatchStatus](t, site.call(http.MethodGet, "/api/quick-match", nil, shobu.session))
	require.NotNil(t, status.Ticket)
	assert.Equal(t, []api.QueueCount{{FormatId: openFormat.Id, Players: 1}}, status.Waiting)

	response = site.call(http.MethodPost, "/api/quick-match", api.QuickMatchInput{FormatId: openFormat.Id, DeckId: kokujo.deckID}, kokujo.session)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	paired := decode[api.QuickMatchEntered](t, response)
	require.NotNil(t, paired.GameId)

	assert.Equal(t, *paired.GameId, liveMessage(t, shobu.tab, "game_started")["gameId"], "the waiting player is told")
	game := decode[api.LiveGame](t, site.call(http.MethodGet, "/api/games/"+*paired.GameId, nil, kokujo.session))
	assert.True(t, game.Matchmaking)
	assert.Equal(t, "Quick match", game.Name)
}

func TestLeavingQuickMatchThroughTheSite(t *testing.T) {
	site := newPlayServer(t)
	shobu := site.player(t, "Shobu")
	openFormat, _ := site.formats(t)

	response := site.call(http.MethodPost, "/api/quick-match", api.QuickMatchInput{FormatId: openFormat.Id, DeckId: shobu.deckID}, shobu.session)
	require.Equal(t, http.StatusOK, response.Code)

	for range 2 {
		response = site.call(http.MethodDelete, "/api/quick-match", nil, shobu.session)
		assert.Equal(t, http.StatusNoContent, response.Code, "leaving twice is fine")
	}
	status := decode[api.QuickMatchStatus](t, site.call(http.MethodGet, "/api/quick-match", nil, shobu.session))
	assert.Nil(t, status.Ticket)
	assert.Empty(t, status.Waiting)
}

func TestWhoMayOpenAGameSocket(t *testing.T) {
	site := newPlayServer(t)
	shobu := site.player(t, "Shobu")
	kokujo := site.player(t, "Kokujo")
	mimi := site.player(t, "Mimi")

	open := site.openGame(t, shobu, api.OpenGameInput{Listed: true, Spectators: false})
	response := site.call(http.MethodPost, "/api/lobby/games/"+open.Id+"/join", api.JoinGameInput{DeckId: kokujo.deckID}, kokujo.session)
	require.Equal(t, http.StatusCreated, response.Code)
	gameID := decode[api.GameStarted](t, response).GameId

	for name, test := range map[string]struct {
		session *http.Cookie
		gameID  string
		status  int
	}{
		"a visitor":                    {nil, gameID, http.StatusUnauthorized},
		"a spectator of a closed game": {mimi.session, gameID, http.StatusForbidden},
		"an unknown game":              {shobu.session, "no-such-game", http.StatusNotFound},
	} {
		_, socketResponse, err := site.openGameSocket(t, test.session, test.gameID)
		require.Error(t, err, name)
		require.NotNil(t, socketResponse, name)
		assert.Equal(t, test.status, socketResponse.StatusCode, name)
		_ = socketResponse.Body.Close()
	}

	response = site.call(http.MethodGet, "/api/games/"+gameID, nil, mimi.session)
	assert.Equal(t, http.StatusForbidden, response.Code)
	assert.Equal(t, codeNotWatchable, decode[api.Error](t, response).Code)

	// Another site's page is refused even with the player's cookie.
	header := http.Header{"Origin": {"https://evil.example"}, "Cookie": {shobu.session.String()}}
	_, foreign, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(site.live.URL, "http")+"/api/games/"+gameID+"/socket", header)
	require.Error(t, err)
	assert.Equal(t, http.StatusForbidden, foreign.StatusCode)
	_ = foreign.Body.Close()
}

func TestVisitorsSeeTheNumbersButNotTheLobby(t *testing.T) {
	site := newPlayServer(t)
	visitor, _, err := openLive(t, site.live, nil, testSiteOrigin)
	require.NoError(t, err)
	shobu := site.player(t, "Shobu")

	site.openGame(t, shobu, api.OpenGameInput{Listed: true})

	liveMessageWhere(t, visitor, "site_stats", func(message map[string]any) bool {
		return message["stats"].(map[string]any)["playersOnline"] == float64(1)
	})
	assert.Equal(t, api.SiteStats{PlayersOnline: 1, GamesRunning: 0}, decode[api.SiteStats](t, site.call(http.MethodGet, "/api/stats", nil, nil)))

	response := site.call(http.MethodGet, "/api/lobby", nil, nil)
	assert.Equal(t, http.StatusUnauthorized, response.Code, "the lobby is for players")
}
