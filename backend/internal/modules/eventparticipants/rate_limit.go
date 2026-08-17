package eventparticipants

import (
	"sync"
	"time"
)

type RateLimiter interface {
	Allow(key string) bool
}

type rateBucket struct {
	count   int
	resetAt time.Time
}

// MemoryRateLimiter — fixed-window limiter для текущего single-process deployment.
// Интерфейс позволяет заменить реализацию без изменения handlers.
type MemoryRateLimiter struct {
	mu          sync.Mutex
	buckets     map[string]rateBucket
	maxAttempts int
	window      time.Duration
	now         func() time.Time
}

func NewMemoryRateLimiter(maxAttempts int, window time.Duration) *MemoryRateLimiter {
	if maxAttempts <= 0 {
		maxAttempts = 10
	}
	if window <= 0 {
		window = 5 * time.Minute
	}
	return &MemoryRateLimiter{
		buckets: make(map[string]rateBucket), maxAttempts: maxAttempts,
		window: window, now: time.Now,
	}
}

func (l *MemoryRateLimiter) Allow(key string) bool {
	now := l.now()
	l.mu.Lock()
	defer l.mu.Unlock()

	bucket, ok := l.buckets[key]
	if !ok || !now.Before(bucket.resetAt) {
		l.buckets[key] = rateBucket{count: 1, resetAt: now.Add(l.window)}
		return true
	}
	if bucket.count >= l.maxAttempts {
		return false
	}
	bucket.count++
	l.buckets[key] = bucket

	// Ограничиваем рост map при большом числе одноразовых IP/device keys.
	if len(l.buckets) > 10_000 {
		for bucketKey, candidate := range l.buckets {
			if !now.Before(candidate.resetAt) {
				delete(l.buckets, bucketKey)
			}
		}
	}
	return true
}
