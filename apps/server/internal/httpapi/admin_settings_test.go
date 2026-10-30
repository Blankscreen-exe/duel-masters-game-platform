package httpapi

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/games"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/httpapi/api"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/lobby"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/opponents"
)

const settingsPath = "/api/admin/settings"

func TestSettingsAreForAdmins(t *testing.T) {
	site := newTestServer(t, false)
	site.signUp(t, "Boss")
	player := site.signUp(t, "Shobu")
	body := api.AdminSettingInput{Value: "5"}
	path := settingsPath + "/" + games.MaxComputerGames.Key

	for _, request := range []struct {
		method, path string
		body         any
	}{
		{http.MethodGet, settingsPath, nil},
		{http.MethodPut, path, body},
		{http.MethodDelete, path, nil},
	} {
		assert.Equal(t, http.StatusUnauthorized, site.call(request.method, request.path, request.body, nil).Code, request.method)
		assert.Equal(t, http.StatusForbidden, site.call(request.method, request.path, request.body, player).Code, request.method)
	}
	assert.EqualValues(t, games.MaxComputerGames.Default, site.settings.Number(games.MaxComputerGames), "nothing changed")
}

func TestAdminsSeeEverySetting(t *testing.T) {
	site := newTestServer(t, false)
	boss := site.signUp(t, "Boss")

	response := site.call(http.MethodGet, settingsPath, nil, boss)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())

	byKey := map[string]api.AdminSetting{}
	for _, setting := range decode[[]api.AdminSetting](t, response) {
		byKey[setting.Key] = setting
	}
	assert.Len(t, byKey, len(site.settings.List()))

	pause, ok := byKey[opponents.Pause.Key]
	require.True(t, ok, "the computer's pause is among them")
	assert.Equal(t, api.AdminSettingKindNumber, pause.Kind)
	assert.Equal(t, "Computer opponents", pause.Group)
	assert.Equal(t, "800", pause.Default)
	assert.Equal(t, "800", pause.Value)
	assert.True(t, pause.IsDefault)
	require.NotNil(t, pause.Max)
	assert.EqualValues(t, 5000, *pause.Max)
	assert.Nil(t, pause.UpdatedAt)

	paused, ok := byKey[lobby.NewGamesOpen.Key]
	require.True(t, ok)
	assert.Equal(t, api.AdminSettingKindBoolean, paused.Kind)
	assert.Nil(t, paused.Max)
}

func TestAnAdminChangesASettingAndPutsItBack(t *testing.T) {
	site := newTestServer(t, false)
	boss := site.signUp(t, "Boss")
	path := settingsPath + "/" + games.MaxComputerGames.Key

	changed := site.call(http.MethodPut, path, api.AdminSettingInput{Value: "5"}, boss)
	require.Equal(t, http.StatusOK, changed.Code, changed.Body.String())
	setting := decode[api.AdminSetting](t, changed)
	assert.Equal(t, "5", setting.Value)
	assert.False(t, setting.IsDefault)
	assert.NotNil(t, setting.UpdatedAt)
	assert.EqualValues(t, 5, site.settings.Number(games.MaxComputerGames), "it applies at once")

	log := site.adminLog(t, boss)
	require.Len(t, log, 1)
	assert.Equal(t, api.AdminLogActionSettingChanged, log[0].Action)
	assert.Equal(t, games.MaxComputerGames.Key, log[0].Subject)
	assert.Equal(t, "From 100 to 5", log[0].Details)

	// The same value again changes nothing to record.
	require.Equal(t, http.StatusOK, site.call(http.MethodPut, path, api.AdminSettingInput{Value: "5"}, boss).Code)
	assert.Len(t, site.adminLog(t, boss), 1)

	reset := site.call(http.MethodDelete, path, nil, boss)
	require.Equal(t, http.StatusOK, reset.Code, reset.Body.String())
	assert.True(t, decode[api.AdminSetting](t, reset).IsDefault)
	assert.EqualValues(t, 100, site.settings.Number(games.MaxComputerGames))

	log = site.adminLog(t, boss)
	require.Len(t, log, 2)
	assert.Equal(t, api.AdminLogActionSettingReset, log[0].Action)
	assert.Equal(t, "From 5 back to 100", log[0].Details)

	// At its default already, there is nothing to record.
	require.Equal(t, http.StatusOK, site.call(http.MethodDelete, path, nil, boss).Code)
	assert.Len(t, site.adminLog(t, boss), 2)
}

func TestASettingRefusesWhatItCannotHold(t *testing.T) {
	site := newTestServer(t, false)
	boss := site.signUp(t, "Boss")
	path := settingsPath + "/" + games.MaxComputerGames.Key

	for _, value := range []string{"100000", "-1", "lots", ""} {
		refused := site.call(http.MethodPut, path, api.AdminSettingInput{Value: value}, boss)
		require.Equal(t, http.StatusBadRequest, refused.Code, value)
		answer := decode[api.Error](t, refused)
		assert.Equal(t, codeInvalidInput, answer.Code)
		require.NotNil(t, answer.Field)
		assert.Equal(t, "value", *answer.Field)
		assert.Contains(t, answer.Message, "This value ")
	}
	assert.EqualValues(t, 100, site.settings.Number(games.MaxComputerGames))
	assert.Empty(t, site.adminLog(t, boss))

	assert.Equal(t, http.StatusNotFound, site.call(http.MethodPut, settingsPath+"/no.such.setting", api.AdminSettingInput{Value: "1"}, boss).Code)
	assert.Equal(t, http.StatusNotFound, site.call(http.MethodDelete, settingsPath+"/no.such.setting", nil, boss).Code)
}

func TestPausingNewGamesShowsInTheLobbyAtOnce(t *testing.T) {
	site := newPlayServer(t)
	boss := site.player(t, "Boss")

	require.Equal(t, http.StatusOK, site.call(http.MethodPut, settingsPath+"/"+lobby.NewGamesOpen.Key, api.AdminSettingInput{Value: "false"}, boss.session).Code)

	liveMessageWhere(t, boss.tab, "lobby", func(message map[string]any) bool {
		return message["lobby"].(map[string]any)["paused"] == true
	})
}
