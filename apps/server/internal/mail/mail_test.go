package mail

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/mail/mailtest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/secrets"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/store/storetest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/validation"
)

var testNow = time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)

func discard() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func testBox(t *testing.T, fill string) *secrets.Box {
	t.Helper()

	box, err := secrets.NewBox([]byte(strings.Repeat(fill, secrets.KeySize)))
	require.NoError(t, err)
	return box
}

func pointer(text string) *string { return &text }

func goodInput() Input {
	return Input{
		Host:        "smtp.example.com",
		Port:        587,
		Security:    StartTLS,
		Username:    "mailer",
		Password:    pointer("the-smtp-password"),
		FromAddress: "noreply@example.com",
		FromName:    "Duel Masters",
	}
}

func save(t *testing.T, service *Service, db *gorm.DB, input Input) {
	t.Helper()

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return service.SaveIn(context.Background(), tx, input, testNow)
	}))
}

func TestSettingsAreEncryptedAtRest(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		service := NewService(db, testBox(t, "a"), discard())
		save(t, service, db, goodInput())

		var row settingsRow
		require.NoError(t, db.Take(&row).Error)
		for _, secret := range []string{"the-smtp-password", "mailer", "smtp.example.com", "noreply@example.com"} {
			assert.NotContains(t, row.Sealed, secret)
		}

		status, err := service.Status(context.Background())
		require.NoError(t, err)
		assert.True(t, status.Configured)
		assert.True(t, status.HasPassword)
		assert.Equal(t, "smtp.example.com", status.Host)
		assert.Equal(t, 587, status.Port)
		assert.Equal(t, "mailer", status.Username)
		assert.Empty(t, status.Password, "the password never leaves")
		require.NotNil(t, status.UpdatedAt)
	})
}

func TestStatusBeforeAnythingIsSet(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		service := NewService(db, testBox(t, "a"), discard())

		status, err := service.Status(context.Background())
		require.NoError(t, err)
		assert.False(t, status.Configured)
		assert.True(t, status.KeyAvailable)
		assert.Nil(t, status.UpdatedAt)

		configured, err := service.Configured(context.Background())
		require.NoError(t, err)
		assert.False(t, configured)

		assert.ErrorIs(t, service.Send(context.Background(), "someone@example.com", "s", "b"), ErrNotConfigured)
	})
}

func TestSavingWithoutAPasswordKeepsTheStoredOne(t *testing.T) {
	db := storetest.Open(t, "sqlite")
	service := NewService(db, testBox(t, "a"), discard())
	save(t, service, db, goodInput())

	changed := goodInput()
	changed.Host = "smtp.other.example.com"
	changed.Password = nil
	save(t, service, db, changed)

	stored, _, found, err := service.read(context.Background(), db)
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, "smtp.other.example.com", stored.Host)
	assert.Equal(t, "the-smtp-password", stored.Password)

	cleared := goodInput()
	cleared.Password = pointer("")
	save(t, service, db, cleared)

	status, err := service.Status(context.Background())
	require.NoError(t, err)
	assert.False(t, status.HasPassword, "an empty password removes it")
}

func TestWithoutAKeyNothingCanBeStored(t *testing.T) {
	db := storetest.Open(t, "sqlite")
	service := NewService(db, nil, discard())

	err := db.Transaction(func(tx *gorm.DB) error {
		return service.SaveIn(context.Background(), tx, goodInput(), testNow)
	})
	assert.ErrorIs(t, err, ErrNoSecretKey)

	status, err := service.Status(context.Background())
	require.NoError(t, err)
	assert.False(t, status.KeyAvailable)
	assert.False(t, status.Configured)
}

func TestSettingsReadableOnlyWithTheirKey(t *testing.T) {
	db := storetest.Open(t, "sqlite")
	save(t, NewService(db, testBox(t, "a"), discard()), db, goodInput())

	for name, box := range map[string]*secrets.Box{"another key": testBox(t, "b"), "no key": nil} {
		t.Run(name, func(t *testing.T) {
			service := NewService(db, box, discard())

			status, err := service.Status(context.Background())
			require.NoError(t, err)
			assert.True(t, status.Unreadable)
			assert.False(t, status.Configured)
			assert.Empty(t, status.Host)

			configured, err := service.Configured(context.Background())
			require.NoError(t, err)
			assert.False(t, configured)

			assert.ErrorIs(t, service.Send(context.Background(), "someone@example.com", "s", "b"), ErrUnreadable)
		})
	}

	t.Run("new settings replace ones that cannot be read", func(t *testing.T) {
		service := NewService(db, testBox(t, "b"), discard())
		save(t, service, db, goodInput())

		configured, err := service.Configured(context.Background())
		require.NoError(t, err)
		assert.True(t, configured)
	})
}

func TestClearForgetsTheSettings(t *testing.T) {
	db := storetest.Open(t, "sqlite")
	service := NewService(db, testBox(t, "a"), discard())
	save(t, service, db, goodInput())

	var had bool
	require.NoError(t, db.Transaction(func(tx *gorm.DB) (err error) {
		had, err = service.ClearIn(context.Background(), tx)
		return err
	}))
	assert.True(t, had)

	configured, err := service.Configured(context.Background())
	require.NoError(t, err)
	assert.False(t, configured)
}

func TestSettingsRules(t *testing.T) {
	cases := map[string]struct {
		change func(*Input)
		field  string
	}{
		"no host":          {func(i *Input) { i.Host = "  " }, "host"},
		"host with a path": {func(i *Input) { i.Host = "smtp.example.com/x" }, "host"},
		"host with a port": {func(i *Input) { i.Host = "smtp.example.com:587" }, "host"},
		"port too low":     {func(i *Input) { i.Port = 0 }, "port"},
		"port too high":    {func(i *Input) { i.Port = 65536 }, "port"},
		"unknown security": {func(i *Input) { i.Security = "maybe" }, "security"},
		"bad from address": {func(i *Input) { i.FromAddress = "nobody" }, "fromAddress"},
		"named from":       {func(i *Input) { i.FromAddress = "Me <me@example.com>" }, "fromAddress"},
		"header injection": {func(i *Input) { i.FromName = "A\r\nBcc: x@example.com" }, "fromName"},
		"long username":    {func(i *Input) { i.Username = strings.Repeat("u", maxUsernameLength+1) }, "username"},
		"long password":    {func(i *Input) { i.Password = pointer(strings.Repeat("p", maxPasswordLength+1)) }, "password"},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			input := goodInput()
			test.change(&input)

			_, err := normalize(input)
			invalid, ok := validation.As(err)
			require.True(t, ok, "got %v", err)
			assert.Equal(t, test.field, invalid.Field)
		})
	}
}

func TestCheckAddress(t *testing.T) {
	for _, good := range []string{"a@example.com", "first.last+tag@sub.example.co"} {
		assert.NoError(t, CheckAddress(good), good)
	}
	for _, bad := range []string{"", "nobody", "@example.com", "a@", "Me <a@example.com>", "a@example.com\r\nBcc: b@example.com", strings.Repeat("a", maxAddressLength) + "@example.com"} {
		assert.Error(t, CheckAddress(bad), bad)
	}
}

// localSettings saves settings pointing at a test mail server and returns a
// service that trusts its certificate.
func localSettings(t *testing.T, server *mailtest.Server, input Input) *Service {
	t.Helper()

	db := storetest.Open(t, "sqlite")
	service := NewService(db, testBox(t, "a"), discard())
	service.dial = trusting(server.Roots)

	input.Host = "127.0.0.1"
	input.Port = server.Port
	save(t, service, db, input)

	return service
}

// trusting dials as the real dialer does, trusting only the test certificate.
func trusting(roots *x509.CertPool) dialer {
	return func(ctx context.Context, settings Settings) (net.Conn, *tls.Config, error) {
		config := &tls.Config{ServerName: settings.Host, RootCAs: roots, MinVersion: tls.VersionTLS12}
		address := net.JoinHostPort(settings.Host, strconv.Itoa(settings.Port))
		if settings.Security == ImplicitTLS {
			conn, err := (&tls.Dialer{Config: config}).DialContext(ctx, "tcp", address)
			return conn, config, err
		}
		conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", address)
		return conn, config, err
	}
}

func TestSendThroughStartTLSWithALogin(t *testing.T) {
	server := mailtest.Start(t, mailtest.Options{Username: "mailer", Password: "the-smtp-password", StartTLS: true})
	service := localSettings(t, server, goodInput())

	require.NoError(t, service.Send(context.Background(), "player@example.com", "Your code", "Your code is 123456.\nIt ends soon."))

	messages := server.Messages()
	require.Len(t, messages, 1)
	assert.Equal(t, "noreply@example.com", messages[0].From)
	assert.Equal(t, "player@example.com", messages[0].To)
	assert.Contains(t, messages[0].Data, "Subject: Your code\r\n")
	assert.Contains(t, messages[0].Data, `From: "Duel Masters" <noreply@example.com>`)
	assert.Contains(t, messages[0].Data, "\r\n\r\nYour code is 123456.\r\nIt ends soon.\r\n")
}

func TestSendThroughImplicitTLS(t *testing.T) {
	server := mailtest.Start(t, mailtest.Options{Username: "mailer", Password: "the-smtp-password", ImplicitTLS: true})
	input := goodInput()
	input.Security = ImplicitTLS
	service := localSettings(t, server, input)

	require.NoError(t, service.Send(context.Background(), "player@example.com", "s", "b"))
	assert.Len(t, server.Messages(), 1)
}

func TestSendLogsInWithLoginWhereThatIsAllThereIs(t *testing.T) {
	server := mailtest.Start(t, mailtest.Options{Username: "mailer", Password: "the-smtp-password", Auth: "LOGIN", StartTLS: true})
	service := localSettings(t, server, goodInput())

	require.NoError(t, service.Send(context.Background(), "player@example.com", "s", "b"))
	assert.Len(t, server.Messages(), 1)
}

func TestSendWithoutEncryptionToALocalServer(t *testing.T) {
	server := mailtest.Start(t, mailtest.Options{})
	input := goodInput()
	input.Security = NoSecurity
	input.Username = ""
	input.Password = pointer("")
	service := localSettings(t, server, input)

	require.NoError(t, service.Send(context.Background(), "player@example.com", "s", "b"))
	assert.Len(t, server.Messages(), 1)
}

func TestSendFailures(t *testing.T) {
	t.Run("STARTTLS asked for but not offered sends nothing", func(t *testing.T) {
		server := mailtest.Start(t, mailtest.Options{Username: "mailer", Password: "the-smtp-password"})
		service := localSettings(t, server, goodInput())

		assert.ErrorContains(t, service.Send(context.Background(), "player@example.com", "s", "b"), "STARTTLS")
		assert.Empty(t, server.Messages())
	})

	t.Run("a certificate it should not trust", func(t *testing.T) {
		server := mailtest.Start(t, mailtest.Options{Username: "mailer", Password: "the-smtp-password", StartTLS: true})
		service := localSettings(t, server, goodInput())
		service.dial = dialServer

		assert.ErrorContains(t, service.Send(context.Background(), "player@example.com", "s", "b"), "encrypting")
		assert.Empty(t, server.Messages())
	})

	t.Run("a wrong login", func(t *testing.T) {
		server := mailtest.Start(t, mailtest.Options{Username: "mailer", Password: "other", StartTLS: true})
		service := localSettings(t, server, goodInput())

		err := service.Send(context.Background(), "player@example.com", "s", "b")
		assert.ErrorContains(t, err, "logging in")
		assert.NotContains(t, err.Error(), "the-smtp-password")
	})

	t.Run("a refused recipient", func(t *testing.T) {
		server := mailtest.Start(t, mailtest.Options{Username: "mailer", Password: "the-smtp-password", StartTLS: true, RefuseRecipient: true})
		service := localSettings(t, server, goodInput())

		assert.ErrorContains(t, service.Send(context.Background(), "player@example.com", "s", "b"), "recipient")
	})

	t.Run("a bad recipient never reaches the server", func(t *testing.T) {
		server := mailtest.Start(t, mailtest.Options{Username: "mailer", Password: "the-smtp-password", StartTLS: true})
		service := localSettings(t, server, goodInput())

		assert.Error(t, service.Send(context.Background(), "a@example.com\r\nBcc: b@example.com", "s", "b"))
		assert.Empty(t, server.Messages())
	})

	t.Run("nothing listening", func(t *testing.T) {
		db := storetest.Open(t, "sqlite")
		service := NewService(db, testBox(t, "a"), discard())
		input := goodInput()
		input.Host = "127.0.0.1"
		input.Port = 1
		save(t, service, db, input)

		assert.ErrorContains(t, service.Send(context.Background(), "player@example.com", "s", "b"), "connecting")
	})

	t.Run("a password in the clear to another machine is refused", func(t *testing.T) {
		server := mailtest.Start(t, mailtest.Options{Username: "mailer", Password: "the-smtp-password"})
		db := storetest.Open(t, "sqlite")
		service := NewService(db, testBox(t, "a"), discard())
		// Reached as a name other than this machine's, without encryption.
		service.dial = func(ctx context.Context, _ Settings) (net.Conn, *tls.Config, error) {
			conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(server.Port)))
			return conn, nil, err
		}
		input := goodInput()
		input.Security = NoSecurity
		save(t, service, db, input)

		err := service.Send(context.Background(), "player@example.com", "s", "b")
		assert.ErrorContains(t, err, "logging in")
		assert.Empty(t, server.Messages())
	})
}

func TestComposeMessageEncodesWhatIsNotPlain(t *testing.T) {
	message, err := composeMessage(Settings{FromAddress: "noreply@example.com", FromName: "Dúel"}, "player@example.com", "Código", "Hola", testNow)
	require.NoError(t, err)

	text := string(message)
	assert.NotContains(t, text, "Dúel\"")
	assert.Contains(t, text, "Subject: =?utf-8?q?C=C3=B3digo?=\r\n")
	assert.Contains(t, text, "Message-ID: <")
	assert.Contains(t, text, "@example.com>\r\n")
}
