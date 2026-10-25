package auth_test

import (
	"context"
	"encoding/base64"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/argon2"
	"gorm.io/gorm"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/auth"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/settings"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/store/storetest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/users"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/validation"
)

const password = "correct horse battery staple"

// clock is a settable time for tests.
type clock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *clock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *clock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

type fixture struct {
	auth     *auth.Service
	users    *users.Service
	settings *settings.Service
	clock    *clock
	db       *gorm.DB
}

func newFixture(t *testing.T, db *gorm.DB) fixture {
	t.Helper()

	siteSettings, err := settings.NewService(context.Background(), db, storetest.Logger(), auth.Settings()...)
	require.NoError(t, err)

	accounts := users.NewService(db, storetest.Logger())
	c := &clock{t: time.Now().UTC()}

	return fixture{
		auth:     auth.NewService(db, accounts, siteSettings, auth.NewPasswordHasher(4), c.now, storetest.Logger()),
		users:    accounts,
		settings: siteSettings,
		clock:    c,
		db:       db,
	}
}

func (f fixture) signUp(t *testing.T, username string) (users.User, auth.Session) {
	t.Helper()

	user, session, err := f.auth.SignUp(context.Background(), username, password)
	require.NoError(t, err)

	return user, session
}

func TestSignUpCreatesAnAccountAndLogsItIn(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db)
		ctx := context.Background()

		user, session, err := f.auth.SignUp(ctx, " Shobu ", password)
		require.NoError(t, err)
		assert.Equal(t, "Shobu", user.Username)
		assert.Equal(t, users.RoleAdmin, user.Role, "the first account is the admin")
		assert.NotEmpty(t, session.Token)
		assert.WithinDuration(t, f.clock.now().Add(auth.SessionDuration), session.ExpiresAt, time.Second)

		authenticated, err := f.auth.Authenticate(ctx, session.Token)
		require.NoError(t, err)
		assert.Equal(t, user.ID, authenticated.ID)
		assert.Equal(t, "Shobu", authenticated.Username, "the account's own columns are read, not the session's")
		assert.Equal(t, user.CreatedAt.UTC(), authenticated.CreatedAt.UTC())

		assert.NotContains(t, user.PasswordHash, password)
		assert.True(t, strings.HasPrefix(user.PasswordHash, "$argon2id$"))
	})
}

func TestSessionTokensAreStoredOnlyAsHashes(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db)
		_, session := f.signUp(t, "Shobu")

		var ids []string
		require.NoError(t, db.Raw("SELECT id FROM sessions").Scan(&ids).Error)
		require.Len(t, ids, 1)
		assert.NotEqual(t, session.Token, ids[0])
		assert.Len(t, ids[0], 64, "a SHA-256 in hexadecimal")
	})
}

func TestSignUpChecksItsInput(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db)
		ctx := context.Background()

		for _, bad := range []struct{ username, password, field string }{
			{"no", password, "username"},
			{"bad name", password, "username"},
			{"Shobu", "short", "password"},
			{"Shobu", strings.Repeat("x", auth.PasswordMaxLength+1), "password"},
		} {
			_, _, err := f.auth.SignUp(ctx, bad.username, bad.password)
			invalid, ok := validation.As(err)
			require.True(t, ok, "%q/%q: got %v", bad.username, bad.password, err)
			assert.Equal(t, bad.field, invalid.Field)
		}

		_, _, err := f.auth.SignUp(ctx, "Shobu", strings.Repeat("é", auth.PasswordMinLength))
		assert.NoError(t, err, "password length counts characters, not bytes")
	})
}

func TestSignUpRefusesATakenName(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db)
		f.signUp(t, "Shobu")

		_, _, err := f.auth.SignUp(context.Background(), "SHOBU", password)
		assert.ErrorIs(t, err, users.ErrUsernameTaken)
	})
}

func TestSignUpRespectsTheAdminSetting(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db)
		ctx := context.Background()
		_, adminSession := f.signUp(t, "Admin")

		_, err := f.settings.Set(ctx, auth.SignupsOpen.Key, "false")
		require.NoError(t, err)

		_, _, err = f.auth.SignUp(ctx, "Newcomer", password)
		assert.ErrorIs(t, err, auth.ErrSignupsClosed)

		_, _, err = f.auth.LogIn(ctx, "Admin", password)
		assert.NoError(t, err, "existing players can still log in")
		_, err = f.auth.Authenticate(ctx, adminSession.Token)
		assert.NoError(t, err)
	})
}

func TestLogInStartsASession(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db)
		ctx := context.Background()
		created, _ := f.signUp(t, "Shobu")

		user, session, err := f.auth.LogIn(ctx, "shobu", password)
		require.NoError(t, err, "names are matched in any case")
		assert.Equal(t, created.ID, user.ID)

		authenticated, err := f.auth.Authenticate(ctx, session.Token)
		require.NoError(t, err)
		assert.Equal(t, created.ID, authenticated.ID)
	})
}

func TestLogInDoesNotSayWhichPartWasWrong(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db)
		ctx := context.Background()
		f.signUp(t, "Shobu")

		_, _, wrongPassword := f.auth.LogIn(ctx, "Shobu", "not the password")
		_, _, unknownName := f.auth.LogIn(ctx, "Nobody", password)
		_, _, overlong := f.auth.LogIn(ctx, "Shobu", strings.Repeat("x", auth.PasswordMaxLength+1))

		assert.ErrorIs(t, wrongPassword, auth.ErrInvalidCredentials)
		assert.ErrorIs(t, unknownName, auth.ErrInvalidCredentials)
		assert.ErrorIs(t, overlong, auth.ErrInvalidCredentials)
	})
}

func TestLogInRefusesAnAccountAfterTooManyWrongPasswords(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db)
		ctx := context.Background()
		f.signUp(t, "Shobu")
		f.signUp(t, "Kokujo")

		for range 5 {
			_, _, err := f.auth.LogIn(ctx, "Shobu", "wrong")
			require.ErrorIs(t, err, auth.ErrInvalidCredentials)
		}

		_, _, err := f.auth.LogIn(ctx, "SHOBU", password)
		var tooMany *auth.TooManyAttemptsError
		require.ErrorAs(t, err, &tooMany, "even the right password is refused for now, in any case of the name")
		assert.Equal(t, 15*time.Minute, tooMany.RetryAfter)

		_, _, err = f.auth.LogIn(ctx, "Kokujo", password)
		assert.NoError(t, err, "other accounts are not affected")

		f.clock.advance(15 * time.Minute)
		_, _, err = f.auth.LogIn(ctx, "Shobu", password)
		assert.NoError(t, err, "the account accepts logins again once the window ends")
	})
}

func TestLogInCountsGuessesAtUnknownNamesToo(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db)
		ctx := context.Background()

		for range 5 {
			_, _, err := f.auth.LogIn(ctx, "Nobody", password)
			require.ErrorIs(t, err, auth.ErrInvalidCredentials)
		}

		_, _, err := f.auth.LogIn(ctx, "Nobody", password)
		var tooMany *auth.TooManyAttemptsError
		assert.ErrorAs(t, err, &tooMany, "an unknown name is limited like a real one, so the two look alike")
	})
}

func TestASuccessfulLogInClearsTheWrongAttempts(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db)
		ctx := context.Background()
		f.signUp(t, "Shobu")

		for range 4 {
			_, _, _ = f.auth.LogIn(ctx, "Shobu", "wrong")
		}
		_, _, err := f.auth.LogIn(ctx, "Shobu", password)
		require.NoError(t, err)

		for range 4 {
			_, _, _ = f.auth.LogIn(ctx, "Shobu", "wrong")
		}
		_, _, err = f.auth.LogIn(ctx, "Shobu", password)
		assert.NoError(t, err)
	})
}

func TestLogInUpgradesAnOldPasswordHash(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db)
		ctx := context.Background()

		salt := []byte("0123456789abcdef")
		key := argon2.IDKey([]byte(password), salt, 1, 8*1024, 1, 32)
		old := "$argon2id$v=19$m=8192,t=1,p=1$" + base64.RawStdEncoding.EncodeToString(salt) + "$" + base64.RawStdEncoding.EncodeToString(key)
		user, err := f.users.Create(ctx, "Veteran", old)
		require.NoError(t, err)

		_, _, err = f.auth.LogIn(ctx, "Veteran", password)
		require.NoError(t, err)

		upgraded, err := f.users.ByID(ctx, user.ID)
		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(upgraded.PasswordHash, "$argon2id$v=19$m=19456,t=2,p=1$"))
		assert.False(t, auth.NeedsRehash(upgraded.PasswordHash))

		_, _, err = f.auth.LogIn(ctx, "Veteran", password)
		assert.NoError(t, err, "the upgraded hash still matches the password")
	})
}

func TestAnAccountWithAnUnreadableHashCannotLogIn(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db)
		_, err := f.users.Create(context.Background(), "Broken", "not a hash")
		require.NoError(t, err)

		_, _, err = f.auth.LogIn(context.Background(), "Broken", password)
		assert.ErrorIs(t, err, auth.ErrInvalidCredentials)
	})
}

func TestAuthenticateRefusesUnknownAndExpiredSessions(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db)
		ctx := context.Background()
		_, session := f.signUp(t, "Shobu")

		_, err := f.auth.Authenticate(ctx, "")
		assert.ErrorIs(t, err, auth.ErrNoSession)
		_, err = f.auth.Authenticate(ctx, "made-up-token")
		assert.ErrorIs(t, err, auth.ErrNoSession)

		f.clock.advance(auth.SessionDuration - time.Minute)
		_, err = f.auth.Authenticate(ctx, session.Token)
		assert.NoError(t, err, "still valid just before it expires")

		f.clock.advance(time.Minute)
		_, err = f.auth.Authenticate(ctx, session.Token)
		assert.ErrorIs(t, err, auth.ErrNoSession, "expired")
	})
}

func TestLogOutEndsOnlyThatSession(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db)
		ctx := context.Background()
		_, laptop := f.signUp(t, "Shobu")
		_, _, err := f.auth.LogIn(ctx, "Shobu", password)
		require.NoError(t, err)
		_, phone, err := f.auth.LogIn(ctx, "Shobu", password)
		require.NoError(t, err)

		require.NoError(t, f.auth.LogOut(ctx, laptop.Token))

		_, err = f.auth.Authenticate(ctx, laptop.Token)
		assert.ErrorIs(t, err, auth.ErrNoSession)
		_, err = f.auth.Authenticate(ctx, phone.Token)
		assert.NoError(t, err)

		assert.NoError(t, f.auth.LogOut(ctx, laptop.Token), "logging out twice is harmless")
	})
}

func TestChangePasswordReplacesItAndEndsOtherSessions(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db)
		ctx := context.Background()
		user, laptop := f.signUp(t, "Shobu")
		_, phone, err := f.auth.LogIn(ctx, "Shobu", password)
		require.NoError(t, err)
		other, otherSession := f.signUp(t, "Kokujo")

		require.NoError(t, f.auth.ChangePassword(ctx, user, laptop.Token, password, "a brand new password"))

		_, err = f.auth.Authenticate(ctx, laptop.Token)
		assert.NoError(t, err, "the session that changed it stays logged in")
		_, err = f.auth.Authenticate(ctx, phone.Token)
		assert.ErrorIs(t, err, auth.ErrNoSession, "every other session of the account ends")
		authenticated, err := f.auth.Authenticate(ctx, otherSession.Token)
		require.NoError(t, err, "other accounts keep their sessions")
		assert.Equal(t, other.ID, authenticated.ID)

		_, _, err = f.auth.LogIn(ctx, "Shobu", password)
		assert.ErrorIs(t, err, auth.ErrInvalidCredentials)
		_, _, err = f.auth.LogIn(ctx, "Shobu", "a brand new password")
		assert.NoError(t, err)
	})
}

func TestChangePasswordChecksTheCurrentOne(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db)
		ctx := context.Background()
		user, session := f.signUp(t, "Shobu")

		err := f.auth.ChangePassword(ctx, user, session.Token, "not the password", "a brand new password")
		assert.ErrorIs(t, err, auth.ErrWrongPassword)

		err = f.auth.ChangePassword(ctx, user, session.Token, password, "short")
		invalid, ok := validation.As(err)
		require.True(t, ok, "got %v", err)
		assert.Equal(t, "password", invalid.Field)

		for range 4 {
			_ = f.auth.ChangePassword(ctx, user, session.Token, "not the password", "a brand new password")
		}
		err = f.auth.ChangePassword(ctx, user, session.Token, password, "a brand new password")
		var tooMany *auth.TooManyAttemptsError
		assert.ErrorAs(t, err, &tooMany, "wrong current passwords count against the login limit")

		_, _, err = f.auth.LogIn(ctx, "Shobu", password)
		assert.ErrorAs(t, err, &tooMany, "and so do logins")
	})
}

func TestDeleteExpiredSessionsKeepsTheLiveOnes(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db)
		ctx := context.Background()
		_, old := f.signUp(t, "Shobu")

		f.clock.advance(auth.SessionDuration / 2)
		_, recent, err := f.auth.LogIn(ctx, "Shobu", password)
		require.NoError(t, err)

		f.clock.advance(auth.SessionDuration / 2)
		deleted, err := f.auth.DeleteExpiredSessions(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(1), deleted)

		_, err = f.auth.Authenticate(ctx, old.Token)
		assert.ErrorIs(t, err, auth.ErrNoSession)
		_, err = f.auth.Authenticate(ctx, recent.Token)
		assert.NoError(t, err)
	})
}

func TestABannedPlayerIsToldWhyOnlyWithTheRightPassword(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		f := newFixture(t, db)
		f.signUp(t, "Boss")
		player, _ := f.signUp(t, "Shobu")
		until := f.clock.now().Add(time.Hour)
		require.NoError(t, users.SetBanIn(db, player.ID, users.Ban{Penalty: users.Penalty{Since: f.clock.now(), Until: &until}, Reason: "Spam"}))

		_, _, err := f.auth.LogIn(ctx, "Shobu", "not the password")
		assert.ErrorIs(t, err, auth.ErrInvalidCredentials, "a wrong password says nothing of the ban")

		_, _, err = f.auth.LogIn(ctx, "Shobu", password)
		var banned *auth.BannedError
		require.ErrorAs(t, err, &banned)
		assert.Equal(t, "Spam", banned.Ban.Reason)
		require.NotNil(t, banned.Ban.Until)
		assert.WithinDuration(t, until, *banned.Ban.Until, time.Microsecond, "as stored, to the microsecond")

		f.clock.advance(time.Hour)
		_, _, err = f.auth.LogIn(ctx, "Shobu", password)
		assert.NoError(t, err, "the ban is over")
	})
}

func TestABannedPlayersSessionIsRefused(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		f := newFixture(t, db)
		f.signUp(t, "Boss")
		player, session := f.signUp(t, "Shobu")

		// A login that crossed with the ban could leave a session behind.
		require.NoError(t, users.SetBanIn(db, player.ID, users.Ban{Penalty: users.Penalty{Since: f.clock.now()}, Reason: "Cheating"}))

		_, err := f.auth.Authenticate(ctx, session.Token)
		assert.ErrorIs(t, err, auth.ErrNoSession)
	})
}

func TestEndSessionsInEndsEveryOneOfTheAccount(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		f := newFixture(t, db)
		_, other := f.signUp(t, "Boss")
		player, first := f.signUp(t, "Shobu")
		_, second, err := f.auth.LogIn(ctx, "Shobu", password)
		require.NoError(t, err)

		require.NoError(t, auth.EndSessionsIn(db, player.ID))

		for _, session := range []auth.Session{first, second} {
			_, err := f.auth.Authenticate(ctx, session.Token)
			assert.ErrorIs(t, err, auth.ErrNoSession)
		}
		_, err = f.auth.Authenticate(ctx, other.Token)
		assert.NoError(t, err, "other accounts stay logged in")
	})
}

func TestHashNewPasswordKeepsToTheRules(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db)

		_, err := f.auth.HashNewPassword(context.Background(), "short")
		invalid, ok := validation.As(err)
		require.True(t, ok, "got %v", err)
		assert.Equal(t, "password", invalid.Field)

		hash, err := f.auth.HashNewPassword(context.Background(), password)
		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(hash, "$argon2id$"))
	})
}

func TestForgetWrongPasswordsLetsAPlayerLogInAgainAtOnce(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		f := newFixture(t, db)
		f.signUp(t, "Shobu")
		for range 5 {
			_, _, err := f.auth.LogIn(ctx, "Shobu", "not the password")
			require.ErrorIs(t, err, auth.ErrInvalidCredentials)
		}
		_, _, err := f.auth.LogIn(ctx, "Shobu", password)
		var tooMany *auth.TooManyAttemptsError
		require.ErrorAs(t, err, &tooMany)

		f.auth.ForgetWrongPasswords("SHOBU")

		_, _, err = f.auth.LogIn(ctx, "Shobu", password)
		assert.NoError(t, err)
	})
}
