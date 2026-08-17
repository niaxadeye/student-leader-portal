package eventparticipants

import (
	"testing"
	"time"
)

func TestMemoryRateLimiter(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	limiter := NewMemoryRateLimiter(2, time.Minute)
	limiter.now = func() time.Time { return now }

	if !limiter.Allow("event|ip|device") || !limiter.Allow("event|ip|device") {
		t.Fatal("first two attempts must be allowed")
	}
	if limiter.Allow("event|ip|device") {
		t.Fatal("third attempt must be blocked")
	}
	if !limiter.Allow("another-key") {
		t.Fatal("independent key must be allowed")
	}
	now = now.Add(time.Minute)
	if !limiter.Allow("event|ip|device") {
		t.Fatal("attempt after window must be allowed")
	}
}
