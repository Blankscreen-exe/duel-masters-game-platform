// Package users keeps the player accounts: their names, roles, password
// hashes, bans and mutes. It knows nothing about passwords themselves or
// logins; package auth hashes passwords and runs sessions on top of it, and
// package admin decides who may change what.
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
	"unicode/utf8"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/avatars"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/mail"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/paging"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/store"
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
	// ErrEmailTaken is returned when another account already has the email
	// address, in any combination of upper and lower case.
	ErrEmailTaken = errors.New("email taken")
	// ErrEmailConfirmed is returned for changing an address already confirmed.
	ErrEmailConfirmed = errors.New("the email address is already confirmed")
	// ErrNotAdmin is returned when the account acting is not, or no longer,
	// an admin.
	ErrNotAdmin = errors.New("only admins can do this")
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
	// Computer marks the account of a computer opponent: it has no
	// password, so nobody logs in to it, and admins cannot change it.
	Computer bool
	// AvatarID is the picture the player chose from package avatars, empty until
	// they choose one.
	AvatarID string
	// Email is the address as the player typed it; empty for computer
	// opponents.
	Email string
	// EmailKey is the lowercase form that keeps addresses unique.
	EmailKey string
	// EmailVerifiedAt is when the address was confirmed, or an admin let the
	// player in; nil keeps the player out until then.
	EmailVerifiedAt *time.Time
	// VerifyBy is when an account still not confirmed is deleted; nil for one
	// that has never been sent a code, waiting for the site's mail to work.
	VerifyBy *time.Time
	// AnnouncementsSeenAt is when the player last looked at the site's
	// announcements; nil before they first do.
	AnnouncementsSeenAt *time.Time
	Moderation
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (User) TableName() string { return "users" }

// IsAdmin reports whether the account may use the admin panel.
func (u User) IsAdmin() bool { return u.Role == RoleAdmin }

// IsVerified reports whether the player may use the site: their email address
// is confirmed, or needs no confirming.
func (u User) IsVerified() bool { return u.EmailVerifiedAt != nil }

// Moderation is an account's latest ban and mute, as the users table keeps
// them. Other features reading the table embed it to know who is banned.
type Moderation struct {
	// BannedAt, BannedUntil and BanReason are the latest ban; see BanAt.
	// BannedAt is nil for an account never banned, or whose ban an admin
	// lifted.
	BannedAt    *time.Time
	BannedUntil *time.Time
	BanReason   string
	// MutedAt and MutedUntil are the latest mute; see MuteAt.
	MutedAt    *time.Time
	MutedUntil *time.Time
}

// Penalty is a ban or a mute: since when, and until when, or for good.
type Penalty struct {
	Since time.Time
	// Until is when the penalty lifts itself; nil for one for good.
	Until *time.Time
}

// Ban is a ban on an account, with the reason its player is shown.
type Ban struct {
	Penalty
	Reason string
}

// penaltyAt returns the penalty that since and until describe, if it is in
// force at now.
func penaltyAt(since, until *time.Time, now time.Time) (Penalty, bool) {
	if since == nil || (until != nil && !until.After(now)) {
		return Penalty{}, false
	}
	return Penalty{Since: *since, Until: until}, true
}

// BanAt returns the account's ban, if it is banned at now.
func (m Moderation) BanAt(now time.Time) (Ban, bool) {
	penalty, banned := penaltyAt(m.BannedAt, m.BannedUntil, now)
	if !banned {
		return Ban{}, false
	}
	return Ban{Penalty: penalty, Reason: m.BanReason}, true
}

// MuteAt returns the account's mute, if it is muted at now.
func (m Moderation) MuteAt(now time.Time) (Penalty, bool) {
	return penaltyAt(m.MutedAt, m.MutedUntil, now)
}

// The columns of a ban and of a mute.
var (
	banColumns  = penaltyColumns{since: "banned_at", until: "banned_until"}
	muteColumns = penaltyColumns{since: "muted_at", until: "muted_until"}
)

// penaltyColumns are the users table's columns of a ban or a mute, for
// queries, which find penalties in force the same way penaltyAt does.
type penaltyColumns struct{ since, until string }

func (c penaltyColumns) notInForceAt(now time.Time) clause.Expression {
	return clause.Expr{
		SQL:  "(? IS NULL OR ? <= ?)",
		Vars: []any{clause.Column{Table: "users", Name: c.since}, clause.Column{Table: "users", Name: c.until}, now},
	}
}

func (c penaltyColumns) inForceAt(now time.Time) clause.Expression {
	return clause.Expr{
		SQL:  "(? IS NOT NULL AND (? IS NULL OR ? > ?))",
		Vars: []any{clause.Column{Table: "users", Name: c.since}, clause.Column{Table: "users", Name: c.until}, clause.Column{Table: "users", Name: c.until}, now},
	}
}

// NotBannedAt is the condition that an account of the users table is not
// banned at now, for queries that leave banned players out.
func NotBannedAt(now time.Time) clause.Expression {
	return banColumns.notInForceAt(now)
}

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

// NormalizeEmail checks an email address and returns it tidied, with the
// lowercase key that keeps addresses unique. It returns a *validation.Error for
// one that is not a bare address.
func NormalizeEmail(raw string) (email, key string, err error) {
	email = strings.TrimSpace(raw)
	if mail.CheckAddress(email) != nil {
		return "", "", validation.New("email", "must be an email address such as name@example.com")
	}

	return email, strings.ToLower(email), nil
}

// Register adds a player's account with an email address and an already hashed
// password. The first person's account ever created becomes an admin, computer
// opponents' not counting, and has no code to enter: it sets up the mail server
// that sends them. Every later one is a player, kept out until the address is
// confirmed. It returns a *validation.Error for a name or address breaking the
// rules, ErrUsernameTaken for a name in use and ErrEmailTaken for an address in
// use.
func (s *Service) Register(ctx context.Context, username, email, passwordHash string) (User, error) {
	address, addressKey, err := NormalizeEmail(email)
	if err != nil {
		return User{}, err
	}

	return s.create(ctx, username, passwordHash, address, addressKey)
}

// Create adds an account with no email address to confirm, and an already
// hashed password, for setting up players without going through sign-up. See
// Register.
func (s *Service) Create(ctx context.Context, username, passwordHash string) (User, error) {
	return s.create(ctx, username, passwordHash, "", "")
}

// create adds an account. One with no address, or the first, is confirmed from
// the start.
func (s *Service) create(ctx context.Context, username, passwordHash, email, emailKey string) (User, error) {
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

	user := User{ID: id.String(), Username: name, UsernameKey: key, PasswordHash: passwordHash, Role: RolePlayer, Email: email, EmailKey: emailKey}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing int64
		if err := people(tx.Model(&User{})).Limit(1).Count(&existing).Error; err != nil {
			return err
		}
		if existing == 0 {
			user.Role = RoleAdmin
		}
		if existing == 0 || email == "" {
			now := tx.NowFunc()
			user.EmailVerifiedAt = &now
		}

		if emailKey != "" {
			taken, err := emailTakenIn(tx, emailKey, user.ID)
			if err != nil {
				return err
			}
			if taken {
				return ErrEmailTaken
			}
		}

		return tx.Create(&user).Error
	})
	if errors.Is(err, ErrEmailTaken) {
		return User{}, ErrEmailTaken
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		// Another server could have taken the address since it was checked.
		if emailKey != "" {
			if taken, checkErr := emailTakenIn(s.db.WithContext(ctx), emailKey, user.ID); checkErr == nil && taken {
				return User{}, ErrEmailTaken
			}
		}
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

// emailTakenIn reports whether an account other than id has the address.
func emailTakenIn(tx *gorm.DB, emailKey, id string) (bool, error) {
	var count int64
	err := tx.Model(&User{}).Where("email_key = ? AND id <> ?", emailKey, id).Limit(1).Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("checking an email address: %w", err)
	}

	return count > 0, nil
}

// ChangeEmail replaces the address of an account not yet confirmed, for a player
// who mistyped it. It returns a *validation.Error for an address breaking the
// rules, ErrEmailTaken for one in use, and ErrEmailConfirmed for an account
// already confirmed.
func (s *Service) ChangeEmail(ctx context.Context, id, email string) (User, error) {
	address, key, err := NormalizeEmail(email)
	if err != nil {
		return User{}, err
	}

	s.createMu.Lock()
	defer s.createMu.Unlock()

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		taken, err := emailTakenIn(tx, key, id)
		if err != nil {
			return err
		}
		if taken {
			return ErrEmailTaken
		}

		result := tx.Model(&User{}).Where("id = ? AND email_verified_at IS NULL", id).
			Updates(map[string]any{"email": address, "email_key": key})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrEmailConfirmed
		}
		return nil
	})
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return User{}, ErrEmailTaken
	}
	if err != nil {
		return User{}, err
	}

	return s.ByID(ctx, id)
}

// ConfirmEmailIn marks an account's address confirmed inside the caller's
// transaction, which lets the player in and saves the account from deletion.
func ConfirmEmailIn(tx *gorm.DB, id string, at time.Time) error {
	return update(tx, id, "confirming an email address", map[string]any{"email_verified_at": at, "verify_by": nil})
}

// SetVerifyBy sets when an account not yet confirmed is deleted, unless it
// already has a time: the first code starts the clock, and later ones do not
// put it back.
func (s *Service) SetVerifyBy(ctx context.Context, id string, at time.Time) error {
	err := s.db.WithContext(ctx).Model(&User{}).
		Where("id = ? AND email_verified_at IS NULL AND verify_by IS NULL", id).
		Update("verify_by", at).Error
	if err != nil {
		return fmt.Errorf("setting when an account is deleted: %w", err)
	}

	return nil
}

// DeleteUnconfirmed deletes the accounts whose time to confirm their address has
// run out, and returns how many there were. Their sessions and code go with
// them. An account never sent a code has no time set, and stays.
func (s *Service) DeleteUnconfirmed(ctx context.Context, now time.Time) (int64, error) {
	result := s.db.WithContext(ctx).
		Where("email_verified_at IS NULL AND verify_by IS NOT NULL AND verify_by <= ?", now).
		Delete(&User{})
	if result.Error != nil {
		return 0, fmt.Errorf("deleting unconfirmed accounts: %w", result.Error)
	}

	return result.RowsAffected, nil
}

// SetAnnouncementsSeen notes when the player last looked at the announcements.
func (s *Service) SetAnnouncementsSeen(ctx context.Context, id string, at time.Time) error {
	return update(s.db.WithContext(ctx), id, "noting announcements as seen", map[string]any{"announcements_seen_at": at})
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
	return SetPasswordHashIn(s.db.WithContext(ctx), id, passwordHash)
}

// SetPasswordHashIn is SetPasswordHash inside the caller's transaction.
func SetPasswordHashIn(tx *gorm.DB, id, passwordHash string) error {
	return update(tx, id, "changing a password", map[string]any{"password_hash": passwordHash})
}

// SetAvatar gives an account one of the pictures of package avatars. It returns
// a *validation.Error for a picture there is not, and ErrNotFound if the account
// does not exist.
func (s *Service) SetAvatar(ctx context.Context, id, avatarID string) error {
	if _, found := avatars.ByID(avatarID); !found {
		return validation.New("avatarId", "must be one of the pictures")
	}
	return update(s.db.WithContext(ctx), id, "changing a picture", map[string]any{"avatar_id": avatarID})
}

// AnyExist reports whether at least one person's account exists; computer
// opponents' do not count.
func (s *Service) AnyExist(ctx context.Context) (bool, error) {
	var count int64
	if err := people(s.db.WithContext(ctx).Model(&User{})).Limit(1).Count(&count).Error; err != nil {
		return false, fmt.Errorf("counting accounts: %w", err)
	}

	return count > 0, nil
}

// people narrows a query to the accounts of people, leaving out computer
// opponents'.
func people(query *gorm.DB) *gorm.DB {
	return query.Where("computer = ?", false)
}

// CreateComputer adds the account of a computer opponent, with the name
// given: an account with no password, which nobody can log in to. It returns
// a *validation.Error for a name breaking the rules and ErrUsernameTaken for
// a name already in use.
func (s *Service) CreateComputer(ctx context.Context, username string) (User, error) {
	return CreateComputerIn(s.db.WithContext(ctx), username)
}

// TakenIn reports whether an account has the name, in any case, inside the
// transaction tx: checking first rather than meeting the duplicate keeps a
// PostgreSQL transaction usable.
func TakenIn(tx *gorm.DB, username string) (bool, error) {
	var count int64
	if err := tx.Model(&User{}).Where("username_key = ?", Key(username)).Count(&count).Error; err != nil {
		return false, fmt.Errorf("looking for account %q: %w", username, err)
	}
	return count > 0, nil
}

// RenameComputerIn changes the name of a computer opponent's account inside the
// transaction tx. It returns a *validation.Error for a name breaking the rules,
// ErrUsernameTaken for a name another account has, and ErrNotFound for no such
// computer account.
func RenameComputerIn(tx *gorm.DB, id, username string) error {
	name, key, err := NormalizeUsername(username)
	if err != nil {
		return err
	}

	var others int64
	if err := tx.Model(&User{}).Where("username_key = ? AND id <> ?", key, id).Count(&others).Error; err != nil {
		return fmt.Errorf("looking for account %q: %w", name, err)
	}
	if others > 0 {
		return ErrUsernameTaken
	}

	result := tx.Model(&User{}).Where("id = ? AND computer = ?", id, true).Updates(map[string]any{"username": name, "username_key": key})
	if result.Error != nil {
		return fmt.Errorf("renaming computer account %q: %w", name, result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// CreateComputerIn is CreateComputer inside the transaction tx, such as a
// seed adding computer opponents with their accounts.
func CreateComputerIn(tx *gorm.DB, username string) (User, error) {
	name, key, err := NormalizeUsername(username)
	if err != nil {
		return User{}, err
	}

	id, err := uuid.NewV7()
	if err != nil {
		return User{}, fmt.Errorf("making an account id: %w", err)
	}

	confirmed := tx.NowFunc()
	user := User{ID: id.String(), Username: name, UsernameKey: key, Role: RolePlayer, Computer: true, EmailVerifiedAt: &confirmed}
	err = tx.Create(&user).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return User{}, ErrUsernameTaken
	}
	if err != nil {
		return User{}, fmt.Errorf("creating computer account %q: %w", name, err)
	}
	return user, nil
}

// Account search limits.
const (
	DefaultPageSize = 25
	MaxPageSize     = 100
)

// Status narrows an account search.
type Status string

const (
	// StatusAny is every account.
	StatusAny Status = ""
	// StatusBanned is the accounts banned now.
	StatusBanned Status = "banned"
	// StatusMuted is the accounts muted now.
	StatusMuted Status = "muted"
	// StatusAdmin is the admins.
	StatusAdmin Status = "admin"
	// StatusUnverified is the accounts whose email address is not yet
	// confirmed.
	StatusUnverified Status = "unverified"
)

// Query asks for one page of the accounts.
type Query struct {
	// Search keeps the accounts whose name contains it, ignoring case.
	Search string
	Status Status
	paging.Request
}

// Page is one page of the accounts, by name.
type Page struct {
	Users    []User
	Total    int
	Page     int
	PageSize int
}

// Search returns one page of the accounts, by name, banned or muted as they
// are at now. It returns a *validation.Error for a query breaking the limits.
func (s *Service) Search(ctx context.Context, query Query, now time.Time) (Page, error) {
	query.Search = strings.TrimSpace(query.Search)
	if utf8.RuneCountInString(query.Search) > UsernameMaxLength {
		return Page{}, validation.New("search", fmt.Sprintf("must be at most %d characters", UsernameMaxLength))
	}
	if err := query.Normalize(DefaultPageSize, MaxPageSize); err != nil {
		return Page{}, err
	}

	matching := s.db.WithContext(ctx).Model(&User{})
	if query.Search != "" {
		matching = matching.Where(store.Contains("username_key", Key(query.Search)))
	}
	switch query.Status {
	case StatusAny:
	case StatusBanned:
		matching = matching.Where(banColumns.inForceAt(now))
	case StatusMuted:
		matching = matching.Where(muteColumns.inForceAt(now))
	case StatusAdmin:
		matching = matching.Where("role = ?", RoleAdmin)
	case StatusUnverified:
		matching = matching.Where("email_verified_at IS NULL")
	default:
		return Page{}, validation.New("status", "must be banned, muted, admin or unverified")
	}

	var total int64
	if err := matching.Count(&total).Error; err != nil {
		return Page{}, fmt.Errorf("counting accounts: %w", err)
	}

	var found []User
	if err := matching.Order("username_key").Limit(query.PageSize).Offset(query.Offset()).Find(&found).Error; err != nil {
		return Page{}, fmt.Errorf("searching accounts: %w", err)
	}

	return Page{Users: found, Total: int(total), Page: query.Page, PageSize: query.PageSize}, nil
}

// LockIn reads accounts by id to change them inside the caller's
// transaction, locking their rows until it ends. Rows are locked in id
// order, so transactions locking several accounts never wait on each other
// in a circle. withAdmins locks and returns every admin too, for a change
// that must leave at least one. Ids of accounts that do not exist are left
// out.
func LockIn(tx *gorm.DB, ids []string, withAdmins bool) (map[string]User, error) {
	query := tx.Clauses(clause.Locking{Strength: clause.LockingStrengthUpdate}).Where("id IN ?", ids)
	if withAdmins {
		query = query.Or("role = ?", RoleAdmin)
	}

	var locked []User
	if err := query.Order("id").Find(&locked).Error; err != nil {
		return nil, fmt.Errorf("locking accounts: %w", err)
	}

	byID := make(map[string]User, len(locked))
	for _, user := range locked {
		byID[user.ID] = user
	}
	return byID, nil
}

// LockAdminIn locks the account acting, inside the transaction making its
// change, and returns it, or ErrNotAdmin if at now it is no longer an admin.
func LockAdminIn(tx *gorm.DB, id string, now time.Time) (User, error) {
	locked, err := LockIn(tx, []string{id}, false)
	if err != nil {
		return User{}, err
	}

	admin, found := locked[id]
	if _, banned := admin.BanAt(now); !found || !admin.IsAdmin() || banned {
		return User{}, ErrNotAdmin
	}
	return admin, nil
}

// SetRoleIn changes an account's role inside the caller's transaction. It
// returns ErrNotFound if the account does not exist.
func SetRoleIn(tx *gorm.DB, id string, role Role) error {
	return update(tx, id, "changing a role", map[string]any{"role": role})
}

// SetBanIn bans an account inside the caller's transaction, replacing any
// earlier ban. It returns ErrNotFound if the account does not exist.
func SetBanIn(tx *gorm.DB, id string, ban Ban) error {
	return update(tx, id, "banning an account", map[string]any{
		banColumns.since: ban.Since,
		banColumns.until: ban.Until,
		"ban_reason":     ban.Reason,
	})
}

// ClearBanIn lifts an account's ban inside the caller's transaction. It
// returns ErrNotFound if the account does not exist.
func ClearBanIn(tx *gorm.DB, id string) error {
	return update(tx, id, "lifting a ban", map[string]any{
		banColumns.since: nil,
		banColumns.until: nil,
		"ban_reason":     "",
	})
}

// SetMuteIn mutes an account inside the caller's transaction, replacing any
// earlier mute. It returns ErrNotFound if the account does not exist.
func SetMuteIn(tx *gorm.DB, id string, mute Penalty) error {
	return update(tx, id, "muting an account", map[string]any{
		muteColumns.since: mute.Since,
		muteColumns.until: mute.Until,
	})
}

// ClearMuteIn lifts an account's mute inside the caller's transaction. It
// returns ErrNotFound if the account does not exist.
func ClearMuteIn(tx *gorm.DB, id string) error {
	return update(tx, id, "lifting a mute", map[string]any{
		muteColumns.since: nil,
		muteColumns.until: nil,
	})
}

// update changes an account's columns. It returns ErrNotFound if the account
// does not exist.
func update(db *gorm.DB, id, doing string, columns map[string]any) error {
	result := db.Model(&User{}).Where("id = ?", id).Updates(columns)
	if result.Error != nil {
		return fmt.Errorf("%s: %w", doing, result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}
