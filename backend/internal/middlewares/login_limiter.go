package middlewares

import (
	"strings"
	"sync"
	"time"
)

const (
	loginMaxFailures = 5
	loginWindow      = 10 * time.Minute
	loginLockout     = 10 * time.Minute
)

type loginAttempts struct {
	failures    int
	windowStart time.Time
	lockedUntil time.Time
}

// LoginLimiter slows down password guessing: after 5 wrong passwords within 10 minutes the
// same client + username pair is locked for 10 minutes. State is kept in memory, which is
// enough for a single backend instance and resets on restart.
type LoginLimiter struct {
	mu       sync.Mutex
	attempts map[string]*loginAttempts
	now      func() time.Time
}

func NewLoginLimiter() *LoginLimiter {
	return &LoginLimiter{attempts: make(map[string]*loginAttempts), now: time.Now}
}

func loginKey(clientIP, username string) string {
	return clientIP + "|" + strings.ToLower(strings.TrimSpace(username))
}

// RetryAfter returns how long the pair is still locked; zero means it may try.
func (l *LoginLimiter) RetryAfter(clientIP, username string) time.Duration {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry, ok := l.attempts[loginKey(clientIP, username)]
	if !ok {
		return 0
	}
	if remaining := entry.lockedUntil.Sub(l.now()); remaining > 0 {
		return remaining
	}
	return 0
}

// Fail records a wrong password.
func (l *LoginLimiter) Fail(clientIP, username string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	l.prune(now)

	key := loginKey(clientIP, username)
	entry, ok := l.attempts[key]
	if !ok || now.Sub(entry.windowStart) > loginWindow {
		entry = &loginAttempts{windowStart: now}
		l.attempts[key] = entry
	}
	entry.failures++
	if entry.failures >= loginMaxFailures {
		entry.lockedUntil = now.Add(loginLockout)
		entry.failures = 0
		entry.windowStart = now
	}
}

// Success clears the history after a correct login.
func (l *LoginLimiter) Success(clientIP, username string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.attempts, loginKey(clientIP, username))
}

// prune drops entries that can no longer matter, so the map cannot grow without bound.
func (l *LoginLimiter) prune(now time.Time) {
	if len(l.attempts) < 1000 {
		return
	}
	for key, entry := range l.attempts {
		if now.After(entry.lockedUntil) && now.Sub(entry.windowStart) > loginWindow {
			delete(l.attempts, key)
		}
	}
}
