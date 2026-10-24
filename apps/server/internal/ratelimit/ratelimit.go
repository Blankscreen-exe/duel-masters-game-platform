// Package ratelimit limits how often something may be attempted per key, such
// as failed logins per account, in fixed time windows.
//
// State lives in the server's memory, which suits this project: one server
// runs per database.
package ratelimit

import (
	"sync"
	"time"
)

// Limiter allows at most a number of attempts per key in each window. A key's
// window starts with its first attempt. It is safe for concurrent use.
type Limiter struct {
	maxAttempts int
	window      time.Duration
	now         func() time.Time

	mu      sync.Mutex
	windows map[string]*window
	// lastSweep is when expired windows were last removed, so keys never seen
	// again do not pile up.
	lastSweep time.Time
}

type window struct {
	attempts int
	resetsAt time.Time
}

// New returns a limiter allowing maxAttempts per window of the given length.
// now is the clock; pass time.Now outside tests.
func New(maxAttempts int, length time.Duration, now func() time.Time) *Limiter {
	return &Limiter{
		maxAttempts: maxAttempts,
		window:      length,
		now:         now,
		windows:     make(map[string]*window),
		lastSweep:   now(),
	}
}

// Allowed reports whether key may make another attempt, and if not, how long
// until it may.
func (l *Limiter) Allowed(key string) (allowed bool, retryAfter time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	current := l.active(key, now)
	if current == nil || current.attempts < l.maxAttempts {
		return true, 0
	}

	return false, current.resetsAt.Sub(now)
}

// Record counts an attempt by key.
func (l *Limiter) Record(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	l.sweep(now)

	if current := l.active(key, now); current != nil {
		current.attempts++
		return
	}

	l.windows[key] = &window{attempts: 1, resetsAt: now.Add(l.window)}
}

// Reset forgets key's attempts, such as after a successful login.
func (l *Limiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	delete(l.windows, key)
}

// active returns key's window, or nil once it has ended. The caller holds mu.
func (l *Limiter) active(key string, now time.Time) *window {
	current, ok := l.windows[key]
	if !ok {
		return nil
	}
	if !now.Before(current.resetsAt) {
		delete(l.windows, key)
		return nil
	}

	return current
}

// sweep removes ended windows at most once per window length, so memory stays
// bounded by the keys attempted within one window. The caller holds mu.
func (l *Limiter) sweep(now time.Time) {
	if now.Sub(l.lastSweep) < l.window {
		return
	}

	for key, current := range l.windows {
		if !now.Before(current.resetsAt) {
			delete(l.windows, key)
		}
	}
	l.lastSweep = now
}
