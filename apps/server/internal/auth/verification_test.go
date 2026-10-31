package auth_test

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/auth"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/store/storetest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/users"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/validation"
)

// sentMail is one email a test sent.
type sentMail struct {
	to, subject, body string
}

// fakeMailer keeps what it is asked to send.
type fakeMailer struct {
	mu         sync.Mutex
	configured bool
	failWith   error
	sent       []sentMail
}

func (m *fakeMailer) Configured(context.Context) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.configured, nil
}

func (m *fakeMailer) Send(_ context.Context, to, subject, body string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failWith != nil {
		return m.failWith
	}
	m.sent = append(m.sent, sentMail{to, subject, body})
	return nil
}

var codePattern = regexp.MustCompile(`\b\d{6}\b`)

// lastCode is the code in the last email sent.
func (m *fakeMailer) lastCode(t *testing.T) string {
	t.Helper()

	m.mu.Lock()
	defer m.mu.Unlock()

	require.NotEmpty(t, m.sent, "no email was sent")
	code := codePattern.FindString(m.sent[len(m.sent)-1].body)
	require.NotEmpty(t, code, "no code in the email")
	return code
}

func (m *fakeMailer) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return len(m.sent)
}

// newPlayer signs up a second account, the first being the admin, and leaves it
// unconfirmed.
func newPlayer(t *testing.T, f fixture) (users.User, auth.Session) {
	t.Helper()

	f.signUp(t, "Admin")
	player, session, err := f.auth.SignUp(context.Background(), "Shobu", "Shobu@Example.test", password)
	require.NoError(t, err)
	require.False(t, player.IsVerified())
	return player, session
}

// fresh reads an account again.
func fresh(t *testing.T, f fixture, id string) users.User {
	t.Helper()

	user, err := f.users.ByID(context.Background(), id)
	require.NoError(t, err)
	return user
}

func TestTheFirstAccountNeedsNoCode(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db)
		f.mail.configured = false

		admin, session, err := f.auth.SignUp(context.Background(), "Admin", "admin@example.test", password)
		require.NoError(t, err)

		assert.True(t, admin.IsAdmin())
		assert.True(t, admin.IsVerified())
		assert.Equal(t, "admin@example.test", admin.Email)
		assert.Zero(t, f.mail.count())

		_, err = f.auth.Authenticate(context.Background(), session.Token)
		assert.NoError(t, err)
	})
}

func TestSignUpSendsACodeAndKeepsThePlayerOut(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db)
		player, session := newPlayer(t, f)

		require.Equal(t, 1, f.mail.count())
		assert.Equal(t, "Shobu@Example.test", f.mail.sent[0].to)
		assert.Equal(t, "Shobu@Example.test", player.Email)
		assert.Equal(t, "shobu@example.test", player.EmailKey)

		_, err := f.auth.Authenticate(context.Background(), session.Token)
		assert.ErrorIs(t, err, auth.ErrNoSession, "kept out of the site")

		same, err := f.auth.AuthenticateUnconfirmed(context.Background(), session.Token)
		require.NoError(t, err)
		assert.Equal(t, player.ID, same.ID, "but can confirm")

		confirmed, err := f.auth.ConfirmCode(context.Background(), player, f.mail.lastCode(t))
		require.NoError(t, err)
		assert.True(t, confirmed.IsVerified())
		assert.Nil(t, confirmed.VerifyBy, "no longer due for deletion")

		_, err = f.auth.Authenticate(context.Background(), session.Token)
		assert.NoError(t, err, "let in")
	})
}

func TestAddressesAreUniqueAndChecked(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db)
		ctx := context.Background()
		newPlayer(t, f)

		_, _, err := f.auth.SignUp(ctx, "Kokujo", "SHOBU@example.TEST", password)
		assert.ErrorIs(t, err, users.ErrEmailTaken, "in any case")

		for _, bad := range []string{"", "nobody", "a@", "Me <me@example.test>", "a@example.test\r\nBcc: b@example.test"} {
			_, _, err := f.auth.SignUp(ctx, "Kokujo", bad, password)
			invalid, ok := validation.As(err)
			require.True(t, ok, "%q: %v", bad, err)
			assert.Equal(t, "email", invalid.Field)
		}
	})
}

func TestCodeRules(t *testing.T) {
	ctx := context.Background()

	t.Run("spaces in a code are forgiven", func(t *testing.T) {
		f := newFixture(t, storetest.Open(t, "sqlite"))
		player, _ := newPlayer(t, f)
		code := f.mail.lastCode(t)

		_, err := f.auth.ConfirmCode(ctx, player, " "+code[:3]+" "+code[3:]+" ")
		assert.NoError(t, err)
	})

	t.Run("something that is not a code costs no attempt", func(t *testing.T) {
		f := newFixture(t, storetest.Open(t, "sqlite"))
		player, _ := newPlayer(t, f)

		for range 10 {
			_, err := f.auth.ConfirmCode(ctx, player, "abc")
			_, invalid := validation.As(err)
			assert.True(t, invalid)
		}

		_, err := f.auth.ConfirmCode(ctx, player, f.mail.lastCode(t))
		assert.NoError(t, err)
	})

	t.Run("a wrong code is refused", func(t *testing.T) {
		f := newFixture(t, storetest.Open(t, "sqlite"))
		player, _ := newPlayer(t, f)

		_, err := f.auth.ConfirmCode(ctx, player, wrongCode(f.mail.lastCode(t)))
		assert.ErrorIs(t, err, auth.ErrWrongCode)
		assert.False(t, fresh(t, f, player.ID).IsVerified())
	})

	t.Run("too many wrong codes use it up", func(t *testing.T) {
		f := newFixture(t, storetest.Open(t, "sqlite"))
		player, _ := newPlayer(t, f)
		code := f.mail.lastCode(t)

		for range 5 {
			_, err := f.auth.ConfirmCode(ctx, player, wrongCode(code))
			assert.ErrorIs(t, err, auth.ErrWrongCode)
		}

		_, err := f.auth.ConfirmCode(ctx, player, code)
		assert.ErrorIs(t, err, auth.ErrCodeExpired, "even the right one no longer works")

		f.clock.advance(61 * time.Second)
		require.NoError(t, f.auth.SendCode(ctx, player))
		_, err = f.auth.ConfirmCode(ctx, player, f.mail.lastCode(t))
		assert.NoError(t, err, "a new one does")
	})

	t.Run("a code stops working", func(t *testing.T) {
		f := newFixture(t, storetest.Open(t, "sqlite"))
		player, _ := newPlayer(t, f)

		f.clock.advance(14 * time.Minute)
		_, err := f.auth.ConfirmCode(ctx, player, wrongCode(f.mail.lastCode(t)))
		assert.ErrorIs(t, err, auth.ErrWrongCode, "still alive")

		f.clock.advance(2 * time.Minute)
		_, err = f.auth.ConfirmCode(ctx, player, f.mail.lastCode(t))
		assert.ErrorIs(t, err, auth.ErrCodeExpired)
	})

	t.Run("a new code replaces the old", func(t *testing.T) {
		f := newFixture(t, storetest.Open(t, "sqlite"))
		player, _ := newPlayer(t, f)
		old := f.mail.lastCode(t)

		f.clock.advance(61 * time.Second)
		require.NoError(t, f.auth.SendCode(ctx, player))
		newest := f.mail.lastCode(t)
		if old == newest {
			t.Skip("the two random codes happened to match")
		}

		_, err := f.auth.ConfirmCode(ctx, player, old)
		assert.ErrorIs(t, err, auth.ErrWrongCode)
	})

	t.Run("no code sent yet", func(t *testing.T) {
		f := newFixture(t, storetest.Open(t, "sqlite"))
		f.signUp(t, "Admin")
		f.mail.configured = false
		player, _, err := f.auth.SignUp(ctx, "Shobu", "shobu@example.test", password)
		require.NoError(t, err)

		_, err = f.auth.ConfirmCode(ctx, player, "123456")
		assert.ErrorIs(t, err, auth.ErrCodeExpired)
	})

	t.Run("confirming twice", func(t *testing.T) {
		f := newFixture(t, storetest.Open(t, "sqlite"))
		player, _ := newPlayer(t, f)

		confirmed, err := f.auth.ConfirmCode(ctx, player, f.mail.lastCode(t))
		require.NoError(t, err)

		_, err = f.auth.ConfirmCode(ctx, confirmed, "123456")
		assert.ErrorIs(t, err, users.ErrEmailConfirmed)
		assert.ErrorIs(t, f.auth.SendCode(ctx, confirmed), users.ErrEmailConfirmed)
	})

	t.Run("the code is not stored as it was sent", func(t *testing.T) {
		f := newFixture(t, storetest.Open(t, "sqlite"))
		player, _ := newPlayer(t, f)
		code := f.mail.lastCode(t)

		var stored string
		require.NoError(t, f.db.Table("email_verifications").Where("user_id = ?", player.ID).Select("code_hash").Scan(&stored).Error)
		assert.NotEmpty(t, stored)
		assert.False(t, strings.Contains(stored, code))
	})
}

// wrongCode is a code other than the one given.
func wrongCode(code string) string {
	if code == "000000" {
		return "111111"
	}
	return "000000"
}

func TestAskingForCodes(t *testing.T) {
	ctx := context.Background()

	t.Run("not again too soon", func(t *testing.T) {
		f := newFixture(t, storetest.Open(t, "sqlite"))
		player, _ := newPlayer(t, f)

		var tooSoon *auth.ResendTooSoonError
		require.ErrorAs(t, f.auth.SendCode(ctx, player), &tooSoon)
		assert.Equal(t, time.Minute, tooSoon.RetryAfter)
		assert.Equal(t, 1, f.mail.count())

		f.clock.advance(time.Minute)
		require.NoError(t, f.auth.SendCode(ctx, player))
		assert.Equal(t, 2, f.mail.count())
	})

	t.Run("while there is no mail server the player waits", func(t *testing.T) {
		f := newFixture(t, storetest.Open(t, "sqlite"))
		f.signUp(t, "Admin")
		f.mail.configured = false

		player, _, err := f.auth.SignUp(ctx, "Shobu", "shobu@example.test", password)
		require.NoError(t, err, "signing up still works")
		assert.Zero(t, f.mail.count())
		assert.Nil(t, fresh(t, f, player.ID).VerifyBy, "no clock is running")

		assert.ErrorIs(t, f.auth.SendCode(ctx, player), auth.ErrMailUnavailable)

		status, err := f.auth.VerificationOf(ctx, player)
		require.NoError(t, err)
		assert.False(t, status.MailWorks)
		assert.Nil(t, status.CodeSentAt)
		assert.Nil(t, status.DeleteAt)

		f.mail.configured = true
		require.NoError(t, f.auth.SendCode(ctx, player), "once one is set up, the player can ask")
		_, err = f.auth.ConfirmCode(ctx, player, f.mail.lastCode(t))
		assert.NoError(t, err)
	})

	t.Run("a failed send can be retried at once and does not start the clock", func(t *testing.T) {
		f := newFixture(t, storetest.Open(t, "sqlite"))
		f.signUp(t, "Admin")
		f.mail.failWith = errors.New("the server said no")

		player, _, err := f.auth.SignUp(ctx, "Shobu", "shobu@example.test", password)
		require.NoError(t, err, "signing up still works")
		assert.Nil(t, fresh(t, f, player.ID).VerifyBy)

		var failed *auth.MailFailedError
		require.ErrorAs(t, f.auth.SendCode(ctx, player), &failed)

		f.mail.failWith = nil
		require.NoError(t, f.auth.SendCode(ctx, player))
		assert.NotNil(t, fresh(t, f, player.ID).VerifyBy)
	})

	t.Run("how it stands", func(t *testing.T) {
		f := newFixture(t, storetest.Open(t, "sqlite"))
		player, _ := newPlayer(t, f)

		status, err := f.auth.VerificationOf(ctx, player)
		require.NoError(t, err)
		assert.Equal(t, "Shobu@Example.test", status.Email)
		assert.True(t, status.MailWorks)
		require.NotNil(t, status.CodeSentAt)
		require.NotNil(t, status.CodeExpiresAt)
		assert.Equal(t, 15*time.Minute, status.CodeExpiresAt.Sub(*status.CodeSentAt))
		assert.Equal(t, time.Minute, status.ResendIn)
		require.NotNil(t, status.DeleteAt)

		f.clock.advance(2 * time.Minute)
		status, err = f.auth.VerificationOf(ctx, player)
		require.NoError(t, err)
		assert.Zero(t, status.ResendIn)
	})
}

func TestCorrectingAnAddress(t *testing.T) {
	ctx := context.Background()

	t.Run("sends a code to the new one, and the old code stops working", func(t *testing.T) {
		f := newFixture(t, storetest.Open(t, "sqlite"))
		player, _ := newPlayer(t, f)
		old := f.mail.lastCode(t)

		f.clock.advance(time.Minute)
		changed, sent, err := f.auth.ChangeEmail(ctx, player, " right@example.test ")
		require.NoError(t, err)
		assert.True(t, sent)
		assert.Equal(t, "right@example.test", changed.Email)
		assert.Equal(t, "right@example.test", f.mail.sent[len(f.mail.sent)-1].to)

		if newest := f.mail.lastCode(t); old != newest {
			_, err = f.auth.ConfirmCode(ctx, changed, old)
			assert.ErrorIs(t, err, auth.ErrWrongCode)
		}
		_, err = f.auth.ConfirmCode(ctx, changed, f.mail.lastCode(t))
		assert.NoError(t, err)
	})

	t.Run("cannot be used to send mail to address after address", func(t *testing.T) {
		f := newFixture(t, storetest.Open(t, "sqlite"))
		player, _ := newPlayer(t, f)

		_, _, err := f.auth.ChangeEmail(ctx, player, "right@example.test")
		var tooSoon *auth.ResendTooSoonError
		assert.ErrorAs(t, err, &tooSoon)
		assert.Equal(t, "Shobu@Example.test", fresh(t, f, player.ID).Email)
		assert.Equal(t, 1, f.mail.count())
	})

	t.Run("not to an address in use, or a confirmed account", func(t *testing.T) {
		f := newFixture(t, storetest.Open(t, "sqlite"))
		player, _ := newPlayer(t, f)
		f.clock.advance(time.Minute)

		_, _, err := f.auth.ChangeEmail(ctx, player, "ADMIN@example.test")
		assert.ErrorIs(t, err, users.ErrEmailTaken)

		_, _, err = f.auth.ChangeEmail(ctx, player, "nonsense")
		_, invalid := validation.As(err)
		assert.True(t, invalid)

		confirmed, err := f.auth.ConfirmCode(ctx, player, f.mail.lastCode(t))
		require.NoError(t, err)
		_, _, err = f.auth.ChangeEmail(ctx, confirmed, "other@example.test")
		assert.ErrorIs(t, err, users.ErrEmailConfirmed)
	})

	t.Run("while there is no mail server the address still changes", func(t *testing.T) {
		f := newFixture(t, storetest.Open(t, "sqlite"))
		f.signUp(t, "Admin")
		f.mail.configured = false
		player, _, err := f.auth.SignUp(ctx, "Shobu", "wrong@example.test", password)
		require.NoError(t, err)

		changed, sent, err := f.auth.ChangeEmail(ctx, player, "right@example.test")
		require.NoError(t, err)
		assert.False(t, sent)
		assert.Equal(t, "right@example.test", changed.Email)
	})
}

func TestUnconfirmedAccountsAreDeleted(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db)
		ctx := context.Background()
		player, session := newPlayer(t, f)

		// One who has never been sent a code waits, however long.
		f.mail.configured = false
		waiting, _, err := f.auth.SignUp(ctx, "Waiting", "waiting@example.test", password)
		require.NoError(t, err)
		f.mail.configured = true

		// One who confirmed stays.
		confirming, _, err := f.auth.SignUp(ctx, "Confirming", "confirming@example.test", password)
		require.NoError(t, err)
		_, err = f.auth.ConfirmCode(ctx, confirming, f.mail.lastCode(t))
		require.NoError(t, err)

		f.clock.advance(23 * time.Hour)
		deleted, err := f.auth.DeleteUnconfirmedAccounts(ctx)
		require.NoError(t, err)
		assert.Zero(t, deleted, "not yet")

		f.clock.advance(time.Hour)
		deleted, err = f.auth.DeleteUnconfirmedAccounts(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(1), deleted)

		_, err = f.users.ByID(ctx, player.ID)
		assert.ErrorIs(t, err, users.ErrNotFound)
		_, err = f.auth.AuthenticateUnconfirmed(ctx, session.Token)
		assert.ErrorIs(t, err, auth.ErrNoSession, "its session went with it")

		fresh(t, f, waiting.ID)
		fresh(t, f, confirming.ID)

		_, _, err = f.auth.SignUp(ctx, "Shobu", "Shobu@Example.test", password)
		assert.NoError(t, err, "its name and address are free again")
	})
}

func TestAnAskingForANewCodeDoesNotPutOffDeletion(t *testing.T) {
	f := newFixture(t, storetest.Open(t, "sqlite"))
	ctx := context.Background()
	player, _ := newPlayer(t, f)
	deadline := fresh(t, f, player.ID).VerifyBy

	f.clock.advance(time.Hour)
	require.NoError(t, f.auth.SendCode(ctx, player))

	assert.True(t, deadline.Equal(*fresh(t, f, player.ID).VerifyBy))
}
