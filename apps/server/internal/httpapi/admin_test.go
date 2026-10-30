package httpapi

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/httpapi/api"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/realtime"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/store/storetest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/users"
)

// adminUser finds a player's account as the admin panel lists it.
func (s testServer) adminUser(t *testing.T, admin *http.Cookie, name string) api.AdminUser {
	t.Helper()

	response := s.call(http.MethodGet, "/api/admin/users?search="+name, nil, admin)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	for _, user := range decode[api.AdminUserPage](t, response).Users {
		if user.Username == name {
			return user
		}
	}
	t.Fatalf("%s is not listed", name)
	return api.AdminUser{}
}

func (s testServer) adminLog(t *testing.T, admin *http.Cookie) []api.AdminLogEntry {
	t.Helper()

	response := s.call(http.MethodGet, "/api/admin/log", nil, admin)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	return decode[api.AdminLogPage](t, response).Entries
}

func TestTheAdminPanelIsForAdmins(t *testing.T) {
	site := newTestServer(t, false)
	site.signUp(t, "Boss")
	player := site.signUp(t, "Shobu")

	requests := []struct {
		method, path string
		body         any
	}{
		{http.MethodGet, "/api/admin/users", nil},
		{http.MethodGet, "/api/admin/log", nil},
		{http.MethodPost, "/api/admin/users/someone/ban", api.BanInput{Reason: "No"}},
		{http.MethodDelete, "/api/admin/users/someone/ban", nil},
		{http.MethodPost, "/api/admin/users/someone/mute", api.MuteInput{}},
		{http.MethodDelete, "/api/admin/users/someone/mute", nil},
		{http.MethodPut, "/api/admin/users/someone/role", api.RoleInput{Role: api.RoleAdmin}},
		{http.MethodPut, "/api/admin/users/someone/password", api.NewPassword{Password: "a long new password"}},
	}
	for _, request := range requests {
		visitor := site.call(request.method, request.path, request.body, nil)
		assert.Equal(t, http.StatusUnauthorized, visitor.Code, "%s %s", request.method, request.path)

		refused := site.call(request.method, request.path, request.body, player)
		assert.Equal(t, http.StatusForbidden, refused.Code, "%s %s", request.method, request.path)
		assert.Equal(t, codeForbidden, decode[api.Error](t, refused).Code)
	}
}

func TestAdminsSearchTheAccounts(t *testing.T) {
	site := newTestServer(t, false)
	boss := site.signUp(t, "Boss")
	site.signUp(t, "Shobu")
	site.signUp(t, "Kokujo")

	response := site.call(http.MethodGet, "/api/admin/users", nil, boss)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	page := decode[api.AdminUserPage](t, response)
	assert.Equal(t, 3, page.Total)
	require.Len(t, page.Users, 3)
	assert.Equal(t, []string{"Boss", "Kokujo", "Shobu"}, []string{page.Users[0].Username, page.Users[1].Username, page.Users[2].Username})
	assert.Equal(t, api.RoleAdmin, page.Users[0].Role)
	assert.Equal(t, bronze(1000, api.RankDivisionII, "Bronze II"), page.Users[1].Rank)
	assert.False(t, page.Users[1].Online)
	assert.Nil(t, page.Users[1].Ban)

	admins := decode[api.AdminUserPage](t, site.call(http.MethodGet, "/api/admin/users?status=admin", nil, boss))
	assert.Equal(t, 1, admins.Total)

	assert.Equal(t, http.StatusBadRequest, site.call(http.MethodGet, "/api/admin/users?status=sleeping", nil, boss).Code)
	assert.Equal(t, http.StatusBadRequest, site.call(http.MethodGet, "/api/admin/users?pageSize=101", nil, boss).Code)
}

func TestABanPutsAPlayerOutOfTheSite(t *testing.T) {
	site := newPlayServer(t)
	boss := site.player(t, "Boss")
	shobu := site.player(t, "Shobu")
	kokujo := site.player(t, "Kokujo")

	open := site.openGame(t, shobu, api.OpenGameInput{Listed: true})
	response := site.call(http.MethodPost, "/api/lobby/games/"+open.Id+"/join", api.JoinGameInput{DeckId: kokujo.deckID}, kokujo.session)
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())
	host, guest := site.play(t, shobu, kokujo, decode[api.GameStarted](t, response).GameId)

	target := site.adminUser(t, boss.session, "Shobu")
	assert.True(t, target.Online)
	days := 7 * 24 * 60
	response = site.call(http.MethodPost, "/api/admin/users/"+target.Id+"/ban", api.BanInput{DurationMinutes: &days, Reason: "Spamming the chat"}, boss.session)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	banned := decode[api.AdminUser](t, response)
	require.NotNil(t, banned.Ban)
	assert.Equal(t, "Spamming the chat", banned.Ban.Reason)
	require.NotNil(t, banned.Ban.Until)
	assert.WithinDuration(t, site.clock.Now().Add(7*24*time.Hour), *banned.Ban.Until, time.Second)

	// The banned player's pages are told they are logged out, and their
	// game stops for both players without counting.
	require.NoError(t, shobu.tab.SetReadDeadline(time.Now().Add(5*time.Second)))
	for {
		if _, _, err := shobu.tab.ReadMessage(); err != nil {
			assert.True(t, websocket.IsCloseError(err, realtime.CloseSignedOut), "got %v", err)
			break
		}
	}
	for _, player := range []*gameConn{host, guest} {
		assert.Equal(t, "aborted", gameMessage(t, player, "duel_finished")["ending"])
	}
	assert.Equal(t, http.StatusUnauthorized, site.call(http.MethodGet, "/api/auth/me", nil, shobu.session).Code)

	// Logging in shows the reason and when the ban ends.
	response = site.call(http.MethodPost, "/api/auth/login", api.Credentials{Username: "Shobu", Password: testPassword}, nil)
	require.Equal(t, http.StatusForbidden, response.Code, response.Body.String())
	refusal := decode[api.Error](t, response)
	assert.Equal(t, codeBanned, refusal.Code)
	require.NotNil(t, refusal.Reason)
	assert.Equal(t, "Spamming the chat", *refusal.Reason)
	require.NotNil(t, refusal.Until)
	assert.Equal(t, http.StatusUnauthorized, site.call(http.MethodPost, "/api/auth/login", api.Credentials{Username: "Shobu", Password: "not the password"}, nil).Code,
		"a wrong password says nothing of the ban")

	banList := decode[api.AdminUserPage](t, site.call(http.MethodGet, "/api/admin/users?status=banned", nil, boss.session))
	require.Len(t, banList.Users, 1)
	assert.Equal(t, "Shobu", banList.Users[0].Username)

	response = site.call(http.MethodDelete, "/api/admin/users/"+target.Id+"/ban", nil, boss.session)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	assert.Nil(t, decode[api.AdminUser](t, response).Ban)
	assert.Equal(t, http.StatusOK, site.call(http.MethodPost, "/api/auth/login", api.Credentials{Username: "Shobu", Password: testPassword}, nil).Code)

	log := site.adminLog(t, boss.session)
	require.Len(t, log, 2)
	assert.Equal(t, api.AdminLogActionUserUnbanned, log[0].Action)
	assert.Equal(t, api.AdminLogActionUserBanned, log[1].Action)
	assert.Equal(t, "Boss", log[1].AdminName)
	assert.Equal(t, "Shobu", log[1].Subject)
	assert.Equal(t, "For 7 days: Spamming the chat", log[1].Details)
}

func TestAMutedPlayerReadsButCannotPost(t *testing.T) {
	site := newPlayServer(t)
	boss := site.player(t, "Boss")
	shobu := site.player(t, "Shobu")
	target := site.adminUser(t, boss.session, "Shobu")

	hour := 60
	response := site.call(http.MethodPost, "/api/admin/users/"+target.Id+"/mute", api.MuteInput{DurationMinutes: &hour}, boss.session)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	require.NotNil(t, decode[api.AdminUser](t, response).Mute)
	liveMessage(t, shobu.tab, "account_changed")

	me := decode[api.Account](t, site.call(http.MethodGet, "/api/auth/me", nil, shobu.session))
	require.NotNil(t, me.Mute)
	require.NotNil(t, me.Mute.Until)

	response = site.call(http.MethodPost, "/api/chat", api.ChatPost{Text: "can anyone hear me?"}, shobu.session)
	require.Equal(t, http.StatusForbidden, response.Code, response.Body.String())
	refusal := decode[api.Error](t, response)
	assert.Equal(t, codeMuted, refusal.Code)
	require.NotNil(t, refusal.Until)
	assert.Equal(t, http.StatusOK, site.call(http.MethodGet, "/api/chat", nil, shobu.session).Code, "still reads the chat")

	response = site.call(http.MethodDelete, "/api/admin/users/"+target.Id+"/mute", nil, boss.session)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	assert.Nil(t, decode[api.AdminUser](t, response).Mute)
	assert.Equal(t, http.StatusCreated, site.call(http.MethodPost, "/api/chat", api.ChatPost{Text: "back again"}, shobu.session).Code)

	tooLong := 400 * 24 * 60
	response = site.call(http.MethodPost, "/api/admin/users/"+target.Id+"/mute", api.MuteInput{DurationMinutes: &tooLong}, boss.session)
	require.Equal(t, http.StatusBadRequest, response.Code)
	invalid := decode[api.Error](t, response)
	require.NotNil(t, invalid.Field)
	assert.Equal(t, "durationMinutes", *invalid.Field)
}

func TestTheGuardsAreAnsweredAsConflicts(t *testing.T) {
	site := newTestServer(t, false)
	boss := site.signUp(t, "Boss")
	shobu := site.signUp(t, "Shobu")
	self := site.adminUser(t, boss, "Boss")
	target := site.adminUser(t, boss, "Shobu")

	conflict := func(method, path string, body any, code string) {
		t.Helper()
		response := site.call(method, path, body, boss)
		require.Equal(t, http.StatusConflict, response.Code, response.Body.String())
		assert.Equal(t, code, decode[api.Error](t, response).Code)
	}

	conflict(http.MethodPost, "/api/admin/users/"+self.Id+"/ban", api.BanInput{Reason: "Oops"}, codeOwnAccount)
	conflict(http.MethodPut, "/api/admin/users/"+self.Id+"/role", api.RoleInput{Role: api.RolePlayer}, codeOwnAccount)

	response := site.call(http.MethodPut, "/api/admin/users/"+target.Id+"/role", api.RoleInput{Role: api.RoleAdmin}, boss)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	assert.Equal(t, api.RoleAdmin, decode[api.AdminUser](t, response).Role)
	assert.Equal(t, api.RoleAdmin, decode[api.Account](t, site.call(http.MethodGet, "/api/auth/me", nil, shobu)).Role)
	conflict(http.MethodPost, "/api/admin/users/"+target.Id+"/ban", api.BanInput{Reason: "Rogue"}, codeAdminBan)

	response = site.call(http.MethodPut, "/api/admin/users/"+target.Id+"/role", api.RoleInput{Role: api.RolePlayer}, boss)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	require.Equal(t, http.StatusOK, site.call(http.MethodPost, "/api/admin/users/"+target.Id+"/ban", api.BanInput{Reason: "Rogue"}, boss).Code)
	conflict(http.MethodPut, "/api/admin/users/"+target.Id+"/role", api.RoleInput{Role: api.RoleAdmin}, codeBannedAdmin)

	assert.Equal(t, http.StatusNotFound, site.call(http.MethodPost, "/api/admin/users/no-such-account/mute", api.MuteInput{}, boss).Code)
	assert.Equal(t, http.StatusBadRequest, site.call(http.MethodPut, "/api/admin/users/"+target.Id+"/role", map[string]string{"role": "owner"}, boss).Code)
}

func TestAComputerOpponentsAccountIsShownButCannotBeChanged(t *testing.T) {
	site := newTestServer(t, false)
	boss := site.signUp(t, "Boss")
	_, err := users.NewService(site.db, storetest.Logger()).CreateComputer(context.Background(), "Grimwhisper")
	require.NoError(t, err)

	computer := site.adminUser(t, boss, "Grimwhisper")
	assert.True(t, computer.Computer)
	assert.False(t, site.adminUser(t, boss, "Boss").Computer)

	for _, change := range []struct {
		method, path string
		body         any
	}{
		{http.MethodPost, "/ban", api.BanInput{Reason: "Too good"}},
		{http.MethodPost, "/mute", api.MuteInput{}},
		{http.MethodPut, "/role", api.RoleInput{Role: api.RoleAdmin}},
		{http.MethodPut, "/password", api.NewPassword{Password: "another long password"}},
	} {
		response := site.call(change.method, "/api/admin/users/"+computer.Id+change.path, change.body, boss)
		require.Equal(t, http.StatusConflict, response.Code, "%s: %s", change.path, response.Body.String())
		assert.Equal(t, codeComputer, decode[api.Error](t, response).Code, change.path)
	}
}

func TestAnAdminGivesAPlayerANewPassword(t *testing.T) {
	site := newTestServer(t, false)
	boss := site.signUp(t, "Boss")
	shobu := site.signUp(t, "Shobu")
	target := site.adminUser(t, boss, "Shobu")

	response := site.call(http.MethodPut, "/api/admin/users/"+target.Id+"/password", api.NewPassword{Password: "short"}, boss)
	require.Equal(t, http.StatusBadRequest, response.Code)
	invalid := decode[api.Error](t, response)
	require.NotNil(t, invalid.Field)
	assert.Equal(t, "password", *invalid.Field)

	response = site.call(http.MethodPut, "/api/admin/users/"+target.Id+"/password", api.NewPassword{Password: "a brand new password"}, boss)
	require.Equal(t, http.StatusNoContent, response.Code, response.Body.String())

	assert.Equal(t, http.StatusUnauthorized, site.call(http.MethodGet, "/api/auth/me", nil, shobu).Code, "the player's sessions end")
	assert.Equal(t, http.StatusUnauthorized, site.call(http.MethodPost, "/api/auth/login", api.Credentials{Username: "Shobu", Password: testPassword}, nil).Code)
	assert.Equal(t, http.StatusOK, site.call(http.MethodPost, "/api/auth/login", api.Credentials{Username: "Shobu", Password: "a brand new password"}, nil).Code)

	log := site.adminLog(t, boss)
	require.Len(t, log, 1)
	assert.Equal(t, api.AdminLogActionPasswordReset, log[0].Action)
	require.NotNil(t, log[0].AdminId)

	assert.Equal(t, http.StatusBadRequest, site.call(http.MethodGet, "/api/admin/log?action=teleported", nil, boss).Code)
}
