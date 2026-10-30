// Package auth signs players up and in: it hashes passwords, runs login
// sessions and slows down password guessing. Accounts themselves belong to
// package users.
package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/ratelimit"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/settings"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/users"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/validation"
)

// Password rules, the same as Triple Triad's. Lengths count characters.
const (
	PasswordMinLength = 8
	PasswordMaxLength = 128
)

// Wrong passwords allowed per account before logins to it are refused for the
// rest of the window.
const (
	loginAttempts = 5
	loginWindow   = 15 * time.Minute
)

// SignupsOpen is the admin setting that lets new players sign up.
var SignupsOpen = settings.Bool{
	Key:         "accounts.signups_open",
	Group:       "Accounts",
	Description: "Whether new players can sign up. Players who already have an account can still log in.",
	Default:     true,
}

// Settings are the settings this feature declares.
func Settings() []settings.Definition {
	return []settings.Definition{SignupsOpen}
}

var (
	// ErrSignupsClosed is returned when an admin has closed sign-ups.
	ErrSignupsClosed = errors.New("sign-ups are closed")
	// ErrInvalidCredentials is returned for a wrong name or password, without
	// saying which.
	ErrInvalidCredentials = errors.New("wrong username or password")
	// ErrNoSession is returned for a missing, unknown or expired session.
	ErrNoSession = errors.New("not logged in")
	// ErrWrongPassword is returned when the current password given to change
	// it is wrong.
	ErrWrongPassword = errors.New("wrong password")
)

// TooManyAttemptsError is returned while an account refuses logins after too
// many wrong passwords.
type TooManyAttemptsError struct {
	RetryAfter time.Duration
}

func (e *TooManyAttemptsError) Error() string {
	return fmt.Sprintf("too many failed attempts; try again in %s", e.RetryAfter.Round(time.Second))
}

// BannedError is returned to a banned player logging in with the right
// password. Only they learn of the ban: a wrong password says nothing.
type BannedError struct {
	Ban users.Ban
}

func (e *BannedError) Error() string {
	if e.Ban.Until == nil {
		return "this account is banned"
	}
	return fmt.Sprintf("this account is banned until %s", e.Ban.Until.Format(time.RFC3339))
}

// Service signs players up, in and out. It is safe for concurrent use.
type Service struct {
	db       *gorm.DB
	users    *users.Service
	settings *settings.Service
	hasher   *PasswordHasher
	logins   *ratelimit.Limiter
	now      func() time.Time
	logger   *slog.Logger
}

// NewService returns the service. now is the clock; pass time.Now outside
// tests.
func NewService(db *gorm.DB, accounts *users.Service, siteSettings *settings.Service, hasher *PasswordHasher, now func() time.Time, logger *slog.Logger) *Service {
	return &Service{
		db:       db,
		users:    accounts,
		settings: siteSettings,
		hasher:   hasher,
		logins:   ratelimit.New(loginAttempts, loginWindow, now),
		now:      now,
		logger:   logger,
	}
}

// ValidatePassword checks a new password against the rules.
func ValidatePassword(password string) error {
	switch length := utf8.RuneCountInString(password); {
	case length < PasswordMinLength:
		return validation.New("password", fmt.Sprintf("must be at least %d characters", PasswordMinLength))
	case length > PasswordMaxLength:
		return validation.New("password", fmt.Sprintf("must be at most %d characters", PasswordMaxLength))
	}

	return nil
}

// SignUp creates an account and logs it in. It returns ErrSignupsClosed while
// an admin has closed sign-ups, a *validation.Error for a name or password
// breaking the rules, and users.ErrUsernameTaken for a name in use.
func (s *Service) SignUp(ctx context.Context, username, password string) (users.User, Session, error) {
	if !s.settings.Bool(SignupsOpen) {
		return users.User{}, Session{}, ErrSignupsClosed
	}

	// Checked before hashing, which is the slow part.
	if _, _, err := users.NormalizeUsername(username); err != nil {
		return users.User{}, Session{}, err
	}
	if err := ValidatePassword(password); err != nil {
		return users.User{}, Session{}, err
	}

	hash, err := s.hasher.Hash(ctx, password)
	if err != nil {
		return users.User{}, Session{}, err
	}

	user, err := s.users.Create(ctx, username, hash)
	if err != nil {
		return users.User{}, Session{}, err
	}

	session, err := s.startSession(ctx, user.ID)
	if err != nil {
		return users.User{}, Session{}, err
	}

	return user, session, nil
}

// LogIn checks a name and password and starts a session. It returns
// ErrInvalidCredentials for a wrong name or password alike, taking as long
// either way, a *BannedError for a banned account's right password, and a
// *TooManyAttemptsError once the account has had too many wrong passwords in
// the window.
func (s *Service) LogIn(ctx context.Context, username, password string) (users.User, Session, error) {
	key := users.Key(username)
	if allowed, retryAfter := s.logins.Allowed(key); !allowed {
		return users.User{}, Session{}, &TooManyAttemptsError{RetryAfter: retryAfter}
	}

	user, ok, err := s.checkPassword(ctx, username, password)
	if err != nil {
		return users.User{}, Session{}, err
	}
	if !ok {
		s.logins.Record(key)
		return users.User{}, Session{}, ErrInvalidCredentials
	}
	s.logins.Reset(key)

	if ban, banned := user.BanAt(s.timestamp()); banned {
		return users.User{}, Session{}, &BannedError{Ban: ban}
	}

	s.upgradeHash(ctx, user, password)

	session, err := s.startSession(ctx, user.ID)
	if err != nil {
		return users.User{}, Session{}, err
	}

	return user, session, nil
}

// checkPassword finds the account and checks its password. ok is false for an
// unknown account or a wrong password; err is only for failures.
func (s *Service) checkPassword(ctx context.Context, username, password string) (user users.User, ok bool, err error) {
	if utf8.RuneCountInString(password) > PasswordMaxLength {
		// No account has a longer password, so nothing needs hashing.
		return users.User{}, false, nil
	}

	user, err = s.users.ByUsername(ctx, username)
	if err != nil && !errors.Is(err, users.ErrNotFound) {
		return users.User{}, false, err
	}
	// Nobody logs in to a computer opponent's account, which has no
	// password: it is refused as an unknown account is, taking as long.
	if errors.Is(err, users.ErrNotFound) || user.Computer {
		return users.User{}, false, s.hasher.VerifyNothing(ctx, password)
	}

	ok, err = s.hasher.Verify(ctx, user.PasswordHash, password)
	if err != nil {
		if errors.Is(err, errMalformedHash) {
			s.logger.ErrorContext(ctx, "an account's stored password hash is unreadable, so it cannot log in", "user_id", user.ID)
			return users.User{}, false, nil
		}
		return users.User{}, false, err
	}

	return user, ok, nil
}

// upgradeHash hashes the password again with the current settings when the
// stored hash was made with others. A failure is logged, not returned: the
// login itself succeeded.
func (s *Service) upgradeHash(ctx context.Context, user users.User, password string) {
	if !NeedsRehash(user.PasswordHash) {
		return
	}

	hash, err := s.hasher.Hash(ctx, password)
	if err == nil {
		err = s.users.SetPasswordHash(ctx, user.ID, hash)
	}
	if err != nil {
		s.logger.WarnContext(ctx, "could not upgrade a password hash to the current settings", "user_id", user.ID, "error", err)
	}
}

// Authenticate returns the account a session token belongs to. It returns
// ErrNoSession for a missing, unknown or expired token, and for a banned
// account's: a ban ends the account's sessions, and this also refuses one
// started by a login that crossed with the ban.
func (s *Service) Authenticate(ctx context.Context, token string) (users.User, error) {
	if token == "" {
		return users.User{}, ErrNoSession
	}

	now := s.timestamp()
	var user users.User
	// Only the account's columns are read: both tables have id and
	// created_at columns.
	err := s.db.WithContext(ctx).
		Select("users.*").
		Joins("JOIN sessions ON sessions.user_id = users.id").
		Where("sessions.id = ? AND sessions.expires_at > ?", sessionID(token), now).
		Where(users.NotBannedAt(now)).
		Take(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return users.User{}, ErrNoSession
	}
	if err != nil {
		return users.User{}, fmt.Errorf("checking a session: %w", err)
	}

	return user, nil
}

// LogOut ends a session. Ending one that does not exist is not an error.
func (s *Service) LogOut(ctx context.Context, token string) error {
	if err := s.db.WithContext(ctx).Delete(&sessionRecord{ID: sessionID(token)}).Error; err != nil {
		return fmt.Errorf("ending a session: %w", err)
	}

	return nil
}

// ChangePassword replaces a logged-in player's password after checking the
// current one, and ends every other session of the account, keeping the one
// identified by keepToken. It returns ErrWrongPassword for a wrong current
// password, a *TooManyAttemptsError after too many of them, and a
// *validation.Error for a new password breaking the rules.
func (s *Service) ChangePassword(ctx context.Context, user users.User, keepToken, current, replacement string) error {
	if err := ValidatePassword(replacement); err != nil {
		return err
	}

	// Wrong current passwords count against the same limit as logins, so a
	// borrowed session cannot be used to guess the password either.
	key := users.Key(user.Username)
	if allowed, retryAfter := s.logins.Allowed(key); !allowed {
		return &TooManyAttemptsError{RetryAfter: retryAfter}
	}

	account, ok, err := s.checkPassword(ctx, user.Username, current)
	if err != nil {
		return err
	}
	if !ok || account.ID != user.ID {
		s.logins.Record(key)
		return ErrWrongPassword
	}
	s.logins.Reset(key)

	hash, err := s.hasher.Hash(ctx, replacement)
	if err != nil {
		return err
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := users.SetPasswordHashIn(tx, user.ID, hash); err != nil {
			return err
		}

		err := tx.Where("user_id = ? AND id <> ?", user.ID, sessionID(keepToken)).Delete(&sessionRecord{}).Error
		if err != nil {
			return fmt.Errorf("ending the account's other sessions: %w", err)
		}

		return nil
	})
}

// HashNewPassword checks a password an admin gives a player against the
// rules and hashes it, to store with users.SetPasswordHashIn. It returns a
// *validation.Error for a password breaking the rules.
func (s *Service) HashNewPassword(ctx context.Context, password string) (string, error) {
	if err := ValidatePassword(password); err != nil {
		return "", err
	}
	return s.hasher.Hash(ctx, password)
}

// ForgetWrongPasswords clears the wrong passwords counted against an
// account, so a player an admin has given a new password can log in at once.
func (s *Service) ForgetWrongPasswords(username string) {
	s.logins.Reset(users.Key(username))
}

// EndSessionsIn ends every session of an account inside the caller's
// transaction, as a ban or a new password does.
func EndSessionsIn(tx *gorm.DB, userID string) error {
	if err := tx.Where("user_id = ?", userID).Delete(&sessionRecord{}).Error; err != nil {
		return fmt.Errorf("ending an account's sessions: %w", err)
	}
	return nil
}

// DeleteExpiredSessions removes the sessions that have expired and returns
// how many there were.
func (s *Service) DeleteExpiredSessions(ctx context.Context) (int64, error) {
	result := s.db.WithContext(ctx).Where("expires_at <= ?", s.timestamp()).Delete(&sessionRecord{})
	if result.Error != nil {
		return 0, fmt.Errorf("deleting expired sessions: %w", result.Error)
	}

	return result.RowsAffected, nil
}

func (s *Service) startSession(ctx context.Context, userID string) (Session, error) {
	token, err := newSessionToken()
	if err != nil {
		return Session{}, err
	}

	now := s.timestamp()
	record := sessionRecord{ID: sessionID(token), UserID: userID, CreatedAt: now, ExpiresAt: now.Add(SessionDuration)}
	if err := s.db.WithContext(ctx).Create(&record).Error; err != nil {
		return Session{}, fmt.Errorf("starting a session: %w", err)
	}

	return Session{Token: token, ExpiresAt: record.ExpiresAt}, nil
}

// timestamp is the current time as the database stores it: UTC, to the
// microsecond.
func (s *Service) timestamp() time.Time {
	return s.now().UTC().Truncate(time.Microsecond)
}
