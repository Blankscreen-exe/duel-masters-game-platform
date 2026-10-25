package httpapi

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/httpapi/api"
)

// addStarterDeck adds a starter deck of four copies of each of the play
// server's cards, as the starter decks' seed adds its decks.
func (s playServer) addStarterDeck(t *testing.T, id, name string) {
	t.Helper()

	now := s.db.NowFunc()
	require.NoError(t, s.db.Table("starter_decks").Create(map[string]any{
		"id": id, "name": name, "sort_order": 1, "enabled": true, "created_at": now, "updated_at": now,
	}).Error)
	for _, cardID := range s.cardIDs {
		require.NoError(t, s.db.Table("starter_deck_cards").Create(map[string]any{
			"starter_deck_id": id, "card_id": cardID, "count": 4,
		}).Error)
	}
}

func TestPlayersPlayAndCopyStarterDecks(t *testing.T) {
	site := newPlayServer(t)
	site.addStarterDeck(t, "starter-fire", "Fire: Bolshack Rush")
	shobu := site.player(t, "Shobu")
	kokujo := site.player(t, "Kokujo")

	assert.Equal(t, http.StatusUnauthorized, site.call(http.MethodGet, "/api/starter-decks", nil, nil).Code)
	response := site.call(http.MethodGet, "/api/starter-decks", nil, shobu.session)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	starters := decode[[]api.DeckSummary](t, response)
	require.Len(t, starters, 1)
	assert.Equal(t, "Fire: Bolshack Rush", starters[0].Name)
	assert.Equal(t, 40, starters[0].CardCount)
	open, _ := site.formats(t)
	assert.Contains(t, starters[0].LegalFormatIds, open.Id)

	// Both players play the same starter deck.
	game := site.openGame(t, shobu, api.OpenGameInput{DeckId: "starter-fire", Listed: true})
	response = site.call(http.MethodPost, "/api/lobby/games/"+game.Id+"/join", api.JoinGameInput{DeckId: "starter-fire"}, kokujo.session)
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())
	started := decode[api.GameStarted](t, response)
	running := decode[api.RunningGame](t, site.call(http.MethodGet, "/api/games/"+started.GameId, nil, shobu.session))
	assert.Equal(t, "Fire: Bolshack Rush", running.Host.DeckName)
	assert.Equal(t, "Fire: Bolshack Rush", running.Guest.DeckName)

	// A copy is the player's own, to change.
	response = site.call(http.MethodPost, "/api/starter-decks/starter-fire/copy", nil, shobu.session)
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())
	copied := decode[api.Deck](t, response)
	assert.NotEqual(t, "starter-fire", copied.Id)
	assert.Equal(t, "Fire: Bolshack Rush", copied.Name)
	assert.Equal(t, 40, copied.CardCount)
	assert.Equal(t, http.StatusOK, site.call(http.MethodGet, "/api/decks/"+copied.Id, nil, shobu.session).Code)
	assert.Equal(t, http.StatusNotFound, site.call(http.MethodGet, "/api/decks/starter-fire", nil, shobu.session).Code,
		"a starter deck is nobody's own deck")

	assert.Equal(t, http.StatusNotFound, site.call(http.MethodPost, "/api/starter-decks/"+copied.Id+"/copy", nil, shobu.session).Code)
	assert.Equal(t, http.StatusUnauthorized, site.call(http.MethodPost, "/api/starter-decks/starter-fire/copy", nil, nil).Code)
}
