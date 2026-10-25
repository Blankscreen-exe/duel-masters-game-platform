package httpapi

import (
	"net/http"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/httpapi/api"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/lobby"
)

// playUntilTheGuestSurrenders plays a game the host opens and the guest joins
// until the guest surrenders, and returns its id.
func (s playServer) playUntilTheGuestSurrenders(t *testing.T, host, guest player) string {
	t.Helper()

	open := s.openGame(t, host, api.OpenGameInput{Listed: true, Spectators: true})
	response := s.call(http.MethodPost, "/api/lobby/games/"+open.Id+"/join", api.JoinGameInput{DeckId: guest.deckID}, guest.session)
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())
	gameID := decode[api.GameStarted](t, response).GameId

	hostSocket, guestSocket := s.play(t, host, guest, gameID)
	require.NoError(t, guestSocket.WriteJSON(map[string]any{"header": "resign"}))
	gameMessage(t, hostSocket, "duel_finished")

	return gameID
}

// rematchMessage reads a tab's live messages until the rematch of the game
// arrives as it matches.
func rematchMessage(t *testing.T, tab *websocket.Conn, gameID string, matches func(rematch map[string]any) bool) map[string]any {
	t.Helper()

	message := liveMessageWhere(t, tab, "rematch", func(message map[string]any) bool {
		rematch := message["rematch"].(map[string]any)
		return rematch["gameId"] == gameID && matches(rematch)
	})
	return message["rematch"].(map[string]any)
}

func TestARematchThroughTheSite(t *testing.T) {
	site := newPlayServer(t)
	shobu := site.player(t, "Shobu")
	kokujo := site.player(t, "Kokujo")
	mimi := site.player(t, "Mimi")
	gameID := site.playUntilTheGuestSurrenders(t, shobu, kokujo)

	for _, tab := range []*websocket.Conn{shobu.tab, kokujo.tab} {
		rematchMessage(t, tab, gameID, func(map[string]any) bool { return true })
	}

	offered := decode[api.Rematch](t, site.call(http.MethodGet, "/api/rematches/"+gameID, nil, shobu.session))
	assert.Equal(t, gameID, offered.GameId)
	assert.Equal(t, "Shobu", offered.You.Username)
	assert.Equal(t, "Shobu's deck", offered.You.Deck.Name)
	assert.Equal(t, []string{"Fire"}, offered.You.Deck.Civilizations)
	require.NotNil(t, offered.You.Deck.Id)
	assert.Equal(t, shobu.deckID, *offered.You.Deck.Id, "a player is shown their own deck's id")
	assert.Equal(t, "Kokujo", offered.Opponent.Username)
	assert.Equal(t, "Kokujo's deck", offered.Opponent.Deck.Name)
	assert.Nil(t, offered.Opponent.Deck.Id, "but not their opponent's")
	assert.False(t, offered.You.Ready)
	require.NotNil(t, offered.TossCallerId)
	assert.Equal(t, offered.Opponent.Id, *offered.TossCallerId, "Kokujo lost, so calls the next toss")

	response := site.call(http.MethodGet, "/api/rematches/"+gameID, nil, mimi.session)
	assert.Equal(t, http.StatusNotFound, response.Code, "only its players see a rematch")
	response = site.call(http.MethodGet, "/api/rematches/"+gameID, nil, nil)
	assert.Equal(t, http.StatusUnauthorized, response.Code)

	response = site.call(http.MethodPost, "/api/lobby/games", api.OpenGameInput{DeckId: shobu.deckID, FormatId: offered.FormatId}, shobu.session)
	require.Equal(t, http.StatusConflict, response.Code)
	assert.Equal(t, "You are deciding on a rematch. Accept or decline it first.", decode[api.Error](t, response).Message)

	// Kokujo switches decks; Shobu sees it, by name only.
	other := site.createDeck(t, kokujo.session, api.DeckInput{Name: "Water Wall", Cards: fullDeck(site.cardIDs)})
	response = site.call(http.MethodPut, "/api/rematches/"+gameID+"/deck", api.RematchDeckInput{DeckId: other.Id}, kokujo.session)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	assert.Equal(t, other.Id, *decode[api.Rematch](t, response).You.Deck.Id)
	seen := rematchMessage(t, shobu.tab, gameID, func(rematch map[string]any) bool {
		return rematch["opponent"].(map[string]any)["deck"].(map[string]any)["name"] == "Water Wall"
	})
	assert.NotContains(t, seen["opponent"].(map[string]any)["deck"], "id")

	short := site.createDeck(t, kokujo.session, api.DeckInput{Name: "Unfinished", Cards: fullDeck(site.cardIDs)[:1]})
	response = site.call(http.MethodPut, "/api/rematches/"+gameID+"/deck", api.RematchDeckInput{DeckId: short.Id}, kokujo.session)
	require.Equal(t, http.StatusBadRequest, response.Code)
	assert.Equal(t, codeDeckNotLegal, decode[api.Error](t, response).Code)

	response = site.call(http.MethodPost, "/api/rematches/"+gameID+"/ready", nil, shobu.session)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	assert.True(t, decode[api.Rematch](t, response).You.Ready, "Shobu offers")
	rematchMessage(t, kokujo.tab, gameID, func(rematch map[string]any) bool {
		return rematch["opponent"].(map[string]any)["ready"] == true
	})

	response = site.call(http.MethodPost, "/api/rematches/"+gameID+"/ready", nil, kokujo.session)
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())
	rematchID := decode[api.GameStarted](t, response).GameId
	assert.NotEqual(t, gameID, rematchID)
	for _, tab := range []*websocket.Conn{shobu.tab, kokujo.tab} {
		assert.Equal(t, rematchID, liveMessage(t, tab, "game_started")["gameId"], "both are told to go to the new game")
	}

	game := decode[api.LiveGame](t, site.call(http.MethodGet, "/api/games/"+rematchID, nil, shobu.session))
	assert.Equal(t, "Water Wall", game.Guest.DeckName, "played with the decks chosen")
	response = site.call(http.MethodGet, "/api/rematches/"+gameID, nil, shobu.session)
	assert.Equal(t, http.StatusNotFound, response.Code, "decided")
}

func TestDecliningARematchThroughTheSite(t *testing.T) {
	site := newPlayServer(t)
	shobu := site.player(t, "Shobu")
	kokujo := site.player(t, "Kokujo")
	gameID := site.playUntilTheGuestSurrenders(t, shobu, kokujo)
	rematchMessage(t, shobu.tab, gameID, func(map[string]any) bool { return true })

	response := site.call(http.MethodDelete, "/api/rematches/"+gameID, nil, kokujo.session)
	require.Equal(t, http.StatusNoContent, response.Code, response.Body.String())

	for _, tab := range []*websocket.Conn{shobu.tab, kokujo.tab} {
		ended := liveMessage(t, tab, "rematch_ended")
		assert.Equal(t, gameID, ended["gameId"])
		assert.Equal(t, "declined", ended["reason"])
		assert.Equal(t, "Kokujo", ended["by"])
	}

	response = site.call(http.MethodDelete, "/api/rematches/"+gameID, nil, kokujo.session)
	assert.Equal(t, http.StatusNotFound, response.Code)
	response = site.call(http.MethodPost, "/api/rematches/"+gameID+"/ready", nil, shobu.session)
	assert.Equal(t, http.StatusNotFound, response.Code)
	site.openGame(t, shobu, api.OpenGameInput{Listed: true})
}

func TestARematchWaitsWhileNewGamesArePaused(t *testing.T) {
	site := newPlayServer(t)
	shobu := site.player(t, "Shobu")
	kokujo := site.player(t, "Kokujo")
	gameID := site.playUntilTheGuestSurrenders(t, shobu, kokujo)
	_, err := site.settings.Set(t.Context(), lobby.NewGamesOpen.Key, "false")
	require.NoError(t, err)

	response := site.call(http.MethodPost, "/api/rematches/"+gameID+"/ready", nil, shobu.session)
	require.Equal(t, http.StatusForbidden, response.Code)
	assert.Equal(t, codeGamesPaused, decode[api.Error](t, response).Code)
}

// fullDeck is 4 copies of each card: 40 cards.
func fullDeck(cardIDs []string) []api.DeckEntry {
	entries := make([]api.DeckEntry, 0, len(cardIDs))
	for _, id := range cardIDs {
		entries = append(entries, api.DeckEntry{CardId: id, Count: 4})
	}
	return entries
}
