// Package users keeps the player accounts: their names, roles and password
// hashes. It knows nothing about passwords themselves or logins; package auth
// hashes passwords and runs sessions on top of it.
package users

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/validation"
)

// Role is what an account may do.
type Role string

const (
	RolePlayer Role = "player"
	RoleAdmin  Role = "admin"
)

// Username rules, the same as Triple Triad's.
const (
	UsernameMinLength = 3
	UsernameMaxLength = 20
)

// usernamePattern allows letters, digits, underscores and hyphens.
var usernamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

var (
	// ErrNotFound is returned when no account matches.
	ErrNotFound = errors.New("account not found")
	// ErrUsernameTaken is returned when another account already has the name,
	// in any combination of upper and lower case.
	ErrUsernameTaken = errors.New("username taken")
)

// User is a player account.
type User struct {
	ID string `gorm:"primaryKey"`
	// Username is the name as the player typed it.
	Username string
	// UsernameKey is the lowercase form that keeps names unique.
	UsernameKey  string
	PasswordHash string
	Role         Role
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (User) TableName() string { return "users" }

// IsAdmin reports whether the account may use the admin panel.
func (u User) IsAdmin() bool { return u.Role == RoleAdmin }

// NormalizeUsername trims a name as typed and checks it against the rules. It
// returns the name to show and the key that keeps it unique.
func NormalizeUsername(raw string) (name, key string, err error) {
	name = strings.TrimSpace(raw)

	switch {
	case len(name) < UsernameMinLength:
		return "", "", validation.New("username", fmt.Sprintf("must be at least %d characters", UsernameMinLength))
	case len(name) > UsernameMaxLength:
		return "", "", validation.New("username", fmt.Sprintf("must be at most %d characters", UsernameMaxLength))
	case !usernamePattern.MatchString(name):
		return "", "", validation.New("username", "may only contain letters, numbers, underscores and hyphens")
	}

	return name, Key(name), nil
}

// Key returns the case-insensitive form of a name, used to find accounts and
// to keep names unique.
func Key(username string) string {
	return strings.ToLower(strings.TrimSpace(username))
}

// Service reads and changes accounts. It is safe for concurrent use.
type Service struct {
	db     *gorm.DB
	logger *slog.Logger

	// createMu makes deciding whether an account is the first one and
	// creating it a single step, so two accounts created at the same moment on
	// an empty server cannot both become admin. A lock in memory is enough,
	// since one server runs per database.
	createMu sync.Mutex
}

// NewService returns a service using db.
func NewService(db *gorm.DB, logger *slog.Logger) *Service {
	return &Service{db: db, logger: logger}
}

// Create adds an account with an already hashed password. The first account
// ever created becomes an admin; every later one is a player. It returns a
// *validation.Error for a name breaking the rules and ErrUsernameTaken for a
// name already in use.
func (s *Service) Create(ctx context.Context, username, passwordHash string) (User, error) {
	name, key, err := NormalizeUsername(username)
	if err != nil {
		return User{}, err
	}

	id, err := uuid.NewV7()
	if err != nil {
		return User{}, fmt.Errorf("making an account id: %w", err)
	}

	s.createMu.Lock()
	defer s.createMu.Unlock()

	user := User{ID: id.String(), Username: name, UsernameKey: key, PasswordHash: passwordHash, Role: RolePlayer}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing int64
		if err := tx.Model(&User{}).Limit(1).Count(&existing).Error; err != nil {
			return err
		}
		if existing == 0 {
			user.Role = RoleAdmin
		}

		return tx.Create(&user).Error
	})
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return User{}, ErrUsernameTaken
	}
	if err != nil {
		return User{}, fmt.Errorf("creating account %q: %w", name, err)
	}

	if user.IsAdmin() {
		s.logger.InfoContext(ctx, "created the first account, which is the admin", "username", user.Username)
	} else {
		s.logger.InfoContext(ctx, "created an account", "username", user.Username)
	}

	return user, nil
}

// ByUsername finds an account by name, in any case. It returns ErrNotFound if
// there is none.
func (s *Service) ByUsername(ctx context.Context, username string) (User, error) {
	return s.find(ctx, "username_key = ?", Key(username))
}

// ByID finds an account by id. It returns ErrNotFound if there is none.
func (s *Service) ByID(ctx context.Context, id string) (User, error) {
	return s.find(ctx, "id = ?", id)
}

func (s *Service) find(ctx context.Context, condition string, value string) (User, error) {
	var user User
	err := s.db.WithContext(ctx).Where(condition, value).Take(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("finding an account: %w", err)
	}

	return user, nil
}

// SetPasswordHash replaces an account's password hash. It returns ErrNotFound
// if the account does not exist.
func (s *Service) SetPasswordHash(ctx context.Context, id, passwordHash string) error {
	return setPasswordHash(s.db.WithContext(ctx), id, passwordHash)
}

// SetPasswordHashIn is SetPasswordHash inside the caller's transaction.
func (s *Service) SetPasswordHashIn(tx *gorm.DB, id, passwordHash string) error {
	return setPasswordHash(tx, id, passwordHash)
}

func setPasswordHash(db *gorm.DB, id, passwordHash string) error {
	result := db.Model(&User{}).Where("id = ?", id).Update("password_hash", passwordHash)
	if result.Error != nil {
		return fmt.Errorf("changing a password: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// AnyExist reports whether at least one account exists.
func (s *Service) AnyExist(ctx context.Context) (bool, error) {
	var count int64
	if err := s.db.WithContext(ctx).Model(&User{}).Limit(1).Count(&count).Error; err != nil {
		return false, fmt.Errorf("counting accounts: %w", err)
	}

	return count > 0, nil
}
