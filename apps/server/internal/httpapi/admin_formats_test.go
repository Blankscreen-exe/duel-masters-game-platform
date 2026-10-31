package httpapi

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/httpapi/api"
)

const adminFormatsPath = "/api/admin/formats"

func formatBody(name string) api.FormatInput {
	return api.FormatInput{
		Name:        name,
		Description: "Old cards only.",
		Sets:        []string{"DM-01"},
		Limits:      []api.FormatCardLimit{{CardId: "0079", MaxCopies: 1}},
	}
}

func TestManagingFormatsIsForAdmins(t *testing.T) {
	server := newTestServer(t, false)
	server.signUp(t, "Boss")
	player := server.signUp(t, "Shobu")
	body := formatBody("Legends")

	for _, request := range []struct {
		method, path string
		body         any
	}{
		{http.MethodGet, adminFormatsPath, nil},
		{http.MethodPost, adminFormatsPath, body},
		{http.MethodPut, adminFormatsPath + "/order", api.FormatOrder{FormatIds: []string{}}},
		{http.MethodGet, adminFormatsPath + "/any", nil},
		{http.MethodPut, adminFormatsPath + "/any", body},
		{http.MethodPut, adminFormatsPath + "/any/retired", api.FormatRetired{Retired: true}},
	} {
		assert.Equal(t, http.StatusUnauthorized, server.call(request.method, request.path, request.body, nil).Code, "%s %s", request.method, request.path)
		assert.Equal(t, http.StatusForbidden, server.call(request.method, request.path, request.body, player).Code, "%s %s", request.method, request.path)
	}
}

func TestAnAdminManagesAFormat(t *testing.T) {
	server := newTestServer(t, false)
	boss := server.signUp(t, "Boss")
	open, classic := server.formats(t)

	// Added, with its limited card by name, and players can choose it at once.
	created := server.call(http.MethodPost, adminFormatsPath, formatBody("Legends"), boss)
	require.Equal(t, http.StatusCreated, created.Code, created.Body.String())
	legends := decode[api.AdminFormat](t, created)
	assert.Equal(t, "Legends", legends.Name)
	assert.Equal(t, []string{"DM-01"}, legends.Sets)
	assert.False(t, legends.Retired)
	assert.Empty(t, legends.Opponents)
	require.Len(t, legends.Limits, 1)
	assert.Equal(t, 1, legends.Limits[0].MaxCopies)
	require.NotNil(t, legends.Limits[0].Card)
	assert.Equal(t, "Bolshack Dragon", legends.Limits[0].Card.Name)

	public := decode[[]api.Format](t, server.call(http.MethodGet, "/api/formats", nil, nil))
	require.Len(t, public, 3)
	assert.Equal(t, legends.Id, public[2].Id)

	// Refusals.
	taken := server.call(http.MethodPost, adminFormatsPath, formatBody("legends"), boss)
	assert.Equal(t, http.StatusConflict, taken.Code)
	assert.Equal(t, "format_name_taken", decode[api.Error](t, taken).Code)

	bad := formatBody("Other")
	bad.Sets = []string{}
	refused := server.call(http.MethodPost, adminFormatsPath, bad, boss)
	assert.Equal(t, http.StatusBadRequest, refused.Code)
	body := decode[api.Error](t, refused)
	assert.Equal(t, "invalid_input", body.Code)
	require.NotNil(t, body.Field)
	assert.Equal(t, "sets", *body.Field)

	// Changed.
	change := formatBody("Masters")
	change.AllSets = true
	change.Limits = []api.FormatCardLimit{}
	updated := server.call(http.MethodPut, adminFormatsPath+"/"+legends.Id, change, boss)
	require.Equal(t, http.StatusOK, updated.Code, updated.Body.String())
	masters := decode[api.AdminFormat](t, updated)
	assert.Equal(t, "Masters", masters.Name)
	assert.True(t, masters.AllSets)
	assert.Empty(t, masters.Sets)
	assert.Empty(t, masters.Limits)

	got := server.call(http.MethodGet, adminFormatsPath+"/"+legends.Id, nil, boss)
	require.Equal(t, http.StatusOK, got.Code)
	assert.Equal(t, masters, decode[api.AdminFormat](t, got))

	// Ordered.
	ordered := server.call(http.MethodPut, adminFormatsPath+"/order", api.FormatOrder{FormatIds: []string{legends.Id, classic.Id, open.Id}}, boss)
	require.Equal(t, http.StatusOK, ordered.Code, ordered.Body.String())
	assert.Equal(t, legends.Id, decode[[]api.AdminFormat](t, ordered)[0].Id)
	assert.Equal(t, http.StatusBadRequest, server.call(http.MethodPut, adminFormatsPath+"/order", api.FormatOrder{FormatIds: []string{open.Id}}, boss).Code)

	// Retired: gone for players, still there for admins; and brought back.
	retired := server.call(http.MethodPut, adminFormatsPath+"/"+legends.Id+"/retired", api.FormatRetired{Retired: true}, boss)
	require.Equal(t, http.StatusOK, retired.Code, retired.Body.String())
	assert.True(t, decode[api.AdminFormat](t, retired).Retired)
	assert.Len(t, decode[[]api.Format](t, server.call(http.MethodGet, "/api/formats", nil, nil)), 2)
	assert.Len(t, decode[[]api.AdminFormat](t, server.call(http.MethodGet, adminFormatsPath, nil, boss)), 3)

	// The last one left stays.
	require.Equal(t, http.StatusOK, server.call(http.MethodPut, adminFormatsPath+"/"+classic.Id+"/retired", api.FormatRetired{Retired: true}, boss).Code)
	last := server.call(http.MethodPut, adminFormatsPath+"/"+open.Id+"/retired", api.FormatRetired{Retired: true}, boss)
	assert.Equal(t, http.StatusConflict, last.Code)
	assert.Equal(t, "last_format", decode[api.Error](t, last).Code)

	back := server.call(http.MethodPut, adminFormatsPath+"/"+legends.Id+"/retired", api.FormatRetired{Retired: false}, boss)
	require.Equal(t, http.StatusOK, back.Code)
	assert.False(t, decode[api.AdminFormat](t, back).Retired)

	// A format that is not there.
	assert.Equal(t, http.StatusNotFound, server.call(http.MethodGet, adminFormatsPath+"/nothing", nil, boss).Code)
	assert.Equal(t, http.StatusNotFound, server.call(http.MethodPut, adminFormatsPath+"/nothing", formatBody("Nope"), boss).Code)
	assert.Equal(t, http.StatusNotFound, server.call(http.MethodPut, adminFormatsPath+"/nothing/retired", api.FormatRetired{Retired: true}, boss).Code)

	// Everything is in the log.
	log := decode[api.AdminLogPage](t, server.call(http.MethodGet, "/api/admin/log", nil, boss))
	actions := make([]api.AdminLogAction, 0, len(log.Entries))
	for _, entry := range log.Entries {
		actions = append(actions, entry.Action)
	}
	assert.Equal(t, []api.AdminLogAction{
		api.AdminLogActionFormatRestored, api.AdminLogActionFormatRetired, api.AdminLogActionFormatRetired, api.AdminLogActionFormatsReordered, api.AdminLogActionFormatChanged, api.AdminLogActionFormatAdded,
	}, actions)
}

func TestAnOfferedOpponentsFormatSaysSoAndDropsItWhenRetired(t *testing.T) {
	site := newPlayServer(t)
	boss := site.signUp(t, "Boss")
	shobu := site.player(t, "Shobu")
	open, _ := site.formats(t)

	created := site.call(http.MethodPost, opponentsPath, site.opponentBody(t, "Grim"), boss)
	require.Equal(t, http.StatusCreated, created.Code, created.Body.String())
	grim := decode[api.AdminComputerOpponent](t, created)
	require.Equal(t, http.StatusOK, site.call(http.MethodPut, opponentsPath+"/"+grim.Id+"/offered", api.ComputerOpponentOffered{Offered: true}, boss).Code)

	format := decode[api.AdminFormat](t, site.call(http.MethodGet, adminFormatsPath+"/"+open.Id, nil, boss))
	assert.Equal(t, []string{"Grim"}, format.Opponents)
	assert.Len(t, decode[[]api.ComputerOpponent](t, site.call(http.MethodGet, "/api/computer-opponents", nil, shobu.session)), 1)

	require.Equal(t, http.StatusOK, site.call(http.MethodPut, adminFormatsPath+"/"+open.Id+"/retired", api.FormatRetired{Retired: true}, boss).Code)

	assert.Empty(t, decode[[]api.ComputerOpponent](t, site.call(http.MethodGet, "/api/computer-opponents", nil, shobu.session)), "players are not offered it")
	shown := decode[api.AdminComputerOpponent](t, site.call(http.MethodGet, opponentsPath+"/"+grim.Id, nil, boss))
	require.NotEmpty(t, shown.Problems)
	assert.Equal(t, api.DeckProblemCodeFormatRetired, shown.Problems[0].Code, "the admin panel says why")
	assert.Equal(t, open.Name, shown.FormatName, "it keeps its format")
}
