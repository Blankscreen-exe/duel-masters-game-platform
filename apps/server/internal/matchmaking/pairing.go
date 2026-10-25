package matchmaking

import (
	"math"
	"time"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/settings"
)

// RatingRange is the admin setting for how far apart in rating quick match
// pairs players who have only just started waiting.
var RatingRange = settings.Number{
	Key:         "matchmaking.rating_range",
	Group:       "Quick match",
	Description: "How far apart in rating, at most, quick match pairs players who have just started waiting. The range grows by 100 points for every 15 seconds they wait, and after a minute anyone can be paired.",
	Default:     100,
	Min:         0,
	Max:         1000,
}

// Settings are the settings this feature declares.
func Settings() []settings.Definition {
	return []settings.Definition{RatingRange}
}

// How the rating range widens while a player waits.
const (
	widenBy     = 100
	widenEvery  = 15 * time.Second
	anyoneAfter = time.Minute
)

// pairing decides who suits whom at one moment.
type pairing struct {
	// start is the rating range of a player who has just started waiting.
	start int
	now   time.Time
}

// reach is how far apart in rating a waiting player may be paired.
func (p pairing) reach(ticket Ticket) int {
	waited := p.now.Sub(ticket.Since)
	if waited >= anyoneAfter {
		return math.MaxInt
	}
	return p.start + widenBy*int(max(waited, 0)/widenEvery)
}

// suits reports whether two waiting players may be paired: the gap between
// their ratings is within either one's reach, so a player who has waited
// long may be paired with a newcomer further away.
func (p pairing) suits(a, b Ticket) bool {
	gap := ratingGap(a, b)
	return gap <= p.reach(a) || gap <= p.reach(b)
}

// best returns the index of the player in waiting best paired with ticket,
// of those it suits and online allows: the closest in rating, and of those
// the longest waiting. It returns -1 for none.
func (p pairing) best(ticket Ticket, waiting []Ticket, online func(userID string) bool) int {
	found := -1
	for i, candidate := range waiting {
		if candidate.UserID == ticket.UserID || !p.suits(ticket, candidate) || !online(candidate.UserID) {
			continue
		}
		if found < 0 || ratingGap(ticket, candidate) < ratingGap(ticket, waiting[found]) {
			found = i
		}
	}
	return found
}

func ratingGap(a, b Ticket) int {
	if a.Rating > b.Rating {
		return a.Rating - b.Rating
	}
	return b.Rating - a.Rating
}
