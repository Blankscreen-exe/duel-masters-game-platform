package ratings_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/ratings"
)

func TestExpectedScore(t *testing.T) {
	assert.InDelta(t, 0.5, ratings.ExpectedScore(1000, 1000), 1e-9, "equal players are even")
	assert.InDelta(t, 0.76, ratings.ExpectedScore(1200, 1000), 0.01, "200 points better wins about three games in four")
	assert.InDelta(t, 1, ratings.ExpectedScore(1200, 1000)+ratings.ExpectedScore(1000, 1200), 1e-9)
}

func TestChange(t *testing.T) {
	assert.Equal(t, 16, ratings.Change(1000, 1000, 32), "half the speed between equal players")
	assert.Equal(t, 24, ratings.Change(1000, 1200, 32), "beating a stronger player is worth more")
	assert.Equal(t, 8, ratings.Change(1200, 1000, 32), "and a weaker one less")
	assert.Equal(t, 1, ratings.Change(2400, 1000, 32), "a win is always worth something")
	assert.Equal(t, 5, ratings.Change(1000, 1000, 10), "the speed scales every change")
}

func TestNextStreak(t *testing.T) {
	for _, test := range []struct {
		streak int
		won    bool
		next   int
	}{
		{0, true, 1},
		{0, false, -1},
		{2, true, 3},
		{2, false, -1},
		{-3, false, -4},
		{-3, true, 1},
	} {
		assert.Equal(t, test.next, ratings.NextStreak(test.streak, test.won), "%+v", test)
	}
}
