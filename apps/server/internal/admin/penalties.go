package admin

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"gorm.io/gorm"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/adminlog"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/auth"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/users"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/validation"
)

// Ban and mute limits.
const (
	// MaxPenalty is the longest a timed ban or mute may last; longer ones
	// are for good.
	MaxPenalty = 365 * 24 * time.Hour
	// MinPenalty is the shortest.
	MinPenalty = time.Minute
	// BanReasonMaxLength is the longest ban reason, in characters.
	BanReasonMaxLength = 200
)

// BanInput is a ban as an admin asks for it.
type BanInput struct {
	// Duration is how long the ban lasts; 0 for good.
	Duration time.Duration
	// Reason is what the player is shown when they try to log in.
	Reason string
}

// Ban bans a player, ending their sessions, putting them out of the site and
// stopping the game they play in. A ban replaces the player's earlier one.
// It returns a *validation.Error for a duration or reason breaking the
// limits and ErrAdminBan for an admin, besides the errors every action
// returns: ErrNotAdmin, ErrOwnAccount, ErrComputerAccount and users.ErrNotFound.
func (s *Service) Ban(ctx context.Context, actor users.User, playerID string, input BanInput) (users.User, error) {
	if err := checkDuration(input.Duration); err != nil {
		return users.User{}, err
	}
	reason, err := normalizeReason(input.Reason)
	if err != nil {
		return users.User{}, err
	}

	player, _, err := s.changeAccount(ctx, actor, playerID, false, func(tx *gorm.DB, admin, player users.User, _ map[string]users.User) (bool, error) {
		if player.IsAdmin() {
			return false, ErrAdminBan
		}

		ban := users.Ban{Penalty: s.penalty(input.Duration), Reason: reason}
		if err := users.SetBanIn(tx, player.ID, ban); err != nil {
			return false, err
		}
		if err := auth.EndSessionsIn(tx, player.ID); err != nil {
			return false, err
		}
		return true, adminlog.RecordIn(tx, admin, adminlog.UserBanned, player.Username, fmt.Sprintf("%s: %s", describe(input.Duration), reason))
	})
	if err != nil {
		return users.User{}, err
	}

	s.connections.Disconnect(player.ID)
	s.games.Expel(player.ID)
	s.logger.InfoContext(ctx, "banned a player", "admin", actor.Username, "player", player.Username, "for", describe(input.Duration))
	return player, nil
}

// Unban lifts a player's ban. Lifting a ban that is over, or that never was,
// changes nothing. It returns the errors every action returns: ErrNotAdmin,
// ErrOwnAccount, ErrComputerAccount and users.ErrNotFound.
func (s *Service) Unban(ctx context.Context, actor users.User, playerID string) (users.User, error) {
	player, changed, err := s.changeAccount(ctx, actor, playerID, false, func(tx *gorm.DB, admin, player users.User, _ map[string]users.User) (bool, error) {
		if _, banned := player.BanAt(s.timestamp()); !banned {
			return false, nil
		}
		if err := users.ClearBanIn(tx, player.ID); err != nil {
			return false, err
		}
		return true, adminlog.RecordIn(tx, admin, adminlog.UserUnbanned, player.Username, "")
	})
	if err != nil {
		return users.User{}, err
	}

	if changed {
		s.logger.InfoContext(ctx, "lifted a ban", "admin", actor.Username, "player", player.Username)
	}
	return player, nil
}

// Mute keeps a player out of the lobby chat for duration, or for good when
// it is 0. A mute replaces the player's earlier one. It returns a
// *validation.Error for a duration breaking the limits, besides the errors
// every action returns: ErrNotAdmin, ErrOwnAccount, ErrComputerAccount and users.ErrNotFound.
func (s *Service) Mute(ctx context.Context, actor users.User, playerID string, duration time.Duration) (users.User, error) {
	if err := checkDuration(duration); err != nil {
		return users.User{}, err
	}

	player, _, err := s.changeAccount(ctx, actor, playerID, false, func(tx *gorm.DB, admin, player users.User, _ map[string]users.User) (bool, error) {
		if err := users.SetMuteIn(tx, player.ID, s.penalty(duration)); err != nil {
			return false, err
		}
		return true, adminlog.RecordIn(tx, admin, adminlog.UserMuted, player.Username, describe(duration))
	})
	if err != nil {
		return users.User{}, err
	}

	s.listener.AccountChanged(player.ID)
	s.logger.InfoContext(ctx, "muted a player", "admin", actor.Username, "player", player.Username, "for", describe(duration))
	return player, nil
}

// Unmute lifts a player's mute. Lifting a mute that is over, or that never
// was, changes nothing. It returns the errors every action returns:
// ErrNotAdmin, ErrOwnAccount, ErrComputerAccount and users.ErrNotFound.
func (s *Service) Unmute(ctx context.Context, actor users.User, playerID string) (users.User, error) {
	player, changed, err := s.changeAccount(ctx, actor, playerID, false, func(tx *gorm.DB, admin, player users.User, _ map[string]users.User) (bool, error) {
		if _, muted := player.MuteAt(s.timestamp()); !muted {
			return false, nil
		}
		if err := users.ClearMuteIn(tx, player.ID); err != nil {
			return false, err
		}
		return true, adminlog.RecordIn(tx, admin, adminlog.UserUnmuted, player.Username, "")
	})
	if err != nil {
		return users.User{}, err
	}

	if changed {
		s.listener.AccountChanged(player.ID)
		s.logger.InfoContext(ctx, "lifted a mute", "admin", actor.Username, "player", player.Username)
	}
	return player, nil
}

// penalty is a ban or mute starting now and lasting duration, or for good
// when it is 0.
func (s *Service) penalty(duration time.Duration) users.Penalty {
	now := s.timestamp()
	if duration == 0 {
		return users.Penalty{Since: now}
	}
	until := now.Add(duration)
	return users.Penalty{Since: now, Until: &until}
}

func checkDuration(duration time.Duration) error {
	if duration != 0 && (duration < MinPenalty || duration > MaxPenalty) {
		return validation.New("duration", fmt.Sprintf("must be between %s and %s, or none for good", lengthOf(MinPenalty), lengthOf(MaxPenalty)))
	}
	return nil
}

// normalizeReason makes a ban reason one line of printable text and checks
// its length.
func normalizeReason(reason string) (string, error) {
	if !utf8.ValidString(reason) || strings.ContainsFunc(reason, unicode.IsControl) {
		return "", validation.New("reason", "must be one line of text")
	}

	reason = strings.TrimSpace(reason)
	switch {
	case reason == "":
		return "", validation.New("reason", "must not be empty")
	case utf8.RuneCountInString(reason) > BanReasonMaxLength:
		return "", validation.New("reason", fmt.Sprintf("must be at most %d characters", BanReasonMaxLength))
	}
	return reason, nil
}

// describe says how long a ban or mute lasts, for the admin log: "For 7
// days", "For good".
func describe(duration time.Duration) string {
	if duration == 0 {
		return "For good"
	}
	return "For " + lengthOf(duration)
}

// lengthOf says a length of time in the largest whole unit: "90 minutes",
// "24 hours", "7 days".
func lengthOf(duration time.Duration) string {
	amount, unit := int64(duration/time.Minute), "minute"
	switch {
	case duration%(24*time.Hour) == 0:
		amount, unit = int64(duration/(24*time.Hour)), "day"
	case duration%time.Hour == 0:
		amount, unit = int64(duration/time.Hour), "hour"
	}
	if amount != 1 {
		unit += "s"
	}
	return fmt.Sprintf("%d %s", amount, unit)
}
