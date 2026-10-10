package auth

import (
	"sync"
	"time"
)

// RateLimiter counts failed login attempts per client inside a sliding window
// and refuses further attempts once the allowance is exhausted (AC-23). It is
// safe for concurrent use.
type RateLimiter struct {
	mu       sync.Mutex
	max      int
	window   time.Duration
	failures map[string][]time.Time
	now      func() time.Time
}

// NewRateLimiter builds a limiter that blocks a client after max failed
// attempts within window. A non-positive max means "no limit".
func NewRateLimiter(max int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		max:      max,
		window:   window,
		failures: make(map[string][]time.Time),
		now:      time.Now,
	}
}

// Allow reports whether another attempt from key is permitted right now. It
// prunes attempts that have fallen out of the window.
func (l *RateLimiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.max <= 0 {
		return true
	}
	return len(l.recentLocked(key)) < l.max
}

// RecordFailure records one failed attempt from key.
func (l *RateLimiter) RecordFailure(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	recent := l.recentLocked(key)
	l.failures[key] = append(recent, l.now())
}

// Reset forgets every failure for key. It is called after a successful login so
// a legitimate user's earlier typos do not count against them.
func (l *RateLimiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.failures, key)
}

// recentLocked returns the still-counted failures for key and drops the expired
// ones from the map. The caller must hold l.mu.
func (l *RateLimiter) recentLocked(key string) []time.Time {
	cutoff := l.now().Add(-l.window)
	all := l.failures[key]
	recent := all[:0]
	for _, at := range all {
		if at.After(cutoff) {
			recent = append(recent, at)
		}
	}
	l.failures[key] = recent
	return recent
}
