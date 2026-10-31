package admin_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/admin"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/adminlog"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/auth"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/clocktest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/mail"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/settings"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/store/storetest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/users"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/validation"
)

const password = "correct horse battery staple"

// effects notes what the admin actions put into effect on the live site.
type effects struct {
	mu           sync.Mutex
	disconnected []string
	expelled     []string
	changed      []string
}

func (e *effects) Disconnect(userID string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.disconnected = append(e.disconnected, userID)
}

func (e *effects) Expel(userID string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.expelled = append(e.expelled, userID)
}

func (e *effects) AccountChanged(userID string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.changed = append(e.changed, userID)
}

func (e *effects) seen() (disconnected, expelled, changed []string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.disconnected, e.expelled, e.changed
}

type fixture struct {
	admin   *admin.Service
	auth    *auth.Service
	users   *users.Service
	log     *adminlog.Service
	effects *effects
	clock   *clocktest.Clock
	db      *gorm.DB
}

func newFixture(t *testing.T, db *gorm.DB) fixture {
	t.Helper()

	siteSettings, err := settings.NewService(context.Background(), db, storetest.Logger(), auth.Settings()...)
	require.NoError(t, err)

	clock := clocktest.New(time.Now().UTC())
	accounts := users.NewService(db, storetest.Logger())
	mailService := mail.NewService(db, nil, storetest.Logger())
	authService := auth.NewService(db, accounts, siteSettings, auth.NewPasswordHasher(4), mailService, clock.Now, storetest.Logger())
	seen := &effects{}

	return fixture{
		admin: admin.NewService(admin.Dependencies{
			DB:          db,
			Auth:        authService,
			Connections: seen,
			Games:       seen,
			Listener:    seen,
			Logger:      storetest.Logger(),
			Now:         clock.Now,
		}),
		auth:    authService,
		users:   accounts,
		log:     adminlog.NewService(db),
		effects: seen,
		clock:   clock,
		db:      db,
	}
}

// signUp creates an account; the first one is the admin.
func (f fixture) signUp(t *testing.T, username string) (users.User, auth.Session) {
	t.Helper()

	user, session, err := f.auth.SignUp(context.Background(), username, username+"@example.test", password)
	if err == nil && !user.IsVerified() {
		now := f.clock.Now().UTC().Truncate(time.Microsecond)
		err = users.ConfirmEmailIn(f.db, user.ID, now)
		user.EmailVerifiedAt = &now
	}
	require.NoError(t, err)
	return user, session
}

func (f fixture) logEntries(t *testing.T) []adminlog.Entry {
	t.Helper()

	page, err := f.log.List(context.Background(), adminlog.Query{})
	require.NoError(t, err)
	return page.Entries
}

func TestABanEndsThePlayersSessionsAndPutsThemOut(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		f := newFixture(t, db)
		boss, _ := f.signUp(t, "Boss")
		player, session := f.signUp(t, "Shobu")

		banned, err := f.admin.Ban(ctx, boss, player.ID, admin.BanInput{Duration: 7 * 24 * time.Hour, Reason: "  Spamming the chat  "})
		require.NoError(t, err)

		ban, isBanned := banned.BanAt(f.clock.Now())
		require.True(t, isBanned)
		assert.Equal(t, "Spamming the chat", ban.Reason)
		require.NotNil(t, ban.Until)
		assert.WithinDuration(t, f.clock.Now().Add(7*24*time.Hour), *ban.Until, time.Second)

		_, err = f.auth.Authenticate(ctx, session.Token)
		assert.ErrorIs(t, err, auth.ErrNoSession, "the ban ends the player's sessions")

		disconnected, expelled, _ := f.effects.seen()
		assert.Equal(t, []string{player.ID}, disconnected)
		assert.Equal(t, []string{player.ID}, expelled)

		entries := f.logEntries(t)
		require.Len(t, entries, 1)
		assert.Equal(t, adminlog.UserBanned, entries[0].Action)
		assert.Equal(t, "Boss", entries[0].AdminName)
		assert.Equal(t, boss.ID, entries[0].AdminID)
		assert.Equal(t, "Shobu", entries[0].Subject)
		assert.Equal(t, "For 7 days: Spamming the chat", entries[0].Details)
	})
}

func TestABanForGoodLastsUntilLifted(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		f := newFixture(t, db)
		boss, _ := f.signUp(t, "Boss")
		player, _ := f.signUp(t, "Shobu")

		_, err := f.admin.Ban(ctx, boss, player.ID, admin.BanInput{Reason: "Cheating"})
		require.NoError(t, err)

		f.clock.Advance(10 * 365 * 24 * time.Hour)
		_, _, err = f.auth.LogIn(ctx, "Shobu", password)
		var banned *auth.BannedError
		require.ErrorAs(t, err, &banned)
		assert.Nil(t, banned.Ban.Until)
		assert.Equal(t, "Cheating", banned.Ban.Reason)

		lifted, err := f.admin.Unban(ctx, boss, player.ID)
		require.NoError(t, err)
		_, stillBanned := lifted.BanAt(f.clock.Now())
		assert.False(t, stillBanned)

		_, _, err = f.auth.LogIn(ctx, "Shobu", password)
		require.NoError(t, err)

		entries := f.logEntries(t)
		require.Len(t, entries, 2)
		assert.Equal(t, adminlog.UserUnbanned, entries[0].Action, "the latest first")
		assert.Equal(t, "For good: Cheating", entries[1].Details)
	})
}

func TestATimedBanLiftsItself(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		f := newFixture(t, db)
		boss, _ := f.signUp(t, "Boss")
		player, _ := f.signUp(t, "Shobu")

		_, err := f.admin.Ban(ctx, boss, player.ID, admin.BanInput{Duration: 24 * time.Hour, Reason: "Cooling off"})
		require.NoError(t, err)

		f.clock.Advance(24*time.Hour - time.Minute)
		_, _, err = f.auth.LogIn(ctx, "Shobu", password)
		var banned *auth.BannedError
		require.ErrorAs(t, err, &banned)

		f.clock.Advance(time.Minute)
		_, session, err := f.auth.LogIn(ctx, "Shobu", password)
		require.NoError(t, err, "the ban is over")
		_, err = f.auth.Authenticate(ctx, session.Token)
		require.NoError(t, err)

		// Lifting a ban that is over changes nothing.
		_, err = f.admin.Unban(ctx, boss, player.ID)
		require.NoError(t, err)
		assert.Len(t, f.logEntries(t), 1)
	})
}

func TestBanAndMuteLimits(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		f := newFixture(t, db)
		boss, _ := f.signUp(t, "Boss")
		player, _ := f.signUp(t, "Shobu")

		for input, field := range map[admin.BanInput]string{
			{Reason: "   "}:                                      "reason",
			{Reason: "Two\nlines"}:                               "reason",
			{Reason: string(make([]byte, 201))}:                  "reason",
			{Duration: 30 * time.Second, Reason: "Too short"}:    "duration",
			{Duration: 366 * 24 * time.Hour, Reason: "Too long"}: "duration",
			{Duration: -time.Hour, Reason: "Backwards"}:          "duration",
		} {
			_, err := f.admin.Ban(ctx, boss, player.ID, input)
			invalid, ok := validation.As(err)
			require.True(t, ok, "%+v: got %v", input, err)
			assert.Equal(t, field, invalid.Field)
		}

		_, err := f.admin.Mute(ctx, boss, player.ID, 400*24*time.Hour)
		invalid, ok := validation.As(err)
		require.True(t, ok, "got %v", err)
		assert.Equal(t, "duration", invalid.Field)

		assert.Empty(t, f.logEntries(t), "nothing was done")
	})
}

func TestAMuteKeepsAPlayerOutOfTheChatUntilItEnds(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		f := newFixture(t, db)
		boss, _ := f.signUp(t, "Boss")
		player, session := f.signUp(t, "Shobu")

		muted, err := f.admin.Mute(ctx, boss, player.ID, time.Hour)
		require.NoError(t, err)
		mute, isMuted := muted.MuteAt(f.clock.Now())
		require.True(t, isMuted)
		require.NotNil(t, mute.Until)

		_, err = f.auth.Authenticate(ctx, session.Token)
		require.NoError(t, err, "a muted player stays logged in")
		_, _, changed := f.effects.seen()
		assert.Equal(t, []string{player.ID}, changed, "the player's pages hear of the mute")

		f.clock.Advance(time.Hour)
		_, isMuted = muted.MuteAt(f.clock.Now())
		assert.False(t, isMuted, "the mute lifts itself")

		entries := f.logEntries(t)
		require.Len(t, entries, 1)
		assert.Equal(t, adminlog.UserMuted, entries[0].Action)
		assert.Equal(t, "For 1 hour", entries[0].Details)
	})
}

func TestUnmuteLiftsAMute(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		f := newFixture(t, db)
		boss, _ := f.signUp(t, "Boss")
		player, _ := f.signUp(t, "Shobu")

		_, err := f.admin.Mute(ctx, boss, player.ID, 0)
		require.NoError(t, err)
		unmuted, err := f.admin.Unmute(ctx, boss, player.ID)
		require.NoError(t, err)
		_, isMuted := unmuted.MuteAt(f.clock.Now())
		assert.False(t, isMuted)

		// Lifting it again changes nothing.
		_, err = f.admin.Unmute(ctx, boss, player.ID)
		require.NoError(t, err)

		entries := f.logEntries(t)
		require.Len(t, entries, 2)
		assert.Equal(t, adminlog.UserUnmuted, entries[0].Action)
		assert.Equal(t, "For good", entries[1].Details)
		_, _, changed := f.effects.seen()
		assert.Len(t, changed, 2)
	})
}

func TestNoAdminActsOnTheirOwnAccount(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		f := newFixture(t, db)
		boss, _ := f.signUp(t, "Boss")

		_, err := f.admin.Ban(ctx, boss, boss.ID, admin.BanInput{Reason: "Oops"})
		assert.ErrorIs(t, err, admin.ErrOwnAccount)
		_, err = f.admin.Mute(ctx, boss, boss.ID, time.Hour)
		assert.ErrorIs(t, err, admin.ErrOwnAccount)
		_, err = f.admin.SetRole(ctx, boss, boss.ID, users.RolePlayer)
		assert.ErrorIs(t, err, admin.ErrOwnAccount)
		_, err = f.admin.SetPassword(ctx, boss, boss.ID, "another long password")
		assert.ErrorIs(t, err, admin.ErrOwnAccount)

		assert.Empty(t, f.logEntries(t))
	})
}

func TestOnlyAdminsAct(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		f := newFixture(t, db)
		boss, _ := f.signUp(t, "Boss")
		player, _ := f.signUp(t, "Shobu")
		other, _ := f.signUp(t, "Kokujo")

		_, err := f.admin.Mute(ctx, player, other.ID, time.Hour)
		assert.ErrorIs(t, err, admin.ErrNotAdmin)

		// An admin made a player since their page loaded can no longer act.
		_, err = f.admin.SetRole(ctx, boss, player.ID, users.RoleAdmin)
		require.NoError(t, err)
		promoted, err := f.users.ByID(ctx, player.ID)
		require.NoError(t, err)
		_, err = f.admin.SetRole(ctx, boss, player.ID, users.RolePlayer)
		require.NoError(t, err)
		_, err = f.admin.Mute(ctx, promoted, other.ID, time.Hour)
		assert.ErrorIs(t, err, admin.ErrNotAdmin)

		_, err = f.admin.Mute(ctx, boss, "00000000-0000-0000-0000-000000000000", time.Hour)
		assert.ErrorIs(t, err, users.ErrNotFound)
	})
}

func TestAnAdminMustBeMadeAPlayerBeforeABan(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		f := newFixture(t, db)
		boss, _ := f.signUp(t, "Boss")
		other, _ := f.signUp(t, "Kokujo")
		_, err := f.admin.SetRole(ctx, boss, other.ID, users.RoleAdmin)
		require.NoError(t, err)

		_, err = f.admin.Ban(ctx, boss, other.ID, admin.BanInput{Reason: "Rogue admin"})
		assert.ErrorIs(t, err, admin.ErrAdminBan)

		_, err = f.admin.SetRole(ctx, boss, other.ID, users.RolePlayer)
		require.NoError(t, err)
		_, err = f.admin.Ban(ctx, boss, other.ID, admin.BanInput{Reason: "Rogue admin"})
		require.NoError(t, err)

		_, err = f.admin.SetRole(ctx, boss, other.ID, users.RoleAdmin)
		assert.ErrorIs(t, err, admin.ErrBannedAdmin, "a banned player cannot be made an admin")
	})
}

func TestRoleChangesAreLogged(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		f := newFixture(t, db)
		boss, _ := f.signUp(t, "Boss")
		player, _ := f.signUp(t, "Shobu")

		promoted, err := f.admin.SetRole(ctx, boss, player.ID, users.RoleAdmin)
		require.NoError(t, err)
		assert.True(t, promoted.IsAdmin())

		// Asking for the role the player already has changes nothing.
		_, err = f.admin.SetRole(ctx, boss, player.ID, users.RoleAdmin)
		require.NoError(t, err)

		demoted, err := f.admin.SetRole(ctx, boss, player.ID, users.RolePlayer)
		require.NoError(t, err)
		assert.False(t, demoted.IsAdmin())

		entries := f.logEntries(t)
		require.Len(t, entries, 2)
		assert.Equal(t, "Made a player", entries[0].Details)
		assert.Equal(t, "Made an admin", entries[1].Details)
		_, _, changed := f.effects.seen()
		assert.Equal(t, []string{player.ID, player.ID}, changed)

		_, err = f.admin.SetRole(ctx, boss, player.ID, users.Role("owner"))
		assert.Error(t, err)
	})
}

func TestTwoAdminsMakingEachOtherPlayersLeaveOne(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		f := newFixture(t, db)
		first, _ := f.signUp(t, "Boss")
		second, _ := f.signUp(t, "Deputy")
		_, err := f.admin.SetRole(ctx, first, second.ID, users.RoleAdmin)
		require.NoError(t, err)
		second, err = f.users.ByID(ctx, second.ID)
		require.NoError(t, err)

		var (
			wg   sync.WaitGroup
			errs [2]error
		)
		for i, pair := range [][2]users.User{{first, second}, {second, first}} {
			wg.Go(func() {
				_, errs[i] = f.admin.SetRole(ctx, pair[0], pair[1].ID, users.RolePlayer)
			})
		}
		wg.Wait()

		succeeded := 0
		for _, err := range errs {
			if err == nil {
				succeeded++
				continue
			}
			assert.ErrorIs(t, err, admin.ErrNotAdmin, "the slower one is no longer an admin")
		}
		assert.Equal(t, 1, succeeded)

		page, err := f.users.Search(ctx, users.Query{Status: users.StatusAdmin}, f.clock.Now())
		require.NoError(t, err)
		assert.Equal(t, 1, page.Total, "one admin is left")
	})
}

func TestSetPasswordGivesAPlayerANewOne(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		f := newFixture(t, db)
		boss, _ := f.signUp(t, "Boss")
		player, session := f.signUp(t, "Shobu")

		// The player has been guessing at their forgotten password.
		for range 5 {
			_, _, err := f.auth.LogIn(ctx, "Shobu", "not the password")
			require.ErrorIs(t, err, auth.ErrInvalidCredentials)
		}

		_, err := f.admin.SetPassword(ctx, boss, player.ID, "short")
		invalid, ok := validation.As(err)
		require.True(t, ok, "got %v", err)
		assert.Equal(t, "password", invalid.Field)

		_, err = f.admin.SetPassword(ctx, boss, player.ID, "a brand new password")
		require.NoError(t, err)

		_, err = f.auth.Authenticate(ctx, session.Token)
		assert.ErrorIs(t, err, auth.ErrNoSession, "the player's sessions end")
		_, _, err = f.auth.LogIn(ctx, "Shobu", password)
		assert.ErrorIs(t, err, auth.ErrInvalidCredentials, "the old password no longer works")
		_, _, err = f.auth.LogIn(ctx, "Shobu", "a brand new password")
		require.NoError(t, err, "the new one works at once, despite the wrong guesses")

		disconnected, expelled, _ := f.effects.seen()
		assert.Equal(t, []string{player.ID}, disconnected)
		assert.Empty(t, expelled, "a new password does not stop the player's game")

		entries := f.logEntries(t)
		require.Len(t, entries, 1)
		assert.Equal(t, adminlog.PasswordReset, entries[0].Action)
		assert.Equal(t, "Shobu", entries[0].Subject)
	})
}

func TestAdminsCannotChangeAComputerOpponentsAccount(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		f := newFixture(t, db)
		boss, _ := f.signUp(t, "Boss")
		computer, err := f.users.CreateComputer(ctx, "Grimwhisper")
		require.NoError(t, err)

		_, err = f.admin.Ban(ctx, boss, computer.ID, admin.BanInput{Reason: "Too good"})
		assert.ErrorIs(t, err, admin.ErrComputerAccount)
		_, err = f.admin.Mute(ctx, boss, computer.ID, time.Hour)
		assert.ErrorIs(t, err, admin.ErrComputerAccount)
		_, err = f.admin.SetRole(ctx, boss, computer.ID, users.RoleAdmin)
		assert.ErrorIs(t, err, admin.ErrComputerAccount)
		_, err = f.admin.SetPassword(ctx, boss, computer.ID, "another long password")
		assert.ErrorIs(t, err, admin.ErrComputerAccount)

		assert.Empty(t, f.logEntries(t))
		stored, err := f.users.ByID(ctx, computer.ID)
		require.NoError(t, err)
		assert.Empty(t, stored.PasswordHash, "still nobody can log in to it")
	})
}
