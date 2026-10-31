package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/settings"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/users"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/validation"
)

// Code rules.
const (
	// codeDigits is the length of a code.
	codeDigits = 6
	// codeAttempts is how many wrong guesses a code survives. With a million
	// possible codes and a new one needed after these, guessing does not pay.
	codeAttempts = 5
)

// The admin settings for confirming an email address.
var (
	// CodeMinutes is how long a code works.
	CodeMinutes = settings.Number{
		Key:         "accounts.verification_code_minutes",
		Group:       "Accounts",
		Description: "How many minutes the code sent to a new player's email address works.",
		Default:     15,
		Min:         5,
		Max:         120,
	}
	// ResendSeconds is how long a player waits between codes.
	ResendSeconds = settings.Number{
		Key:         "accounts.verification_resend_seconds",
		Group:       "Accounts",
		Description: "How many seconds a new player waits before asking for another code.",
		Default:     60,
		Min:         15,
		Max:         900,
	}
	// UnconfirmedHours is how long an account may go unconfirmed once it has
	// been sent a code.
	UnconfirmedHours = settings.Number{
		Key:         "accounts.unverified_hours",
		Group:       "Accounts",
		Description: "How many hours after their first code a new player has to confirm their email address, before the account is deleted and its name and address are free again.",
		Default:     24,
		Min:         1,
		Max:         720,
	}
)

// Mailer sends email through the site's mail server.
type Mailer interface {
	// Configured reports whether a mail server is set up.
	Configured(ctx context.Context) (bool, error)
	Send(ctx context.Context, to, subject, body string) error
}

var (
	// ErrMailUnavailable is returned for asking for a code while no mail server
	// is set up.
	ErrMailUnavailable = errors.New("the site cannot send email yet")
	// ErrWrongCode is returned for a code that is not the one sent.
	ErrWrongCode = errors.New("that code is wrong")
	// ErrCodeExpired is returned for a code too old, or tried too many times,
	// and when no code has been sent.
	ErrCodeExpired = errors.New("that code has expired")
)

// MailFailedError is returned when the mail server would not take a code.
type MailFailedError struct {
	Err error
}

func (e *MailFailedError) Error() string { return "sending the code failed: " + e.Err.Error() }
func (e *MailFailedError) Unwrap() error { return e.Err }

// ResendTooSoonError is returned for asking for another code too soon.
type ResendTooSoonError struct {
	RetryAfter time.Duration
}

func (e *ResendTooSoonError) Error() string {
	return fmt.Sprintf("wait %s before asking for another code", e.RetryAfter.Round(time.Second))
}

// verificationRecord is a row of email_verifications: the code last sent to an
// account. Only its hash is kept, so a leaked database holds no usable codes.
type verificationRecord struct {
	UserID    string `gorm:"primaryKey"`
	CodeHash  string
	SentAt    time.Time
	ExpiresAt time.Time
	Attempts  int
}

func (verificationRecord) TableName() string { return "email_verifications" }

// Verification is how confirming an account's email address stands.
type Verification struct {
	Email string
	// MailWorks is whether the site can send a code right now.
	MailWorks bool
	// CodeSentAt and CodeExpiresAt are about the last code; nil before one is
	// sent.
	CodeSentAt    *time.Time
	CodeExpiresAt *time.Time
	// ResendIn is how long until another code may be asked for.
	ResendIn time.Duration
	// DeleteAt is when the account is deleted if still not confirmed; nil
	// before its first code.
	DeleteAt *time.Time
}

// VerificationOf returns how confirming an account's address stands.
func (s *Service) VerificationOf(ctx context.Context, user users.User) (Verification, error) {
	works, err := s.mailer.Configured(ctx)
	if err != nil {
		return Verification{}, err
	}

	account, err := s.users.ByID(ctx, user.ID)
	if err != nil {
		return Verification{}, err
	}

	status := Verification{Email: account.Email, MailWorks: works, DeleteAt: account.VerifyBy}

	record, found, err := s.lastCode(ctx, user.ID)
	if err != nil {
		return Verification{}, err
	}
	if found {
		status.CodeSentAt = &record.SentAt
		status.CodeExpiresAt = &record.ExpiresAt
		status.ResendIn = max(0, s.resendAt(record).Sub(s.timestamp()))
	}

	return status, nil
}

func (s *Service) lastCode(ctx context.Context, userID string) (verificationRecord, bool, error) {
	var record verificationRecord
	err := s.db.WithContext(ctx).Where("user_id = ?", userID).Take(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return verificationRecord{}, false, nil
	}
	if err != nil {
		return verificationRecord{}, false, fmt.Errorf("reading a code: %w", err)
	}

	return record, true, nil
}

func (s *Service) resendAt(record verificationRecord) time.Time {
	return record.SentAt.Add(time.Duration(s.settings.Number(ResendSeconds)) * time.Second)
}

// sendFirstCode sends the code a new account starts with. It waits, without
// complaint, while there is no mail server, and a mail server refusing is logged
// where it happens: either way the player can ask again from the page they land
// on.
func (s *Service) sendFirstCode(ctx context.Context, user users.User) {
	err := s.SendCode(ctx, user)
	var failed *MailFailedError
	if err == nil || errors.Is(err, ErrMailUnavailable) || errors.As(err, &failed) {
		return
	}
	s.logger.WarnContext(ctx, "could not send a new player their code", "user_id", user.ID, "error", err)
}

// SendCode sends a new code to an account's address, replacing any before it. It
// returns users.ErrEmailConfirmed for an account already confirmed,
// ErrMailUnavailable while no mail server is set up, a *ResendTooSoonError for
// asking again too soon and a *MailFailedError when the mail server will not
// take it. Only a code that was sent counts: a failure can be retried at once.
func (s *Service) SendCode(ctx context.Context, user users.User) error {
	if user.IsVerified() {
		return users.ErrEmailConfirmed
	}

	if err := s.checkResend(ctx, user.ID); err != nil {
		return err
	}

	works, err := s.mailer.Configured(ctx)
	if err != nil {
		return err
	}
	if !works {
		return ErrMailUnavailable
	}

	code, err := newCode()
	if err != nil {
		return err
	}

	minutes := s.settings.Number(CodeMinutes)
	body := fmt.Sprintf("Your Duel Masters code is %s.\n\nEnter it on the site to finish creating your account. It works for %d minutes.\n\nIf you did not sign up, you can ignore this email.", code, minutes)
	if err := s.mailer.Send(ctx, user.Email, "Your Duel Masters code: "+code, body); err != nil {
		s.logger.WarnContext(ctx, "could not send a code", "user_id", user.ID, "error", err)
		return &MailFailedError{Err: err}
	}

	now := s.timestamp()
	record := verificationRecord{
		UserID:    user.ID,
		CodeHash:  codeHash(user.ID, code),
		SentAt:    now,
		ExpiresAt: now.Add(time.Duration(minutes) * time.Minute),
	}
	if err := s.db.WithContext(ctx).Save(&record).Error; err != nil {
		return fmt.Errorf("storing a code: %w", err)
	}

	hours := s.settings.Number(UnconfirmedHours)
	return s.users.SetVerifyBy(ctx, user.ID, now.Add(time.Duration(hours)*time.Hour))
}

// checkResend returns a *ResendTooSoonError while the last code is too recent.
func (s *Service) checkResend(ctx context.Context, userID string) error {
	record, found, err := s.lastCode(ctx, userID)
	if err != nil {
		return err
	}
	if !found {
		return nil
	}

	if wait := s.resendAt(record).Sub(s.timestamp()); wait > 0 {
		return &ResendTooSoonError{RetryAfter: wait}
	}

	return nil
}

// ChangeEmail replaces the address of an account not yet confirmed, for a player
// who mistyped it, and sends a code to the new one. It is held to the same wait
// as asking for a code, so it cannot be used to send mail to address after
// address. The new address is kept even if sending fails, as sent reports. It
// returns a *validation.Error for an address breaking the rules,
// users.ErrEmailTaken for one in use, users.ErrEmailConfirmed for an account
// already confirmed and a *ResendTooSoonError for changing too soon after a code.
func (s *Service) ChangeEmail(ctx context.Context, user users.User, email string) (changed users.User, sent bool, err error) {
	if user.IsVerified() {
		return users.User{}, false, users.ErrEmailConfirmed
	}

	if err := s.checkResend(ctx, user.ID); err != nil {
		return users.User{}, false, err
	}

	changed, err = s.users.ChangeEmail(ctx, user.ID, email)
	if err != nil {
		return users.User{}, false, err
	}

	// A code for the old address must not confirm the new one.
	if err := s.db.WithContext(ctx).Where("user_id = ?", user.ID).Delete(&verificationRecord{}).Error; err != nil {
		return users.User{}, false, fmt.Errorf("forgetting a code: %w", err)
	}

	err = s.SendCode(ctx, changed)
	var failed *MailFailedError
	switch {
	case err == nil:
		return changed, true, nil
	case errors.Is(err, ErrMailUnavailable), errors.As(err, &failed):
		return changed, false, nil
	default:
		return users.User{}, false, err
	}
}

// ConfirmCode checks a code and, when it is right, confirms the account's
// address, which lets the player in. It returns a *validation.Error for
// something that is not a code, users.ErrEmailConfirmed for an account already
// confirmed, ErrWrongCode for a wrong code and ErrCodeExpired once a new one is
// needed.
func (s *Service) ConfirmCode(ctx context.Context, user users.User, code string) (users.User, error) {
	if user.IsVerified() {
		return users.User{}, users.ErrEmailConfirmed
	}

	code = strings.ReplaceAll(strings.TrimSpace(code), " ", "")
	if !isCode(code) {
		return users.User{}, validation.New("code", fmt.Sprintf("must be the %d digits from the email", codeDigits))
	}

	now := s.timestamp()
	record, found, err := s.lastCode(ctx, user.ID)
	if err != nil {
		return users.User{}, err
	}
	if !found || !record.ExpiresAt.After(now) {
		return users.User{}, ErrCodeExpired
	}

	// Counted before comparing, in one step, so that guesses made at the same
	// moment cannot get more tries than a code has.
	counted := s.db.WithContext(ctx).Model(&verificationRecord{}).
		Where("user_id = ? AND attempts < ?", user.ID, codeAttempts).
		Update("attempts", gorm.Expr("attempts + 1"))
	if counted.Error != nil {
		return users.User{}, fmt.Errorf("counting a guess at a code: %w", counted.Error)
	}
	if counted.RowsAffected == 0 {
		return users.User{}, ErrCodeExpired
	}

	if subtle.ConstantTimeCompare([]byte(record.CodeHash), []byte(codeHash(user.ID, code))) != 1 {
		return users.User{}, ErrWrongCode
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := ForgetCodeIn(tx, user.ID); err != nil {
			return err
		}
		return users.ConfirmEmailIn(tx, user.ID, now)
	})
	if err != nil {
		return users.User{}, err
	}

	s.logger.InfoContext(ctx, "a player confirmed their email address", "user_id", user.ID)
	return s.users.ByID(ctx, user.ID)
}

// ForgetCodeIn removes an account's code inside the caller's transaction.
func ForgetCodeIn(tx *gorm.DB, userID string) error {
	if err := tx.Where("user_id = ?", userID).Delete(&verificationRecord{}).Error; err != nil {
		return fmt.Errorf("forgetting a code: %w", err)
	}
	return nil
}

// DeleteUnconfirmedAccounts deletes the accounts whose time to confirm their
// address has run out, and returns how many there were.
func (s *Service) DeleteUnconfirmedAccounts(ctx context.Context) (int64, error) {
	return s.users.DeleteUnconfirmed(ctx, s.timestamp())
}

// newCode returns a random code, leading zeros and all.
func newCode() (string, error) {
	limit := new(big.Int).Exp(big.NewInt(10), big.NewInt(codeDigits), nil)
	number, err := rand.Int(rand.Reader, limit)
	if err != nil {
		return "", fmt.Errorf("making a code: %w", err)
	}

	return fmt.Sprintf("%0*d", codeDigits, number), nil
}

func isCode(text string) bool {
	if len(text) != codeDigits {
		return false
	}
	for _, r := range text {
		if r < '0' || r > '9' {
			return false
		}
	}

	return true
}

// codeHash is what the database stores for a code. The account's id keeps the
// same code from having the same hash on two accounts.
func codeHash(userID, code string) string {
	sum := sha256.Sum256([]byte(userID + ":" + code))
	return hex.EncodeToString(sum[:])
}
