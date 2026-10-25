package ratelimit

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/clocktest"
)

func newTestLimiter() (*Limiter, *clocktest.Clock) {
	c := clocktest.New(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))
	return New(3, 15*time.Minute, c.Now), c
}

func TestLimiterAllowsUpToTheLimit(t *testing.T) {
	limiter, _ := newTestLimiter()

	for i := range 3 {
		allowed, _ := limiter.Allowed("alice")
		assert.True(t, allowed, "attempt %d", i+1)
		limiter.Record("alice")
	}

	allowed, retryAfter := limiter.Allowed("alice")
	assert.False(t, allowed)
	assert.Equal(t, 15*time.Minute, retryAfter)
}

func TestLimiterWindowStartsWithTheFirstAttempt(t *testing.T) {
	limiter, clock := newTestLimiter()

	limiter.Record("alice")
	clock.Advance(10 * time.Minute)
	limiter.Record("alice")
	limiter.Record("alice")

	allowed, retryAfter := limiter.Allowed("alice")
	assert.False(t, allowed)
	assert.Equal(t, 5*time.Minute, retryAfter, "the window ends 15 minutes after the first attempt")

	clock.Advance(5 * time.Minute)
	allowed, _ = limiter.Allowed("alice")
	assert.True(t, allowed, "a new window starts once the old one has ended")
}

func TestLimiterKeepsKeysApart(t *testing.T) {
	limiter, _ := newTestLimiter()

	for range 3 {
		limiter.Record("alice")
	}

	allowed, _ := limiter.Allowed("bob")
	assert.True(t, allowed)
}

func TestLimiterResetForgetsAttempts(t *testing.T) {
	limiter, _ := newTestLimiter()

	for range 3 {
		limiter.Record("alice")
	}
	limiter.Reset("alice")

	allowed, _ := limiter.Allowed("alice")
	assert.True(t, allowed)
}

func TestLimiterForgetsKeysWhoseWindowEnded(t *testing.T) {
	limiter, clock := newTestLimiter()

	for i := range 1000 {
		limiter.Record(fmt.Sprintf("guess-%d", i))
	}
	assert.Len(t, limiter.windows, 1000)

	clock.Advance(16 * time.Minute)
	limiter.Record("later")

	assert.Len(t, limiter.windows, 1, "ended windows are swept, so unused keys do not pile up")
}

func TestLimiterIsSafeForConcurrentUse(t *testing.T) {
	limiter, _ := newTestLimiter()

	var wg sync.WaitGroup
	for i := range 50 {
		wg.Go(func() {
			key := fmt.Sprintf("player-%d", i%5)
			limiter.Allowed(key)
			limiter.Record(key)
			if i%7 == 0 {
				limiter.Reset(key)
			}
		})
	}
	wg.Wait()
}

func TestTakeCountsOnlyAllowedAttempts(t *testing.T) {
	limiter, clock := newTestLimiter()

	for i := range 3 {
		allowed, _ := limiter.Take("alice")
		assert.True(t, allowed, "attempt %d", i+1)
	}

	clock.Advance(5 * time.Minute)
	allowed, retryAfter := limiter.Take("alice")
	assert.False(t, allowed)
	assert.Equal(t, 10*time.Minute, retryAfter, "the window started with the first attempt")

	clock.Advance(10 * time.Minute)
	allowed, _ = limiter.Take("alice")
	assert.True(t, allowed, "a refused attempt is not counted, and the window has ended")
}

func TestTakeAtTheSameMomentNeverPassesTheLimit(t *testing.T) {
	limiter, _ := newTestLimiter()

	var wg sync.WaitGroup
	var mu sync.Mutex
	taken := 0
	for range 50 {
		wg.Go(func() {
			if allowed, _ := limiter.Take("alice"); allowed {
				mu.Lock()
				taken++
				mu.Unlock()
			}
		})
	}
	wg.Wait()

	assert.Equal(t, 3, taken)
}
