package httpapi

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/decks"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/games"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/games/gamestest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/httpapi/api"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/opponents"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/opponents/opponentstest"
)

// addOpponent adds a computer opponent playing the site's real cards, moving
// without a pause, in the format.
func (s playServer) addOpponent(t *testing.T, formatID string) opponentstest.Added {
	t.Helper()

	_, err := s.settings.Set(context.Background(), opponents.Pause.Key, "0")
	require.NoError(t, err)

	entries := make([]decks.Entry, 0, len(s.cardIDs))
	for _, id := range s.cardIDs {
		entries = append(entries, decks.Entry{CardID: id, Count: decks.MaxCopies})
	}
	return opponentstest.Add(t, s.db, "Grim", formatID, entries)
}

func TestAGameAgainstTheComputerThroughTheSite(t *testing.T) {
	site := newPlayServer(t)
	shobu := site.player(t, "Shobu")
	open, _ := site.formats(t)
	opponent := site.addOpponent(t, open.Id)

	listed := site.call(http.MethodGet, "/api/computer-opponents", nil, shobu.session)
	require.Equal(t, http.StatusOK, listed.Code, listed.Body.String())
	offers := decode[[]api.ComputerOpponent](t, listed)
	require.Len(t, offers, 1)
	assert.Equal(t, opponent.ID, offers[0].Id)
	assert.Equal(t, "Grim", offers[0].Name)
	assert.Equal(t, "rush", offers[0].Style)
	assert.Equal(t, api.ComputerOpponentStrengthNormal, offers[0].Strength)
	assert.Equal(t, open.Name, offers[0].FormatName)

	started := site.call(http.MethodPost, "/api/computer-opponents/"+opponent.ID+"/games", api.JoinGameInput{DeckId: shobu.deckID}, shobu.session)
	require.Equal(t, http.StatusCreated, started.Code, started.Body.String())
	gameID := decode[api.GameStarted](t, started).GameId

	live := decode[api.LiveGame](t, site.call(http.MethodGet, "/api/games/"+gameID, nil, shobu.session))
	assert.True(t, live.Guest.Computer)
	assert.False(t, live.Host.Computer)
	assert.False(t, live.Spectators)

	lobbyNow := decode[api.Lobby](t, site.call(http.MethodGet, "/api/lobby", nil, shobu.session))
	assert.Empty(t, lobbyNow.RunningGames, "it is not listed")
	require.Len(t, lobbyNow.Online, 1)
	require.NotNil(t, lobbyNow.Online[0].Activity)
	assert.True(t, lobbyNow.Online[0].Activity.Computer, "shown as playing the computer")

	// Someone else cannot watch it.
	mimi := site.player(t, "Mimi")
	assert.Equal(t, http.StatusForbidden, site.call(http.MethodGet, "/api/games/"+gameID, nil, mimi.session).Code)

	// The player settles the toss with the computer and gives up.
	socket, _, err := site.openGameSocket(t, shobu.session, gameID)
	require.NoError(t, err)
	game := readGame(t, socket)
	require.NoError(t, game.WriteJSON(map[string]any{"header": "join_match"}))
	if gameMessage(t, game, "choose_toss", "toss_being_chosen")["header"] == "choose_toss" {
		require.NoError(t, game.WriteJSON(map[string]any{"header": "toss_chosen", "prediction": 1}))
	}
	if result := gameMessage(t, game, "toss_result"); result["winner"].(map[string]any)["username"] == "Shobu" {
		gameMessage(t, game, "choose_turn")
		require.NoError(t, game.WriteJSON(map[string]any{"header": "turn_chosen", "player": 1}))
	}
	gameMessage(t, game, "state_update")
	require.NoError(t, game.WriteJSON(map[string]any{"header": "resign"}))
	finished := gameMessage(t, game, "duel_finished")
	assert.Equal(t, false, finished["counted"])
	assert.Equal(t, true, finished["practice"])

	// The computer agrees to a rematch at once.
	rematchNow := decode[api.Rematch](t, site.call(http.MethodGet, "/api/rematches/"+gameID, nil, shobu.session))
	assert.True(t, rematchNow.Opponent.Computer)
	assert.True(t, rematchNow.Opponent.Ready)
	assert.False(t, rematchNow.You.Computer)
	require.Equal(t, http.StatusNoContent, site.call(http.MethodDelete, "/api/rematches/"+gameID, nil, shobu.session).Code)

	// It is kept in the player's history, marked, without ratings.
	var match api.Match
	require.Eventually(t, func() bool {
		response := site.call(http.MethodGet, "/api/matches/"+gameID, nil, shobu.session)
		if response.Code != http.StatusOK {
			return false
		}
		match = decode[api.Match](t, response)
		return true
	}, gamestest.Timeout, 5*time.Millisecond)
	assert.True(t, match.Guest.Computer)
	assert.False(t, match.Host.Computer)
	assert.False(t, match.Counted)
	assert.Nil(t, match.Host.RatingChange)

	profile := decode[api.PlayerProfile](t, site.call(http.MethodGet, "/api/players/Shobu", nil, mimi.session))
	assert.Len(t, profile.RecentMatches, 1)
	assert.Zero(t, profile.Wins+profile.Losses)

	// The computer has no page of its own.
	assert.Equal(t, http.StatusNotFound, site.call(http.MethodGet, "/api/players/Grim", nil, mimi.session).Code)
	assert.Equal(t, http.StatusNotFound, site.call(http.MethodGet, "/api/players/Grim/matches", nil, mimi.session).Code)
}

func TestAGameAgainstTheComputerIsRefusedThroughTheSite(t *testing.T) {
	site := newPlayServer(t)
	shobu := site.player(t, "Shobu")
	open, _ := site.formats(t)
	opponent := site.addOpponent(t, open.Id)
	path := "/api/computer-opponents/" + opponent.ID + "/games"
	body := api.JoinGameInput{DeckId: shobu.deckID}

	assert.Equal(t, http.StatusUnauthorized, site.call(http.MethodGet, "/api/computer-opponents", nil, nil).Code)
	assert.Equal(t, http.StatusUnauthorized, site.call(http.MethodPost, path, body, nil).Code, "signed-in players only")
	assert.Equal(t, http.StatusNotFound, site.call(http.MethodPost, "/api/computer-opponents/nobody/games", body, shobu.session).Code)
	assert.Equal(t, http.StatusBadRequest, site.call(http.MethodPost, path, api.JoinGameInput{DeckId: "nothing"}, shobu.session).Code)

	_, err := site.settings.Set(context.Background(), games.MaxComputerGames.Key, "0")
	require.NoError(t, err)
	full := site.call(http.MethodPost, path, body, shobu.session)
	require.Equal(t, http.StatusConflict, full.Code)
	assert.Equal(t, codeComputerGamesFull, decode[api.Error](t, full).Code)

	_, err = site.settings.Set(context.Background(), games.MaxComputerGames.Key, "1")
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, site.call(http.MethodPost, path, body, shobu.session).Code)
	busy := site.call(http.MethodPost, path, body, shobu.session)
	require.Equal(t, http.StatusConflict, busy.Code)
	assert.Equal(t, codeBusy, decode[api.Error](t, busy).Code)
}
