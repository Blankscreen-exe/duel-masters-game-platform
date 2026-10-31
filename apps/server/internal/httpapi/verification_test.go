package httpapi

import (
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/httpapi/api"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/mail/mailtest"
)

var emailedCode = regexp.MustCompile(`\b\d{6}\b`)

// setUpMail has the admin point the site at a real test mail server.
func setUpMail(t *testing.T, site testServer, admin *http.Cookie) *mailtest.Server {
	t.Helper()

	server := mailtest.Start(t, mailtest.Options{})
	noPassword := ""
	response := site.call(http.MethodPut, "/api/admin/mail", api.MailSettingsInput{
		Host: "127.0.0.1", Port: server.Port, Security: api.MailSecurityNone, Password: &noPassword, FromAddress: "noreply@example.test", FromName: "Duel Masters",
	}, admin)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())

	return server
}

// lastEmailedCode is the code in the last email the server took.
func lastEmailedCode(t *testing.T, server *mailtest.Server) string {
	t.Helper()

	messages := server.Messages()
	require.NotEmpty(t, messages)
	_, body, _ := strings.Cut(messages[len(messages)-1].Data, "\r\n\r\n")
	code := emailedCode.FindString(body)
	require.NotEmpty(t, code)
	return code
}

// newcomer signs up through the API and stays unconfirmed.
func newcomer(t *testing.T, site testServer, username string) *http.Cookie {
	t.Helper()

	response := site.call(http.MethodPost, "/api/auth/signup", api.SignUpInput{Username: username, Email: strings.ToLower(username) + "@example.test", Password: testPassword}, nil)
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())

	account := decode[api.Account](t, response)
	require.False(t, account.EmailVerified)
	return sessionCookieOf(t, response)
}

func TestSigningUpAndConfirmingWithACode(t *testing.T) {
	site := newTestServer(t, false)
	admin := site.signUp(t, "Admin")
	server := setUpMail(t, site, admin)

	player := newcomer(t, site, "Shobu")

	messages := server.Messages()
	require.Len(t, messages, 1)
	assert.Equal(t, "shobu@example.test", messages[0].To)

	t.Run("until then it is as if they were not logged in", func(t *testing.T) {
		for _, path := range []string{"/api/decks", "/api/lobby"} {
			assert.Equal(t, http.StatusUnauthorized, site.call(http.MethodGet, path, nil, player).Code, path)
		}
		assert.Equal(t, http.StatusUnauthorized, site.call(http.MethodPost, "/api/chat", api.ChatPost{Text: "hello"}, player).Code)
	})

	t.Run("but they can see where they stand", func(t *testing.T) {
		me := site.call(http.MethodGet, "/api/auth/me", nil, player)
		require.Equal(t, http.StatusOK, me.Code)
		account := decode[api.Account](t, me)
		assert.False(t, account.EmailVerified)
		require.NotNil(t, account.Email)
		assert.Equal(t, "shobu@example.test", *account.Email)

		status := site.call(http.MethodGet, "/api/auth/verification", nil, player)
		require.Equal(t, http.StatusOK, status.Code)
		verification := decode[api.Verification](t, status)
		assert.True(t, verification.MailWorks)
		assert.Equal(t, "shobu@example.test", verification.Email)
		assert.Equal(t, 60, verification.ResendInSeconds)
		assert.NotNil(t, verification.CodeSentAt)
		assert.NotNil(t, verification.DeleteAt)
	})

	t.Run("not again too soon", func(t *testing.T) {
		response := site.call(http.MethodPost, "/api/auth/verification/code", nil, player)
		assert.Equal(t, http.StatusTooManyRequests, response.Code)
		assert.Equal(t, "60", response.Header().Get("Retry-After"))
	})

	t.Run("a wrong code and nonsense are refused", func(t *testing.T) {
		wrong := "000000"
		if lastEmailedCode(t, server) == wrong {
			wrong = "111111"
		}
		response := site.call(http.MethodPost, "/api/auth/verification", api.VerificationCode{Code: wrong}, player)
		assert.Equal(t, http.StatusBadRequest, response.Code)
		assert.Equal(t, "wrong_code", decode[api.Error](t, response).Code)

		response = site.call(http.MethodPost, "/api/auth/verification", api.VerificationCode{Code: "abc"}, player)
		assert.Equal(t, http.StatusBadRequest, response.Code)
		assert.Equal(t, "invalid_input", decode[api.Error](t, response).Code)
	})

	t.Run("the right code lets them in", func(t *testing.T) {
		response := site.call(http.MethodPost, "/api/auth/verification", api.VerificationCode{Code: lastEmailedCode(t, server)}, player)
		require.Equal(t, http.StatusOK, response.Code, response.Body.String())
		assert.True(t, decode[api.Account](t, response).EmailVerified)

		assert.Equal(t, http.StatusOK, site.call(http.MethodGet, "/api/decks", nil, player).Code)
	})

	t.Run("and there is nothing left to do", func(t *testing.T) {
		response := site.call(http.MethodPost, "/api/auth/verification", api.VerificationCode{Code: "123456"}, player)
		assert.Equal(t, http.StatusConflict, response.Code)
		assert.Equal(t, "email_confirmed", decode[api.Error](t, response).Code)

		response = site.call(http.MethodPost, "/api/auth/verification/code", nil, player)
		assert.Equal(t, http.StatusConflict, response.Code)

		response = site.call(http.MethodPut, "/api/auth/verification/email", api.EmailChange{Email: "x@example.test"}, player)
		assert.Equal(t, http.StatusConflict, response.Code)
	})
}

func TestLoggingBackInBeforeConfirming(t *testing.T) {
	site := newTestServer(t, false)
	admin := site.signUp(t, "Admin")
	server := setUpMail(t, site, admin)
	newcomer(t, site, "Shobu")

	login := site.call(http.MethodPost, "/api/auth/login", api.Credentials{Username: "Shobu", Password: testPassword}, nil)
	require.Equal(t, http.StatusOK, login.Code, login.Body.String())
	assert.False(t, decode[api.Account](t, login).EmailVerified)

	cookie := sessionCookieOf(t, login)
	assert.Equal(t, http.StatusUnauthorized, site.call(http.MethodGet, "/api/decks", nil, cookie).Code)

	response := site.call(http.MethodPost, "/api/auth/verification", api.VerificationCode{Code: lastEmailedCode(t, server)}, cookie)
	assert.Equal(t, http.StatusOK, response.Code)
}

func TestSignUpRefusesAnAddressInUse(t *testing.T) {
	site := newTestServer(t, false)
	site.signUp(t, "Admin")

	taken := site.call(http.MethodPost, "/api/auth/signup", api.SignUpInput{Username: "Shobu", Email: "ADMIN@example.test", Password: testPassword}, nil)
	assert.Equal(t, http.StatusConflict, taken.Code)
	assert.Equal(t, "email_taken", decode[api.Error](t, taken).Code)

	bad := site.call(http.MethodPost, "/api/auth/signup", api.SignUpInput{Username: "Shobu", Email: "nonsense", Password: testPassword}, nil)
	assert.Equal(t, http.StatusBadRequest, bad.Code)
	refusal := decode[api.Error](t, bad)
	require.NotNil(t, refusal.Field)
	assert.Equal(t, "email", *refusal.Field)
}

func TestWaitingForTheMailServer(t *testing.T) {
	site := newTestServer(t, false)
	admin := site.signUp(t, "Admin")

	player := newcomer(t, site, "Shobu")

	status := decode[api.Verification](t, site.call(http.MethodGet, "/api/auth/verification", nil, player))
	assert.False(t, status.MailWorks)
	assert.Nil(t, status.DeleteAt, "nothing counts down while they wait")

	response := site.call(http.MethodPost, "/api/auth/verification/code", nil, player)
	assert.Equal(t, http.StatusConflict, response.Code)
	assert.Equal(t, "mail_unavailable", decode[api.Error](t, response).Code)

	server := setUpMail(t, site, admin)

	assert.True(t, decode[api.Verification](t, site.call(http.MethodGet, "/api/auth/verification", nil, player)).MailWorks)
	require.Equal(t, http.StatusOK, site.call(http.MethodPost, "/api/auth/verification/code", nil, player).Code)

	response = site.call(http.MethodPost, "/api/auth/verification", api.VerificationCode{Code: lastEmailedCode(t, server)}, player)
	assert.Equal(t, http.StatusOK, response.Code)
}

func TestAnAdminLettingAPlayerIn(t *testing.T) {
	site := newTestServer(t, false)
	admin := site.signUp(t, "Admin")
	player := newcomer(t, site, "Shobu")

	waiting := decode[api.AdminUserPage](t, site.call(http.MethodGet, "/api/admin/users?status=unverified", nil, admin))
	require.Len(t, waiting.Users, 1)
	found := waiting.Users[0]
	assert.Equal(t, "Shobu", found.Username)
	assert.False(t, found.EmailVerified)
	require.NotNil(t, found.Email)
	assert.Equal(t, "shobu@example.test", *found.Email)

	t.Run("only an admin can", func(t *testing.T) {
		other := site.signUp(t, "Kokujo")
		assert.Equal(t, http.StatusForbidden, site.call(http.MethodPost, "/api/admin/users/"+found.Id+"/let-in", nil, other).Code)
		assert.Equal(t, http.StatusUnauthorized, site.call(http.MethodPost, "/api/admin/users/"+found.Id+"/let-in", nil, player).Code)
	})

	response := site.call(http.MethodPost, "/api/admin/users/"+found.Id+"/let-in", nil, admin)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	assert.True(t, decode[api.AdminUser](t, response).EmailVerified)

	assert.Equal(t, http.StatusOK, site.call(http.MethodGet, "/api/decks", nil, player).Code, "they are in")

	waiting = decode[api.AdminUserPage](t, site.call(http.MethodGet, "/api/admin/users?status=unverified", nil, admin))
	assert.Empty(t, waiting.Users)

	log := decode[api.AdminLogPage](t, site.call(http.MethodGet, "/api/admin/log?action=player_let_in", nil, admin))
	require.Len(t, log.Entries, 1)
	assert.Equal(t, "Shobu", log.Entries[0].Subject)

	t.Run("letting them in again records nothing", func(t *testing.T) {
		require.Equal(t, http.StatusOK, site.call(http.MethodPost, "/api/admin/users/"+found.Id+"/let-in", nil, admin).Code)
		log := decode[api.AdminLogPage](t, site.call(http.MethodGet, "/api/admin/log?action=player_let_in", nil, admin))
		assert.Len(t, log.Entries, 1)
	})
}

func TestCorrectingAnAddressThroughTheAPI(t *testing.T) {
	site := newTestServer(t, false)
	admin := site.signUp(t, "Admin")
	server := setUpMail(t, site, admin)
	player := newcomer(t, site, "Shobu")

	tooSoon := site.call(http.MethodPut, "/api/auth/verification/email", api.EmailChange{Email: "right@example.test"}, player)
	assert.Equal(t, http.StatusTooManyRequests, tooSoon.Code)

	site.clock.Advance(time.Minute)

	taken := site.call(http.MethodPut, "/api/auth/verification/email", api.EmailChange{Email: "admin@example.test"}, player)
	assert.Equal(t, http.StatusConflict, taken.Code)
	assert.Equal(t, "email_taken", decode[api.Error](t, taken).Code)

	changed := site.call(http.MethodPut, "/api/auth/verification/email", api.EmailChange{Email: "right@example.test"}, player)
	require.Equal(t, http.StatusOK, changed.Code, changed.Body.String())
	assert.Equal(t, "right@example.test", decode[api.Verification](t, changed).Email)

	messages := server.Messages()
	assert.Equal(t, "right@example.test", messages[len(messages)-1].To)
}

func TestMailSettingsThroughTheAPI(t *testing.T) {
	site := newTestServer(t, false)
	admin := site.signUp(t, "Admin")
	player := site.signUp(t, "Shobu")

	t.Run("only admins", func(t *testing.T) {
		assert.Equal(t, http.StatusUnauthorized, site.call(http.MethodGet, "/api/admin/mail", nil, nil).Code)
		assert.Equal(t, http.StatusForbidden, site.call(http.MethodGet, "/api/admin/mail", nil, player).Code)
		assert.Equal(t, http.StatusForbidden, site.call(http.MethodPost, "/api/admin/mail/test", api.TestMailInput{To: "a@example.test"}, player).Code)
		assert.Equal(t, http.StatusForbidden, site.call(http.MethodDelete, "/api/admin/mail", nil, player).Code)
	})

	t.Run("nothing set up", func(t *testing.T) {
		shown := decode[api.MailSettings](t, site.call(http.MethodGet, "/api/admin/mail", nil, admin))
		assert.False(t, shown.Configured)
		assert.True(t, shown.KeyAvailable)

		response := site.call(http.MethodPost, "/api/admin/mail/test", api.TestMailInput{To: "a@example.test"}, admin)
		assert.Equal(t, http.StatusConflict, response.Code)
		assert.Equal(t, "mail_unavailable", decode[api.Error](t, response).Code)
	})

	t.Run("the password goes in and never comes out", func(t *testing.T) {
		secret := "the-smtp-password"
		input := api.MailSettingsInput{
			Host: "smtp.example.test", Port: 587, Security: api.MailSecurityStarttls, Username: "mailer", Password: &secret,
			FromAddress: "noreply@example.test", FromName: "Duel Masters",
		}
		response := site.call(http.MethodPut, "/api/admin/mail", input, admin)
		require.Equal(t, http.StatusOK, response.Code, response.Body.String())
		assert.NotContains(t, response.Body.String(), secret)

		shown := decode[api.MailSettings](t, site.call(http.MethodGet, "/api/admin/mail", nil, admin))
		assert.True(t, shown.Configured)
		assert.True(t, shown.HasPassword)
		require.NotNil(t, shown.Host)
		assert.Equal(t, "smtp.example.test", *shown.Host)

		var sealed string
		require.NoError(t, site.db.Table("mail_settings").Select("sealed").Scan(&sealed).Error)
		assert.NotEmpty(t, sealed)
		assert.NotContains(t, sealed, secret)
		assert.NotContains(t, sealed, "smtp.example.test")

		log := site.call(http.MethodGet, "/api/admin/log?action=mail_settings_changed", nil, admin)
		assert.NotContains(t, log.Body.String(), secret)
		assert.Len(t, decode[api.AdminLogPage](t, log).Entries, 1)
	})

	t.Run("bad settings are refused", func(t *testing.T) {
		response := site.call(http.MethodPut, "/api/admin/mail", api.MailSettingsInput{Host: "smtp.example.test", Port: 587, Security: api.MailSecurityStarttls, FromAddress: "nonsense"}, admin)
		assert.Equal(t, http.StatusBadRequest, response.Code)
		refusal := decode[api.Error](t, response)
		require.NotNil(t, refusal.Field)
		assert.Equal(t, "fromAddress", *refusal.Field)
	})

	t.Run("a test email says why it failed", func(t *testing.T) {
		response := site.call(http.MethodPut, "/api/admin/mail", api.MailSettingsInput{Host: "127.0.0.1", Port: 1, Security: api.MailSecurityNone, FromAddress: "noreply@example.test"}, admin)
		require.Equal(t, http.StatusOK, response.Code)

		failed := site.call(http.MethodPost, "/api/admin/mail/test", api.TestMailInput{To: "a@example.test"}, admin)
		assert.Equal(t, http.StatusBadGateway, failed.Code)
		refusal := decode[api.Error](t, failed)
		assert.Equal(t, "mail_failed", refusal.Code)
		assert.Contains(t, refusal.Message, "connecting")

		assert.Equal(t, http.StatusBadRequest, site.call(http.MethodPost, "/api/admin/mail/test", api.TestMailInput{To: "nonsense"}, admin).Code)
	})

	t.Run("a test email goes through", func(t *testing.T) {
		server := setUpMail(t, site, admin)

		require.Equal(t, http.StatusNoContent, site.call(http.MethodPost, "/api/admin/mail/test", api.TestMailInput{To: "admin@example.test"}, admin).Code)
		messages := server.Messages()
		require.Len(t, messages, 1)
		assert.Equal(t, "admin@example.test", messages[0].To)
	})

	t.Run("forgetting the mail server", func(t *testing.T) {
		response := site.call(http.MethodDelete, "/api/admin/mail", nil, admin)
		require.Equal(t, http.StatusOK, response.Code)
		assert.False(t, decode[api.MailSettings](t, response).Configured)

		log := decode[api.AdminLogPage](t, site.call(http.MethodGet, "/api/admin/log?action=mail_settings_cleared", nil, admin))
		assert.Len(t, log.Entries, 1)
	})
}

func TestProfilesKeepAddressesPrivate(t *testing.T) {
	site := newTestServer(t, false)
	site.signUp(t, "Admin")
	player := site.signUp(t, "Shobu")

	response := site.call(http.MethodGet, "/api/players/Admin", nil, player)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	assert.NotContains(t, response.Body.String(), "admin@example.test")
}
