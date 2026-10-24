package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/auth"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/config"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/httpapi/api"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/settings"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/store/storetest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/users"
)

const testPassword = "correct horse battery staple"

type authServer struct {
	handler  http.Handler
	settings *settings.Service
}

func newAuthServer(t *testing.T, secureCookies bool) authServer {
	t.Helper()

	db := storetest.Open(t, config.SQLite)
	sqlDB, err := db.DB()
	require.NoError(t, err)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	siteSettings, err := settings.NewService(context.Background(), db, logger, auth.Settings()...)
	require.NoError(t, err)

	authService := auth.NewService(db, users.NewService(db, logger), siteSettings, auth.NewPasswordHasher(4), time.Now, logger)

	return authServer{
		handler:  NewHandler(Dependencies{DB: sqlDB, Auth: authService, Logger: logger, SecureCookies: secureCookies}),
		settings: siteSettings,
	}
}

// call sends a request the way the site does: changes as JSON, with the
// session cookie when one is given.
func (s authServer) call(method, path string, body any, session *http.Cookie) *httptest.ResponseRecorder {
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			panic(err)
		}
		reader = bytes.NewReader(raw)
	}

	request := httptest.NewRequest(method, path, reader)
	if method != http.MethodGet {
		request.Header.Set("Content-Type", "application/json")
	}
	if session != nil {
		request.AddCookie(session)
	}

	recorder := httptest.NewRecorder()
	s.handler.ServeHTTP(recorder, request)
	return recorder
}

func (s authServer) signUp(t *testing.T, username string) *http.Cookie {
	t.Helper()

	response := s.call(http.MethodPost, "/api/auth/signup", api.Credentials{Username: username, Password: testPassword}, nil)
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())

	return sessionCookieOf(t, response)
}

func sessionCookieOf(t *testing.T, response *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()

	for _, cookie := range response.Result().Cookies() {
		if cookie.Name == "session" {
			return cookie
		}
	}
	t.Fatalf("no session cookie in %v", response.Header().Values("Set-Cookie"))
	return nil
}

func TestSignUpAnswersWithTheAccountAndASessionCookie(t *testing.T) {
	server := newAuthServer(t, false)

	response := server.call(http.MethodPost, "/api/auth/signup", api.Credentials{Username: "Shobu", Password: testPassword}, nil)

	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())
	created := decode[api.Account](t, response)
	assert.Equal(t, "Shobu", created.Username)
	assert.Equal(t, api.RoleAdmin, created.Role, "the first account is the admin")
	assert.NotEmpty(t, created.Id)
	assert.WithinDuration(t, time.Now(), created.CreatedAt, time.Minute)

	cookie := sessionCookieOf(t, response)
	assert.NotEmpty(t, cookie.Value)
	assert.True(t, cookie.HttpOnly, "page scripts cannot read it")
	assert.Equal(t, http.SameSiteLaxMode, cookie.SameSite)
	assert.Equal(t, "/", cookie.Path)
	assert.False(t, cookie.Secure, "plain HTTP in development")
	assert.InDelta(t, auth.SessionDuration.Seconds(), cookie.MaxAge, 60)

	me := server.call(http.MethodGet, "/api/auth/me", nil, cookie)
	require.Equal(t, http.StatusOK, me.Code, me.Body.String())
	assert.Equal(t, created.Id, decode[api.Account](t, me).Id)
}

func TestSessionCookiesAreHTTPSOnlyWhenConfigured(t *testing.T) {
	server := newAuthServer(t, true)

	assert.True(t, server.signUp(t, "Shobu").Secure)
}

func TestSignUpErrors(t *testing.T) {
	server := newAuthServer(t, false)
	server.signUp(t, "Shobu")

	invalid := server.call(http.MethodPost, "/api/auth/signup", api.Credentials{Username: "no", Password: testPassword}, nil)
	assert.Equal(t, http.StatusBadRequest, invalid.Code)
	body := decode[api.Error](t, invalid)
	assert.Equal(t, "invalid_input", body.Code)
	assert.Equal(t, "Username must be at least 3 characters.", body.Message)
	require.NotNil(t, body.Field)
	assert.Equal(t, "username", *body.Field)

	weak := server.call(http.MethodPost, "/api/auth/signup", api.Credentials{Username: "Kokujo", Password: "short"}, nil)
	assert.Equal(t, http.StatusBadRequest, weak.Code)
	assert.Equal(t, "password", *decode[api.Error](t, weak).Field)

	taken := server.call(http.MethodPost, "/api/auth/signup", api.Credentials{Username: "SHOBU", Password: testPassword}, nil)
	assert.Equal(t, http.StatusConflict, taken.Code)
	assert.Equal(t, "username_taken", decode[api.Error](t, taken).Code)

	_, err := server.settings.Set(context.Background(), auth.SignupsOpen.Key, "false")
	require.NoError(t, err)
	closed := server.call(http.MethodPost, "/api/auth/signup", api.Credentials{Username: "Kokujo", Password: testPassword}, nil)
	assert.Equal(t, http.StatusForbidden, closed.Code)
	assert.Equal(t, "signups_closed", decode[api.Error](t, closed).Code)
}

func TestLogInAndOut(t *testing.T) {
	server := newAuthServer(t, false)
	server.signUp(t, "Shobu")

	login := server.call(http.MethodPost, "/api/auth/login", api.Credentials{Username: "shobu", Password: testPassword}, nil)
	require.Equal(t, http.StatusOK, login.Code, login.Body.String())
	assert.Equal(t, "Shobu", decode[api.Account](t, login).Username)
	session := sessionCookieOf(t, login)

	logout := server.call(http.MethodPost, "/api/auth/logout", nil, session)
	assert.Equal(t, http.StatusNoContent, logout.Code)
	cleared := sessionCookieOf(t, logout)
	assert.Empty(t, cleared.Value)
	assert.Equal(t, -1, cleared.MaxAge, "the browser deletes it")

	me := server.call(http.MethodGet, "/api/auth/me", nil, session)
	assert.Equal(t, http.StatusUnauthorized, me.Code, "the ended session no longer works")

	again := server.call(http.MethodPost, "/api/auth/logout", nil, nil)
	assert.Equal(t, http.StatusNoContent, again.Code, "logging out without a session is not an error")
}

func TestLogInErrors(t *testing.T) {
	server := newAuthServer(t, false)
	server.signUp(t, "Shobu")

	wrong := server.call(http.MethodPost, "/api/auth/login", api.Credentials{Username: "Shobu", Password: "not the password"}, nil)
	unknown := server.call(http.MethodPost, "/api/auth/login", api.Credentials{Username: "Nobody", Password: testPassword}, nil)

	for _, response := range []*httptest.ResponseRecorder{wrong, unknown} {
		assert.Equal(t, http.StatusUnauthorized, response.Code)
		body := decode[api.Error](t, response)
		assert.Equal(t, "invalid_credentials", body.Code)
		assert.Equal(t, "Wrong username or password.", body.Message, "the answer does not say which part was wrong")
		assert.Empty(t, response.Header().Values("Set-Cookie"))
	}

	for range 4 {
		server.call(http.MethodPost, "/api/auth/login", api.Credentials{Username: "Shobu", Password: "not the password"}, nil)
	}
	throttled := server.call(http.MethodPost, "/api/auth/login", api.Credentials{Username: "Shobu", Password: testPassword}, nil)
	assert.Equal(t, http.StatusTooManyRequests, throttled.Code)
	assert.Equal(t, "900", throttled.Header().Get("Retry-After"))
	body := decode[api.Error](t, throttled)
	assert.Equal(t, "too_many_attempts", body.Code)
	assert.Equal(t, "Too many failed attempts. Try again in 15 minutes.", body.Message)
}

func TestCurrentAccountNeedsASession(t *testing.T) {
	server := newAuthServer(t, false)

	for name, cookie := range map[string]*http.Cookie{
		"no cookie":     nil,
		"unknown token": {Name: "session", Value: "made-up"},
		"empty cookie":  {Name: "session", Value: ""},
	} {
		response := server.call(http.MethodGet, "/api/auth/me", nil, cookie)
		assert.Equal(t, http.StatusUnauthorized, response.Code, name)
		body := decode[api.Error](t, response)
		assert.Equal(t, "unauthenticated", body.Code, name)
		assert.Equal(t, "You are not logged in.", body.Message, name)
	}
}

func TestChangePassword(t *testing.T) {
	server := newAuthServer(t, false)
	session := server.signUp(t, "Shobu")

	unauthenticated := server.call(http.MethodPost, "/api/auth/password", api.PasswordChange{CurrentPassword: testPassword, NewPassword: "a brand new password"}, nil)
	assert.Equal(t, http.StatusUnauthorized, unauthenticated.Code)

	wrong := server.call(http.MethodPost, "/api/auth/password", api.PasswordChange{CurrentPassword: "not it", NewPassword: "a brand new password"}, session)
	assert.Equal(t, http.StatusForbidden, wrong.Code)
	assert.Equal(t, "wrong_password", decode[api.Error](t, wrong).Code)

	weak := server.call(http.MethodPost, "/api/auth/password", api.PasswordChange{CurrentPassword: testPassword, NewPassword: "short"}, session)
	assert.Equal(t, http.StatusBadRequest, weak.Code)
	body := decode[api.Error](t, weak)
	require.NotNil(t, body.Field)
	assert.Equal(t, "newPassword", *body.Field, "named as the request names it")
	assert.Equal(t, "Password must be at least 8 characters.", body.Message)

	changed := server.call(http.MethodPost, "/api/auth/password", api.PasswordChange{CurrentPassword: testPassword, NewPassword: "a brand new password"}, session)
	assert.Equal(t, http.StatusNoContent, changed.Code, changed.Body.String())

	assert.Equal(t, http.StatusOK, server.call(http.MethodGet, "/api/auth/me", nil, session).Code, "this session stays logged in")
	old := server.call(http.MethodPost, "/api/auth/login", api.Credentials{Username: "Shobu", Password: testPassword}, nil)
	assert.Equal(t, http.StatusUnauthorized, old.Code)
	fresh := server.call(http.MethodPost, "/api/auth/login", api.Credentials{Username: "Shobu", Password: "a brand new password"}, nil)
	assert.Equal(t, http.StatusOK, fresh.Code)
}

func TestChangesMustBeSentAsJSON(t *testing.T) {
	server := newAuthServer(t, false)

	for _, contentType := range []string{"", "text/plain", "application/x-www-form-urlencoded", "multipart/form-data; boundary=x"} {
		request := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"username":"Shobu","password":"correct horse battery staple"}`))
		if contentType != "" {
			request.Header.Set("Content-Type", contentType)
		}
		response := httptest.NewRecorder()
		server.handler.ServeHTTP(response, request)

		assert.Equal(t, http.StatusUnsupportedMediaType, response.Code, "%q", contentType)
		assert.Equal(t, "unsupported_media_type", decode[api.Error](t, response).Code)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/auth/signup", strings.NewReader(`{"username":"Shobu","password":"correct horse battery staple"}`))
	request.Header.Set("Content-Type", "application/json; charset=utf-8")
	response := httptest.NewRecorder()
	server.handler.ServeHTTP(response, request)
	assert.Equal(t, http.StatusCreated, response.Code, "a charset parameter is fine")
}

func TestMalformedJSONIsABadRequest(t *testing.T) {
	server := newAuthServer(t, false)

	request := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"username":`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	server.handler.ServeHTTP(response, request)

	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.Equal(t, "bad_request", decode[api.Error](t, response).Code)
}
