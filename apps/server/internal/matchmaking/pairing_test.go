package matchmaking

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var noon = time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)

func waitingFor(userID string, rating int, waited time.Duration) Ticket {
	return Ticket{UserID: userID, Username: userID, Rating: rating, Since: noon.Add(-waited)}
}

func everyoneOnline(string) bool { return true }

func TestTheRangeWidensTheLongerAPlayerWaits(t *testing.T) {
	rules := pairing{start: 100, now: noon}

	for _, test := range []struct {
		waited time.Duration
		reach  int
	}{
		{0, 100},
		{14 * time.Second, 100},
		{15 * time.Second, 200},
		{30 * time.Second, 300},
		{59 * time.Second, 400},
		{time.Minute, math.MaxInt},
	} {
		assert.Equal(t, test.reach, rules.reach(waitingFor("shobu", 1000, test.waited)), "after %s", test.waited)
	}
}

func TestPlayersSuitEachOtherWithinEitherOnesRange(t *testing.T) {
	rules := pairing{start: 100, now: noon}
	newcomer := waitingFor("newcomer", 1000, 0)

	assert.True(t, rules.suits(newcomer, waitingFor("close", 1100, 0)), "the range includes its edge")
	assert.False(t, rules.suits(newcomer, waitingFor("far", 1101, 0)))
	assert.True(t, rules.suits(newcomer, waitingFor("patient", 1250, 30*time.Second)),
		"a player who waited long suits a newcomer further away")
	assert.True(t, rules.suits(waitingFor("patient", 1250, 30*time.Second), newcomer), "both ways")
	assert.True(t, rules.suits(newcomer, waitingFor("anyone", 2400, time.Minute)))
}

func TestTheBestPairIsTheClosestInRating(t *testing.T) {
	rules := pairing{start: 100, now: noon}
	shobu := waitingFor("shobu", 1000, 0)
	waiting := []Ticket{
		waitingFor("kokujo", 1090, 20*time.Second),
		waitingFor("far", 1500, 10*time.Second),
		waitingFor("mimi", 960, 5*time.Second),
		waitingFor("level", 1040, 0),
	}

	assert.Equal(t, 2, rules.best(shobu, waiting, everyoneOnline), "the closest, not the longest waiting")
	assert.Equal(t, 0, rules.best(shobu, waiting[:2], everyoneOnline))
	assert.Equal(t, -1, rules.best(shobu, waiting[1:2], everyoneOnline), "nobody within range")
	assert.Equal(t, -1, rules.best(shobu, []Ticket{shobu}, everyoneOnline), "never themselves")

	assert.Equal(t, 3, rules.best(shobu, waiting, func(userID string) bool { return userID != "mimi" }),
		"only players online")
	tied := []Ticket{waitingFor("first", 1040, 20*time.Second), waitingFor("second", 960, 10*time.Second)}
	assert.Equal(t, 0, rules.best(shobu, tied, everyoneOnline), "level gaps go to the longest waiting")
}
