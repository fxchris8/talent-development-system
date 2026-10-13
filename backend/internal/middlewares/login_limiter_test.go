package middlewares

import (
	"testing"
	"time"
)

func TestLoginLimiterLocksAfterRepeatedFailures(t *testing.T) {
	now := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
	limiter := NewLoginLimiter()
	limiter.now = func() time.Time { return now }

	for i := 0; i < loginMaxFailures-1; i++ {
		limiter.Fail("1.1.1.1", "Admin")
	}
	if wait := limiter.RetryAfter("1.1.1.1", "admin"); wait != 0 {
		t.Fatalf("locked too early: %v", wait)
	}

	limiter.Fail("1.1.1.1", "admin ") // username is matched case-insensitively and trimmed
	if wait := limiter.RetryAfter("1.1.1.1", "admin"); wait != loginLockout {
		t.Fatalf("expected lockout of %v, got %v", loginLockout, wait)
	}
	if wait := limiter.RetryAfter("2.2.2.2", "admin"); wait != 0 {
		t.Fatalf("another client must not be locked, got %v", wait)
	}
	if wait := limiter.RetryAfter("1.1.1.1", "other"); wait != 0 {
		t.Fatalf("another username must not be locked, got %v", wait)
	}

	now = now.Add(loginLockout + time.Second)
	if wait := limiter.RetryAfter("1.1.1.1", "admin"); wait != 0 {
		t.Fatalf("lock must expire, got %v", wait)
	}
}

func TestLoginLimiterResetsOnSuccessAndAfterWindow(t *testing.T) {
	now := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
	limiter := NewLoginLimiter()
	limiter.now = func() time.Time { return now }

	for i := 0; i < loginMaxFailures-1; i++ {
		limiter.Fail("1.1.1.1", "admin")
	}
	limiter.Success("1.1.1.1", "admin")
	limiter.Fail("1.1.1.1", "admin")
	if wait := limiter.RetryAfter("1.1.1.1", "admin"); wait != 0 {
		t.Fatalf("success must clear earlier failures, got %v", wait)
	}

	// Old failures outside the window do not count towards a lock.
	for i := 0; i < loginMaxFailures-2; i++ {
		limiter.Fail("3.3.3.3", "admin")
	}
	now = now.Add(loginWindow + time.Second)
	limiter.Fail("3.3.3.3", "admin")
	limiter.Fail("3.3.3.3", "admin")
	if wait := limiter.RetryAfter("3.3.3.3", "admin"); wait != 0 {
		t.Fatalf("failures outside the window must not lock, got %v", wait)
	}
}
