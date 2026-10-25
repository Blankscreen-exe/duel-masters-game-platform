// Package ratings keeps the players' ratings: an Elo rating each, shown as a
// named tier, their wins, losses and streaks, and the rankings. Games are
// rated as they are saved (see games.Rater), so a rating always matches the
// match history.
package ratings

import (
	"math"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/settings"
)

// StartingRating is every new player's rating.
const StartingRating = 1000

// Speed is the admin setting for how far one game can move a rating: the
// Elo formula's K-factor.
var Speed = settings.Number{
	Key:         "ratings.speed",
	Group:       "Ratings",
	Description: "How many rating points one game can move at most (the Elo K-factor). A game between equally rated players moves both ratings by half of it.",
	Default:     32,
	Min:         1,
	Max:         100,
}

// Settings are the settings this feature declares.
func Settings() []settings.Definition {
	return []settings.Definition{Speed}
}

// ExpectedScore is the chance that a player rated rating beats one rated
// opponent, by the Elo formula.
func ExpectedScore(rating, opponent int) float64 {
	return 1 / (1 + math.Pow(10, float64(opponent-rating)/400))
}

// Change is the points a winner gains and their loser loses, at speed. It is
// always at least 1, so a win is never worth nothing, even against a much
// weaker opponent.
func Change(winner, loser int, speed int64) int {
	return max(1, int(math.Round(float64(speed)*(1-ExpectedScore(winner, loser)))))
}

// NextStreak is a streak after a win or a loss: a win extends a win streak or
// starts one, and a loss likewise. Win streaks are above zero, loss streaks
// below.
func NextStreak(streak int, won bool) int {
	switch {
	case won && streak > 0:
		return streak + 1
	case won:
		return 1
	case streak < 0:
		return streak - 1
	default:
		return -1
	}
}
