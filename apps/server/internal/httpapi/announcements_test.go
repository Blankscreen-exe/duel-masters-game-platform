package httpapi

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/httpapi/api"
)

// post has the admin post a notification, a moment later than anything before.
func (s testServer) post(t *testing.T, admin *http.Cookie, title string, important bool) api.Announcement {
	t.Helper()

	s.clock.Advance(time.Second)
	response := s.call(http.MethodPost, "/api/admin/announcements", api.AnnouncementInput{Title: title, Body: "Details.", Important: important}, admin)
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())
	return decode[api.Announcement](t, response)
}

func TestNotificationsThroughTheAPI(t *testing.T) {
	site := newTestServer(t, false)
	admin := site.signUp(t, "Admin")
	player := site.signUp(t, "Shobu")
	// Accounts take their time from the database's clock, which runs ahead of a
	// stopped test clock.
	site.clock.Advance(time.Minute)

	posted := site.post(t, admin, "Maintenance tonight", true)
	site.post(t, admin, "Plain news", false)

	t.Run("everyone can read them, without logging in", func(t *testing.T) {
		response := site.call(http.MethodGet, "/api/announcements", nil, nil)
		require.Equal(t, http.StatusOK, response.Code, response.Body.String())

		page := decode[api.AnnouncementPage](t, response)
		assert.Equal(t, 2, page.Total)
		require.Len(t, page.Announcements, 2)
		assert.Equal(t, "Plain news", page.Announcements[0].Title, "the latest first")
		assert.True(t, page.Announcements[1].Important)
	})

	t.Run("a bad page is refused", func(t *testing.T) {
		assert.Equal(t, http.StatusBadRequest, site.call(http.MethodGet, "/api/announcements?pageSize=100000", nil, nil).Code)
	})

	t.Run("only players have an own view", func(t *testing.T) {
		assert.Equal(t, http.StatusUnauthorized, site.call(http.MethodGet, "/api/announcements/mine", nil, nil).Code)
		assert.Equal(t, http.StatusUnauthorized, site.call(http.MethodPost, "/api/announcements/seen", nil, nil).Code)
		assert.Equal(t, http.StatusUnauthorized, site.call(http.MethodPost, "/api/announcements/"+posted.Id+"/dismiss", nil, nil).Code)
	})

	t.Run("a player has unread ones and a popup", func(t *testing.T) {
		mine := decode[api.MyAnnouncements](t, site.call(http.MethodGet, "/api/announcements/mine", nil, player))
		assert.Equal(t, 2, mine.Unread)
		require.Len(t, mine.Popups, 1)
		assert.Equal(t, posted.Id, mine.Popups[0].Id)
	})

	t.Run("looking clears the count, closing clears the popup", func(t *testing.T) {
		site.clock.Advance(time.Second)
		require.Equal(t, http.StatusNoContent, site.call(http.MethodPost, "/api/announcements/seen", nil, player).Code)

		mine := decode[api.MyAnnouncements](t, site.call(http.MethodGet, "/api/announcements/mine", nil, player))
		assert.Zero(t, mine.Unread)
		assert.Len(t, mine.Popups, 1)

		require.Equal(t, http.StatusNoContent, site.call(http.MethodPost, "/api/announcements/"+posted.Id+"/dismiss", nil, player).Code)
		mine = decode[api.MyAnnouncements](t, site.call(http.MethodGet, "/api/announcements/mine", nil, player))
		assert.Empty(t, mine.Popups)
		assert.NotNil(t, mine.Popups, "an empty list, not nothing")

		assert.Equal(t, http.StatusNotFound, site.call(http.MethodPost, "/api/announcements/no-such-id/dismiss", nil, player).Code)
	})

	t.Run("only admins change them", func(t *testing.T) {
		input := api.AnnouncementInput{Title: "Mine", Body: "Mine."}
		assert.Equal(t, http.StatusUnauthorized, site.call(http.MethodPost, "/api/admin/announcements", input, nil).Code)
		assert.Equal(t, http.StatusForbidden, site.call(http.MethodPost, "/api/admin/announcements", input, player).Code)
		assert.Equal(t, http.StatusForbidden, site.call(http.MethodPut, "/api/admin/announcements/"+posted.Id, input, player).Code)
		assert.Equal(t, http.StatusForbidden, site.call(http.MethodDelete, "/api/admin/announcements/"+posted.Id, nil, player).Code)
	})

	t.Run("bad text is refused, naming the field", func(t *testing.T) {
		response := site.call(http.MethodPost, "/api/admin/announcements", api.AnnouncementInput{Title: "  ", Body: "B"}, admin)
		assert.Equal(t, http.StatusBadRequest, response.Code)
		refusal := decode[api.Error](t, response)
		require.NotNil(t, refusal.Field)
		assert.Equal(t, "title", *refusal.Field)
	})

	t.Run("showing it again brings the popup back", func(t *testing.T) {
		site.clock.Advance(time.Second)
		again := true
		response := site.call(http.MethodPut, "/api/admin/announcements/"+posted.Id, api.AnnouncementInput{Title: "Maintenance moved", Body: "Tomorrow.", Important: true, ShowAgain: &again}, admin)
		require.Equal(t, http.StatusOK, response.Code, response.Body.String())
		assert.Equal(t, "Maintenance moved", decode[api.Announcement](t, response).Title)

		mine := decode[api.MyAnnouncements](t, site.call(http.MethodGet, "/api/announcements/mine", nil, player))
		require.Len(t, mine.Popups, 1)
		assert.Equal(t, "Maintenance moved", mine.Popups[0].Title)
	})

	t.Run("one that is gone", func(t *testing.T) {
		input := api.AnnouncementInput{Title: "A", Body: "B"}
		assert.Equal(t, http.StatusNotFound, site.call(http.MethodPut, "/api/admin/announcements/no-such-id", input, admin).Code)
		assert.Equal(t, http.StatusNotFound, site.call(http.MethodDelete, "/api/admin/announcements/no-such-id", nil, admin).Code)
	})

	t.Run("deleting takes it from everyone and goes in the log", func(t *testing.T) {
		require.Equal(t, http.StatusNoContent, site.call(http.MethodDelete, "/api/admin/announcements/"+posted.Id, nil, admin).Code)

		mine := decode[api.MyAnnouncements](t, site.call(http.MethodGet, "/api/announcements/mine", nil, player))
		assert.Empty(t, mine.Popups)

		log := decode[api.AdminLogPage](t, site.call(http.MethodGet, "/api/admin/log?action=announcement_deleted", nil, admin))
		require.Len(t, log.Entries, 1)
		assert.Equal(t, "Maintenance moved", log.Entries[0].Subject)
	})
}

func TestAPlayerNotYetConfirmedHasNoOwnNotifications(t *testing.T) {
	site := newTestServer(t, false)
	site.signUp(t, "Admin")
	waiting := newcomer(t, site, "Shobu")

	assert.Equal(t, http.StatusUnauthorized, site.call(http.MethodGet, "/api/announcements/mine", nil, waiting).Code)
	assert.Equal(t, http.StatusOK, site.call(http.MethodGet, "/api/announcements", nil, waiting).Code, "but can read them as a visitor does")
}
