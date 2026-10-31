// Package mail sends the site's email through the SMTP server an admin sets up.
// The server's details, password included, are kept encrypted in the database.
package mail

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/secrets"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/validation"
)

// Security is how the connection to the mail server is protected.
type Security string

const (
	// StartTLS connects in the clear and insists on upgrading, usually on port
	// 587.
	StartTLS Security = "starttls"
	// ImplicitTLS is encrypted from the start, usually on port 465.
	ImplicitTLS Security = "tls"
	// NoSecurity is for a mail server on the same machine or a local test
	// server. A password is refused over it unless the server is local.
	NoSecurity Security = "none"
)

// Limits of the settings.
const (
	maxHostLength     = 253
	maxUsernameLength = 254
	maxPasswordLength = 1024
	maxFromNameLength = 100
	maxAddressLength  = 254
	minPort           = 1
	maxPort           = 65535
)

// sealPurpose is what the settings are sealed for; see secrets.Box.Seal.
const sealPurpose = "mail.settings"

// settingsRowID is the only row of the mail_settings table.
const settingsRowID = 1

var (
	// ErrNotConfigured is returned for sending before an admin has set up a
	// mail server.
	ErrNotConfigured = errors.New("no mail server is set up")
	// ErrNoSecretKey is returned for saving settings while the server has no
	// key to encrypt them with.
	ErrNoSecretKey = errors.New("APP_SECRET_KEY is not set, so mail settings cannot be stored")
	// ErrUnreadable is returned while the stored settings cannot be decrypted,
	// because APP_SECRET_KEY has changed.
	ErrUnreadable = errors.New("the stored mail settings cannot be read with this APP_SECRET_KEY")
)

// Settings say how to reach the mail server and who the mail comes from.
type Settings struct {
	Host        string   `json:"host"`
	Port        int      `json:"port"`
	Security    Security `json:"security"`
	Username    string   `json:"username"`
	Password    string   `json:"password"`
	FromAddress string   `json:"fromAddress"`
	FromName    string   `json:"fromName"`
}

// Input is the settings an admin gives. A nil password keeps the stored one, so
// the form never has to show it.
type Input struct {
	Host        string
	Port        int
	Security    Security
	Username    string
	Password    *string
	FromAddress string
	FromName    string
}

// Status is what the admin panel shows. It never includes the password.
type Status struct {
	Settings
	// Configured is whether a mail server is set up and can be used.
	Configured bool
	// HasPassword is whether a password is stored.
	HasPassword bool
	// KeyAvailable is whether the server has a key to store settings with.
	KeyAvailable bool
	// Unreadable is whether settings are stored that this key cannot read.
	Unreadable bool
	// UpdatedAt is when the settings were last changed; nil when never.
	UpdatedAt *time.Time
}

// settingsRow is the row of the mail_settings table: every setting sealed
// together.
type settingsRow struct {
	ID        int `gorm:"primaryKey;autoIncrement:false"`
	Sealed    string
	UpdatedAt time.Time
}

func (settingsRow) TableName() string { return "mail_settings" }

// Service reads the settings and sends mail. It is safe for concurrent use.
type Service struct {
	db     *gorm.DB
	box    *secrets.Box
	logger *slog.Logger
	dial   dialer
}

// NewService returns the service. box is nil when the server has no key, which
// leaves mail unavailable.
func NewService(db *gorm.DB, box *secrets.Box, logger *slog.Logger) *Service {
	return &Service{db: db, box: box, logger: logger, dial: dialServer}
}

// read returns the stored settings. found is false when none are stored.
func (s *Service) read(ctx context.Context, db *gorm.DB) (settings Settings, updatedAt time.Time, found bool, err error) {
	var row settingsRow
	err = db.WithContext(ctx).Where("id = ?", settingsRowID).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Settings{}, time.Time{}, false, nil
	}
	if err != nil {
		return Settings{}, time.Time{}, false, fmt.Errorf("reading the mail settings: %w", err)
	}
	if s.box == nil {
		return Settings{}, row.UpdatedAt, true, ErrUnreadable
	}

	opened, err := s.box.Open(row.Sealed, sealPurpose)
	if err != nil {
		return Settings{}, row.UpdatedAt, true, ErrUnreadable
	}
	if err := json.Unmarshal(opened, &settings); err != nil {
		return Settings{}, row.UpdatedAt, true, fmt.Errorf("reading the mail settings: %w", err)
	}

	return settings, row.UpdatedAt, true, nil
}

// Status returns what the admin panel shows.
func (s *Service) Status(ctx context.Context) (Status, error) {
	status := Status{KeyAvailable: s.box != nil}

	settings, updatedAt, found, err := s.read(ctx, s.db)
	if errors.Is(err, ErrUnreadable) {
		status.Unreadable = true
		status.UpdatedAt = &updatedAt
		return status, nil
	}
	if err != nil {
		return Status{}, err
	}
	if !found {
		return status, nil
	}

	status.HasPassword = settings.Password != ""
	settings.Password = ""
	status.Settings = settings
	status.Configured = true
	status.UpdatedAt = &updatedAt
	return status, nil
}

// Configured reports whether a mail server is set up and can be used. Settings
// this key cannot read count as not set up, and are logged.
func (s *Service) Configured(ctx context.Context) (bool, error) {
	_, _, found, err := s.read(ctx, s.db)
	if errors.Is(err, ErrUnreadable) {
		s.logger.ErrorContext(ctx, "the stored mail settings cannot be read, so no mail can be sent; set them again in the admin panel", "error", err)
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return found, nil
}

// SaveIn checks and stores the settings inside the caller's transaction. It
// returns ErrNoSecretKey without a key to encrypt them with, and a
// *validation.Error for settings breaking the rules.
func (s *Service) SaveIn(ctx context.Context, tx *gorm.DB, input Input, now time.Time) error {
	if s.box == nil {
		return ErrNoSecretKey
	}

	settings, err := normalize(input)
	if err != nil {
		return err
	}

	if input.Password == nil {
		stored, _, found, err := s.read(ctx, tx)
		if err != nil && !errors.Is(err, ErrUnreadable) {
			return err
		}
		if found && err == nil {
			settings.Password = stored.Password
		}
	}

	plain, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("writing the mail settings: %w", err)
	}
	sealed, err := s.box.Seal(plain, sealPurpose)
	if err != nil {
		return fmt.Errorf("encrypting the mail settings: %w", err)
	}

	row := settingsRow{ID: settingsRowID, Sealed: sealed, UpdatedAt: now}
	if err := tx.WithContext(ctx).Save(&row).Error; err != nil {
		return fmt.Errorf("storing the mail settings: %w", err)
	}

	return nil
}

// ClearIn forgets the settings inside the caller's transaction, which leaves
// mail unavailable. It reports whether there were any.
func (s *Service) ClearIn(ctx context.Context, tx *gorm.DB) (bool, error) {
	result := tx.WithContext(ctx).Where("id = ?", settingsRowID).Delete(&settingsRow{})
	if result.Error != nil {
		return false, fmt.Errorf("forgetting the mail settings: %w", result.Error)
	}

	return result.RowsAffected > 0, nil
}

// normalize checks an admin's settings and returns them tidied.
func normalize(input Input) (Settings, error) {
	settings := Settings{
		Host:        strings.TrimSpace(input.Host),
		Port:        input.Port,
		Security:    input.Security,
		Username:    strings.TrimSpace(input.Username),
		FromAddress: strings.TrimSpace(input.FromAddress),
		FromName:    strings.TrimSpace(input.FromName),
	}
	if input.Password != nil {
		settings.Password = *input.Password
	}

	switch {
	case settings.Host == "":
		return Settings{}, validation.New("host", "is required")
	case utf8.RuneCountInString(settings.Host) > maxHostLength, strings.ContainsAny(settings.Host, " \t\r\n/:"):
		return Settings{}, validation.New("host", "must be a server name such as smtp.example.com")
	case settings.Port < minPort || settings.Port > maxPort:
		return Settings{}, validation.New("port", fmt.Sprintf("must be between %d and %d", minPort, maxPort))
	case utf8.RuneCountInString(settings.Username) > maxUsernameLength:
		return Settings{}, validation.New("username", fmt.Sprintf("must be at most %d characters", maxUsernameLength))
	case utf8.RuneCountInString(settings.Password) > maxPasswordLength:
		return Settings{}, validation.New("password", fmt.Sprintf("must be at most %d characters", maxPasswordLength))
	case utf8.RuneCountInString(settings.FromName) > maxFromNameLength:
		return Settings{}, validation.New("fromName", fmt.Sprintf("must be at most %d characters", maxFromNameLength))
	case strings.ContainsAny(settings.FromName, "\r\n"):
		return Settings{}, validation.New("fromName", "must be one line")
	}

	switch settings.Security {
	case StartTLS, ImplicitTLS, NoSecurity:
	default:
		return Settings{}, validation.New("security", `must be "starttls", "tls" or "none"`)
	}

	if err := CheckAddress(settings.FromAddress); err != nil {
		return Settings{}, validation.New("fromAddress", "must be an email address")
	}

	return settings, nil
}

// CheckAddress checks a bare email address, such as name@example.com, without a
// display name.
func CheckAddress(address string) error {
	if address == "" || utf8.RuneCountInString(address) > maxAddressLength {
		return errors.New("not an email address")
	}
	parsed, err := mail.ParseAddress(address)
	if err != nil || parsed.Address != address {
		return errors.New("not an email address")
	}

	return nil
}
