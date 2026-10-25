// Package clocktest is a clock for tests, which stands still until the test
// moves it, so whatever depends on time comes out exactly the same every run.
package clocktest

import (
	"sync"
	"time"
)

// Clock is a time that only moves when Advance moves it. It is safe for
// concurrent use.
type Clock struct {
	mu  sync.Mutex
	now time.Time
}

// New returns a clock standing at start.
func New(start time.Time) *Clock {
	return &Clock{now: start}
}

// Now is the clock's time, as the features' clocks are called.
func (c *Clock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

// Advance moves the clock on.
func (c *Clock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}
