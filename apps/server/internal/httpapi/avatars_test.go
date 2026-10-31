package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/avatars"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/httpapi/api"
)

const avatarPath = "/api/auth/avatar"

func TestThePicturesAreListedAndServedToAnyone(t *testing.T) {
	server := newTestServer(t, false)

	list := decode[[]api.Avatar](t, server.call(http.MethodGet, "/api/avatars", nil, nil))
	require.Len(t, list, len(avatars.IDs()))
	first := list[0]
	assert.Equal(t, avatars.IDs()[0], first.Id)
	assert.Equal(t, "/api/avatars/"+first.Id, first.ImageUrl)

	picture := server.call(http.MethodGet, first.ImageUrl, nil, nil)
	require.Equal(t, http.StatusOK, picture.Code)
	assert.Equal(t, "image/png", picture.Header().Get("Content-Type"))
	assert.Equal(t, pictureCacheControl, picture.Header().Get("Cache-Control"))
	want, found := avatars.ByID(first.Id)
	require.True(t, found)
	assert.Equal(t, want.Data, picture.Body.Bytes())

	// A browser's copy that is still current is not sent again.
	request := httptest.NewRequest(http.MethodGet, first.ImageUrl, nil)
	request.Header.Set("If-None-Match", picture.Header().Get("ETag"))
	again := httptest.NewRecorder()
	server.handler.ServeHTTP(again, request)
	assert.Equal(t, http.StatusNotModified, again.Code)

	assert.Equal(t, http.StatusNotFound, server.call(http.MethodGet, "/api/avatars/nothing", nil, nil).Code)
}

func TestAPlayerChoosesTheirPicture(t *testing.T) {
	server := newTestServer(t, false)
	shobu := server.signUp(t, "Shobu")
	kokujo := server.signUp(t, "Kokujo")
	picture := avatars.IDs()[3]

	me := decode[api.Account](t, server.call(http.MethodGet, "/api/auth/me", nil, shobu))
	assert.Nil(t, me.AvatarUrl, "none until they choose")

	// Only for their own account, and only one of the pictures.
	assert.Equal(t, http.StatusUnauthorized, server.call(http.MethodPut, avatarPath, api.AvatarChoice{AvatarId: picture}, nil).Code)
	refused := server.call(http.MethodPut, avatarPath, api.AvatarChoice{AvatarId: "nothing"}, shobu)
	assert.Equal(t, http.StatusBadRequest, refused.Code)
	field := decode[api.Error](t, refused).Field
	require.NotNil(t, field)
	assert.Equal(t, "avatarId", *field)

	chosen := server.call(http.MethodPut, avatarPath, api.AvatarChoice{AvatarId: picture}, shobu)
	require.Equal(t, http.StatusOK, chosen.Code, chosen.Body.String())
	want := "/api/avatars/" + picture
	require.NotNil(t, decode[api.Account](t, chosen).AvatarUrl)
	assert.Equal(t, want, *decode[api.Account](t, chosen).AvatarUrl)

	// It stays, and other players see it.
	me = decode[api.Account](t, server.call(http.MethodGet, "/api/auth/me", nil, shobu))
	require.NotNil(t, me.AvatarUrl)
	assert.Equal(t, want, *me.AvatarUrl)

	profile := decode[api.PlayerProfile](t, server.call(http.MethodGet, "/api/players/Shobu", nil, kokujo))
	require.NotNil(t, profile.AvatarUrl)
	assert.Equal(t, want, *profile.AvatarUrl)
	own := decode[api.PlayerProfile](t, server.call(http.MethodGet, "/api/players/Kokujo", nil, kokujo))
	assert.Nil(t, own.AvatarUrl, "and it is only theirs")
}

func TestAPictureNoLongerOfferedShowsAsNone(t *testing.T) {
	assert.Nil(t, avatarURL(""))
	assert.Nil(t, avatarURL("taken-away-since"))
	require.NotNil(t, avatarURL(avatars.IDs()[0]))
}
