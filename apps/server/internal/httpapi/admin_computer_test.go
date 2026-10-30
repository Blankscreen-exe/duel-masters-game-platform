package httpapi

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/httpapi/api"
)

const opponentsPath = "/api/admin/computer-opponents"

// opponentBody is a computer opponent with a whole deck.
func (s playServer) opponentBody(t *testing.T, name string) api.ComputerOpponentInput {
	t.Helper()

	open, _ := s.formats(t)
	return api.ComputerOpponentInput{
		Name:             name,
		Description:      "Hits hard.",
		Style:            "rush",
		Strength:         api.ComputerOpponentInputStrengthEasy,
		FormatId:         open.Id,
		MaxCivilizations: 0,
		CardTypes:        []string{"Creature", "Evolution Creature", "Spell"},
		Cards:            fullDeck(s.cardIDs),
	}
}

func TestManagingComputerOpponentsIsForAdmins(t *testing.T) {
	site := newPlayServer(t)
	site.signUp(t, "Boss")
	player := site.signUp(t, "Shobu")
	body := site.opponentBody(t, "Grim")

	for _, request := range []struct {
		method, path string
		body         any
	}{
		{http.MethodGet, opponentsPath, nil},
		{http.MethodPost, opponentsPath, body},
		{http.MethodGet, opponentsPath + "/options", nil},
		{http.MethodGet, opponentsPath + "/any", nil},
		{http.MethodPut, opponentsPath + "/any", body},
		{http.MethodPut, opponentsPath + "/any/offered", api.ComputerOpponentOffered{Offered: true}},
	} {
		assert.Equal(t, http.StatusUnauthorized, site.call(request.method, request.path, request.body, nil).Code, "%s %s", request.method, request.path)
		assert.Equal(t, http.StatusForbidden, site.call(request.method, request.path, request.body, player).Code, "%s %s", request.method, request.path)
	}
}

func TestAnAdminMakesAnOpponentAndPlayersGetToPlayIt(t *testing.T) {
	site := newPlayServer(t)
	boss := site.signUp(t, "Boss")
	shobu := site.player(t, "Shobu")

	options := decode[api.ComputerOpponentOptions](t, site.call(http.MethodGet, opponentsPath+"/options", nil, boss))
	assert.Contains(t, options.CardTypes, "Castle")
	require.NotEmpty(t, options.Styles)
	labels := map[string]string{}
	for _, style := range options.Styles {
		assert.NotEmpty(t, style.Label, style.Name)
		labels[style.Name] = style.Label
	}
	assert.Equal(t, "Rapid summon", labels["rush"])

	// It starts retired, unfinished, and players do not see it.
	unfinished := site.opponentBody(t, "Grim")
	unfinished.Cards = unfinished.Cards[:2]
	created := site.call(http.MethodPost, opponentsPath, unfinished, boss)
	require.Equal(t, http.StatusCreated, created.Code, created.Body.String())
	opponent := decode[api.AdminComputerOpponent](t, created)
	assert.False(t, opponent.Offered)
	assert.Equal(t, 8, opponent.CardCount)
	require.Len(t, opponent.Cards, 2)
	assert.NotNil(t, opponent.Cards[0].Card, "shown with its cards")
	require.NotEmpty(t, opponent.Problems)
	assert.Equal(t, api.DeckProblemCodeDeckSize, opponent.Problems[0].Code)
	assert.Equal(t, api.AdminComputerOpponentStrengthEasy, opponent.Strength)

	offers := decode[[]api.ComputerOpponent](t, site.call(http.MethodGet, "/api/computer-opponents", nil, shobu.session))
	assert.Empty(t, offers)

	// It cannot be offered yet.
	path := opponentsPath + "/" + opponent.Id
	refused := site.call(http.MethodPut, path+"/offered", api.ComputerOpponentOffered{Offered: true}, boss)
	require.Equal(t, http.StatusBadRequest, refused.Code)
	answer := decode[api.Error](t, refused)
	assert.Equal(t, codeDeckNotLegal, answer.Code)
	require.NotNil(t, answer.Problems)
	assert.Contains(t, answer.Message, "Grim's deck cannot be played")

	// Finished, it can.
	saved := site.call(http.MethodPut, path, site.opponentBody(t, "Grim"), boss)
	require.Equal(t, http.StatusOK, saved.Code, saved.Body.String())
	assert.Empty(t, decode[api.AdminComputerOpponent](t, saved).Problems)

	offered := site.call(http.MethodPut, path+"/offered", api.ComputerOpponentOffered{Offered: true}, boss)
	require.Equal(t, http.StatusOK, offered.Code, offered.Body.String())
	assert.True(t, decode[api.AdminComputerOpponent](t, offered).Offered)

	offers = decode[[]api.ComputerOpponent](t, site.call(http.MethodGet, "/api/computer-opponents", nil, shobu.session))
	require.Len(t, offers, 1)
	assert.Equal(t, "Grim", offers[0].Name)
	assert.Equal(t, http.StatusCreated, site.call(http.MethodPost, "/api/computer-opponents/"+opponent.Id+"/games", api.JoinGameInput{DeckId: shobu.deckID}, shobu.session).Code)

	// It shows in the admin panel's list, and retiring takes it from the players.
	list := decode[[]api.AdminComputerOpponent](t, site.call(http.MethodGet, opponentsPath, nil, boss))
	require.Len(t, list, 1)
	assert.Equal(t, opponent.Id, list[0].Id)

	require.Equal(t, http.StatusOK, site.call(http.MethodPut, path+"/offered", api.ComputerOpponentOffered{Offered: false}, boss).Code)
	offers = decode[[]api.ComputerOpponent](t, site.call(http.MethodGet, "/api/computer-opponents", nil, shobu.session))
	assert.Empty(t, offers)
	retired := decode[api.AdminComputerOpponent](t, site.call(http.MethodGet, path, nil, boss))
	assert.False(t, retired.Offered, "an admin still sees it")

	log := site.adminLog(t, boss)
	actions := make([]api.AdminLogAction, 0, len(log))
	for _, entry := range log {
		actions = append(actions, entry.Action)
	}
	assert.Equal(t, []api.AdminLogAction{
		api.AdminLogActionComputerOpponentRetired,
		api.AdminLogActionComputerOpponentOffered,
		api.AdminLogActionComputerOpponentChanged,
		api.AdminLogActionComputerOpponentAdded,
	}, actions)
}

func TestManagingAnOpponentIsRefusedWithReasons(t *testing.T) {
	site := newPlayServer(t)
	boss := site.signUp(t, "Boss")

	created := site.call(http.MethodPost, opponentsPath, site.opponentBody(t, "Grim"), boss)
	require.Equal(t, http.StatusCreated, created.Code, created.Body.String())

	badName := site.call(http.MethodPost, opponentsPath, site.opponentBody(t, "a"), boss)
	require.Equal(t, http.StatusBadRequest, badName.Code)
	answer := decode[api.Error](t, badName)
	require.NotNil(t, answer.Field)
	assert.Equal(t, "name", *answer.Field, "the request's name for it")

	taken := site.call(http.MethodPost, opponentsPath, site.opponentBody(t, "boss"), boss)
	require.Equal(t, http.StatusConflict, taken.Code)
	assert.Equal(t, codeUsernameTaken, decode[api.Error](t, taken).Code)

	assert.Equal(t, http.StatusNotFound, site.call(http.MethodGet, opponentsPath+"/nothing", nil, boss).Code)
	assert.Equal(t, http.StatusNotFound, site.call(http.MethodPut, opponentsPath+"/nothing", site.opponentBody(t, "Other"), boss).Code)
	assert.Equal(t, http.StatusNotFound, site.call(http.MethodPut, opponentsPath+"/nothing/offered", api.ComputerOpponentOffered{Offered: true}, boss).Code)
}
