package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/httpapi/api"
)

func cardNames(page api.CardPage) []string {
	out := make([]string, 0, len(page.Cards))
	for _, card := range page.Cards {
		out = append(out, card.Name)
	}
	return out
}

func TestSearchCards(t *testing.T) {
	server := newTestServer(t, false)

	all := server.call(http.MethodGet, "/api/cards", nil, nil)
	require.Equal(t, http.StatusOK, all.Code, all.Body.String())
	page := decode[api.CardPage](t, all)
	assert.Equal(t, []string{"Bolshack Dragon", "Solar Ray"}, cardNames(page))
	assert.Equal(t, 2, page.Total)
	assert.Equal(t, 1, page.Page)
	assert.Equal(t, 60, page.PageSize)

	for query, want := range map[string][]string{
		"/api/cards?civilization=Light&civilization=Water": {"Solar Ray"},
		"/api/cards?type=Creature":                         {"Bolshack Dragon"},
		"/api/cards?set=DM-10":                             {"Solar Ray"},
		"/api/cards?minCost=3&maxCost=6":                   {"Bolshack Dragon"},
		"/api/cards?playable=true":                         {"Bolshack Dragon"},
		"/api/cards?q=SOLAR":                               {"Solar Ray"},
		"/api/cards?keyword=blocker&race=Human":            {"Bolshack Dragon", "Solar Ray"},
		"/api/cards?pageSize=1&page=2":                     {"Solar Ray"},
	} {
		response := server.call(http.MethodGet, query, nil, nil)
		require.Equal(t, http.StatusOK, response.Code, "%s: %s", query, response.Body.String())
		assert.Equal(t, want, cardNames(decode[api.CardPage](t, response)), query)
	}
}

func TestSearchCardsRefusesBadFilters(t *testing.T) {
	server := newTestServer(t, false)

	for query, field := range map[string]string{
		"/api/cards?pageSize=101":        "pageSize",
		"/api/cards?minCost=5&maxCost=2": "minCost",
		"/api/cards?q=" + longQuery(101): "q",
	} {
		response := server.call(http.MethodGet, query, nil, nil)
		assert.Equal(t, http.StatusBadRequest, response.Code, query)
		body := decode[api.Error](t, response)
		assert.Equal(t, "invalid_input", body.Code, query)
		require.NotNil(t, body.Field, query)
		assert.Equal(t, field, *body.Field, query)
	}

	malformed := server.call(http.MethodGet, "/api/cards?minCost=lots", nil, nil)
	assert.Equal(t, http.StatusBadRequest, malformed.Code)
	assert.Equal(t, "bad_request", decode[api.Error](t, malformed).Code)
}

func longQuery(n int) string {
	query := make([]byte, n)
	for i := range query {
		query[i] = 'x'
	}
	return string(query)
}

func TestGetCard(t *testing.T) {
	server := newTestServer(t, false)

	response := server.call(http.MethodGet, "/api/cards/0079", nil, nil)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	bolshack := decode[api.Card](t, response)
	assert.Equal(t, "Bolshack Dragon", bolshack.Name)
	assert.Equal(t, 6, bolshack.Cost)
	require.NotNil(t, bolshack.Power)
	assert.Equal(t, 6000, *bolshack.Power)
	assert.Equal(t, "6000+", bolshack.PowerText)
	assert.True(t, bolshack.Playable)
	require.Len(t, bolshack.Abilities, 1)
	require.NotNil(t, bolshack.Abilities[0].Reminder)
	assert.Equal(t, "(This creature breaks 2 shields.)", *bolshack.Abilities[0].Reminder)
	require.NotNil(t, bolshack.ImageUrl)
	assert.Equal(t, "/api/cards/images/0079", *bolshack.ImageUrl)

	solarResponse := server.call(http.MethodGet, "/api/cards/0029", nil, nil)
	require.Equal(t, http.StatusOK, solarResponse.Code)
	solar := decode[api.Card](t, solarResponse)
	assert.Nil(t, solar.Power, "spells have no power")
	assert.False(t, solar.Playable)
	require.Len(t, solar.Printings, 2)
	assert.Equal(t, "DM-01", solar.Printings[0].Set)
	assert.NotNil(t, solar.Printings[0].ImageUrl)
	assert.Nil(t, solar.Printings[1].ImageUrl, "the reprint has no picture")

	missing := server.call(http.MethodGet, "/api/cards/9999", nil, nil)
	assert.Equal(t, http.StatusNotFound, missing.Code)
	assert.Equal(t, "not_found", decode[api.Error](t, missing).Code)
}

func TestGetCardFilters(t *testing.T) {
	server := newTestServer(t, false)

	response := server.call(http.MethodGet, "/api/cards/filters", nil, nil)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	filters := decode[api.CardFilters](t, response)

	assert.Equal(t, []string{"Fire", "Light"}, filters.Civilizations)
	assert.Equal(t, []string{"Creature", "Spell"}, filters.Types)
	require.Len(t, filters.Sets, 2)
	assert.Equal(t, "DM-01", filters.Sets[0].Code)
	require.NotNil(t, filters.Sets[0].IconUrl)
	assert.Equal(t, "/api/cards/sets/DM-01/icon", *filters.Sets[0].IconUrl)
	assert.Nil(t, filters.Sets[1].IconUrl, "DM-10 has no icon")
}

func TestGetCardImage(t *testing.T) {
	server := newTestServer(t, false)

	byPrinting := server.call(http.MethodGet, "/api/cards/images/0079", nil, nil)
	require.Equal(t, http.StatusOK, byPrinting.Code)
	assert.Equal(t, "image/webp", byPrinting.Header().Get("Content-Type"))
	assert.Equal(t, "public, max-age=86400", byPrinting.Header().Get("Cache-Control"))
	assert.Equal(t, "picture 0079", byPrinting.Body.String())
	lastModified := byPrinting.Header().Get("Last-Modified")
	require.NotEmpty(t, lastModified)

	byEngine := server.call(http.MethodGet, "/api/cards/images/engine-bolshack", nil, nil)
	require.Equal(t, http.StatusOK, byEngine.Code, "the duel board asks by engine id")
	assert.Equal(t, "picture 0079", byEngine.Body.String())

	request := httptest.NewRequest(http.MethodGet, "/api/cards/images/0079", nil)
	request.Header.Set("If-Modified-Since", lastModified)
	conditional := httptest.NewRecorder()
	server.handler.ServeHTTP(conditional, request)
	assert.Equal(t, http.StatusNotModified, conditional.Code, "an unchanged picture is not sent again")
	assert.Empty(t, conditional.Body.String())

	request = httptest.NewRequest(http.MethodGet, "/api/cards/images/0079", nil)
	request.Header.Set("If-Modified-Since", time.Unix(0, 0).UTC().Format(http.TimeFormat))
	stale := httptest.NewRecorder()
	server.handler.ServeHTTP(stale, request)
	assert.Equal(t, http.StatusOK, stale.Code, "an older copy gets the picture")

	for _, key := range []string{"0695", "nothing", "..%2F..%2Fdata%2Fcards.json"} {
		missing := server.call(http.MethodGet, "/api/cards/images/"+key, nil, nil)
		assert.Equal(t, http.StatusNotFound, missing.Code, key)
		assert.Equal(t, "not_found", decode[api.Error](t, missing).Code, key)
	}
}

func TestGetCardSetIcon(t *testing.T) {
	server := newTestServer(t, false)

	icon := server.call(http.MethodGet, "/api/cards/sets/DM-01/icon", nil, nil)
	require.Equal(t, http.StatusOK, icon.Code)
	assert.Equal(t, "image/png", icon.Header().Get("Content-Type"))
	assert.Equal(t, "icon dm-01", icon.Body.String())

	for _, path := range []string{"/api/cards/sets/DM-10/icon", "/api/cards/sets/DM-99/icon"} {
		assert.Equal(t, http.StatusNotFound, server.call(http.MethodGet, path, nil, nil).Code, path)
	}
}
