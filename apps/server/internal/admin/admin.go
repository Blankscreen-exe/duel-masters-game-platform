// Package admin carries out what admins do in the admin panel: banning and
// muting players, changing roles, passwords and settings. Each action checks the
// guards (no admin acts on their own account, an admin is not banned, the
// last admin stays), makes its change and records it in the admin log in one
// transaction, and then puts it into effect on the live site, such as
// putting a banned player out of their game.
package admin

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/adminlog"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/auth"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/settings"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/users"
)

var (
	// ErrNotAdmin is returned when the account acting is not, or no longer,
	// an admin.
	ErrNotAdmin = errors.New("only admins can do this")
	// ErrOwnAccount is returned to an admin acting on their own account.
	ErrOwnAccount = errors.New("admins cannot do this to their own account")
	// ErrAdminBan is returned for banning an admin, who must be made a
	// player first.
	ErrAdminBan = errors.New("an admin cannot be banned; make them a player first")
	// ErrBannedAdmin is returned for making a banned player an admin.
	ErrBannedAdmin = errors.New("a banned player cannot be made an admin")
	// ErrLastAdmin is returned for making the last admin a player.
	ErrLastAdmin = errors.New("the last admin cannot be made a player")
	// ErrComputerAccount is returned for acting on a computer opponent's
	// account, which nobody logs in to.
	ErrComputerAccount = errors.New("a computer opponent's account cannot be changed")
)

// Connections are the players' live connections to the site.
type Connections interface {
	// Disconnect closes every live connection of a player, whose pages then
	// show them logged out.
	Disconnect(userID string)
}

// Games are the running games.
type Games interface {
	// Expel stops the game a player plays in and closes their connections
	// to the games they watch.
	Expel(userID string)
}

// Listener hears about changes to accounts their players should see at
// once, such as a mute. It is called after the change is saved and must not
// block.
type Listener interface {
	AccountChanged(userID string)
}

// Dependencies are what the admin actions need.
type Dependencies struct {
	DB          *gorm.DB
	Auth        *auth.Service
	Connections Connections
	Games       Games
	Settings    *settings.Service
	Listener    Listener
	Logger      *slog.Logger
	// Now is the clock; pass time.Now outside tests.
	Now func() time.Time
}

// Service carries out the admin actions. It is safe for concurrent use.
type Service struct {
	db          *gorm.DB
	auth        *auth.Service
	connections Connections
	games       Games
	settings    *settings.Service
	listener    Listener
	logger      *slog.Logger
	now         func() time.Time
}

// NewService returns the admin actions.
func NewService(deps Dependencies) *Service {
	return &Service{
		db:          deps.DB,
		auth:        deps.Auth,
		connections: deps.Connections,
		games:       deps.Games,
		settings:    deps.Settings,
		listener:    deps.Listener,
		logger:      deps.Logger,
		now:         deps.Now,
	}
}

// timestamp is the current time as the database stores it: UTC, to the
// microsecond.
func (s *Service) timestamp() time.Time {
	return s.now().UTC().Truncate(time.Microsecond)
}

// accountChange is one admin action on a player's account, made inside the
// transaction that holds the locked accounts' rows: the admin's, the
// player's and, when asked for, every admin's. It returns whether it changed
// anything: an action that would change nothing, such as lifting a ban that
// is over, records nothing either.
type accountChange func(tx *gorm.DB, admin, player users.User, locked map[string]users.User) (changed bool, err error)

// changeAccount makes an admin's change to a player's account in one
// transaction, after checking the admin still is one and is not acting on
// their own account. withAdmins also locks every admin, for a change that
// must leave at least one. It returns the account as the change left it,
// and whether it changed.
func (s *Service) changeAccount(ctx context.Context, actor users.User, playerID string, withAdmins bool, change accountChange) (users.User, bool, error) {
	if playerID == actor.ID {
		return users.User{}, false, ErrOwnAccount
	}

	var (
		player  users.User
		changed bool
	)
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		locked, err := users.LockIn(tx, []string{actor.ID, playerID}, withAdmins)
		if err != nil {
			return err
		}

		admin, found := locked[actor.ID]
		if _, banned := admin.BanAt(s.timestamp()); !found || !admin.IsAdmin() || banned {
			return ErrNotAdmin
		}
		player, found = locked[playerID]
		if !found {
			return users.ErrNotFound
		}
		if player.Computer {
			return ErrComputerAccount
		}

		if changed, err = change(tx, admin, player, locked); err != nil || !changed {
			return err
		}

		// Read into a new value: reading into one already filled would keep
		// the old values of columns now null.
		var changedPlayer users.User
		if err := tx.Where("id = ?", playerID).Take(&changedPlayer).Error; err != nil {
			return fmt.Errorf("reading the changed account: %w", err)
		}
		player = changedPlayer
		return nil
	})
	if err != nil {
		return users.User{}, false, err
	}

	return player, changed, nil
}

// countAdmins counts the admins among locked accounts.
func countAdmins(locked map[string]users.User) int {
	admins := 0
	for _, user := range locked {
		if user.IsAdmin() {
			admins++
		}
	}
	return admins
}

// SetRole makes a player an admin or an admin a player. It returns
// ErrBannedAdmin for making a banned player an admin and ErrLastAdmin for
// making the last admin a player, besides the errors every action returns:
// ErrNotAdmin, ErrOwnAccount, ErrComputerAccount and users.ErrNotFound.
func (s *Service) SetRole(ctx context.Context, actor users.User, playerID string, role users.Role) (users.User, error) {
	if role != users.RolePlayer && role != users.RoleAdmin {
		return users.User{}, fmt.Errorf("setting a role: unknown role %q", role)
	}

	player, changed, err := s.changeAccount(ctx, actor, playerID, true, func(tx *gorm.DB, admin, player users.User, locked map[string]users.User) (bool, error) {
		if player.Role == role {
			return false, nil
		}
		if _, banned := player.BanAt(s.timestamp()); banned && role == users.RoleAdmin {
			return false, ErrBannedAdmin
		}
		if role == users.RolePlayer && countAdmins(locked) <= 1 {
			return false, ErrLastAdmin
		}

		if err := users.SetRoleIn(tx, player.ID, role); err != nil {
			return false, err
		}
		details := "Made a player"
		if role == users.RoleAdmin {
			details = "Made an admin"
		}
		return true, adminlog.RecordIn(tx, admin, adminlog.RoleChanged, player.Username, details)
	})
	if err != nil {
		return users.User{}, err
	}

	if changed {
		s.logger.InfoContext(ctx, "changed a role", "admin", actor.Username, "player", player.Username, "role", role)
		s.listener.AccountChanged(player.ID)
	}
	return player, nil
}

// SetPassword gives a player a new password, as an admin does for one who
// has forgotten theirs, and ends the player's sessions. It returns a
// *validation.Error for a password breaking the rules, besides the errors
// every action returns: ErrNotAdmin, ErrOwnAccount, ErrComputerAccount and users.ErrNotFound.
func (s *Service) SetPassword(ctx context.Context, actor users.User, playerID, password string) (users.User, error) {
	if playerID == actor.ID {
		return users.User{}, ErrOwnAccount
	}

	// Hashing is slow, so it is done before the transaction.
	hash, err := s.auth.HashNewPassword(ctx, password)
	if err != nil {
		return users.User{}, err
	}

	player, _, err := s.changeAccount(ctx, actor, playerID, false, func(tx *gorm.DB, admin, player users.User, _ map[string]users.User) (bool, error) {
		if err := users.SetPasswordHashIn(tx, player.ID, hash); err != nil {
			return false, err
		}
		if err := auth.EndSessionsIn(tx, player.ID); err != nil {
			return false, err
		}
		return true, adminlog.RecordIn(tx, admin, adminlog.PasswordReset, player.Username, "")
	})
	if err != nil {
		return users.User{}, err
	}

	s.auth.ForgetWrongPasswords(player.Username)
	s.connections.Disconnect(player.ID)
	s.logger.InfoContext(ctx, "reset a password", "admin", actor.Username, "player", player.Username)
	return player, nil
}
