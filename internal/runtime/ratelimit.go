package runtime

import (
	"sync"
	"time"
)

type tokenBucket struct {
	mu     sync.Mutex
	rate   float64
	burst  float64
	tokens float64
	last   time.Time
}

func newTokenBucket(rate float64, burst int) *tokenBucket {
	return &tokenBucket{
		rate:   rate,
		burst:  float64(burst),
		tokens: float64(burst),
		last:   time.Now(),
	}
}

func (b *tokenBucket) allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	b.tokens += now.Sub(b.last).Seconds() * b.rate
	b.last = now

	if b.tokens > b.burst {
		b.tokens = b.burst
	}
	if b.tokens < 1 {
		return false
	}

	b.tokens--
	return true
}

type rateLimitStore struct {
	burst int
	rate  float64
	mu    sync.Mutex
	byIP  map[string]*tokenBucket
}

func newRateLimitStore(cfg RateLimit) *rateLimitStore {
	return &rateLimitStore{
		rate:  cfg.RequestsPerSecond,
		burst: cfg.Burst,
		byIP:  make(map[string]*tokenBucket),
	}
}

func (s *rateLimitStore) allow(key string) bool {
	s.mu.Lock()
	bucket, ok := s.byIP[key]
	if !ok {
		bucket = newTokenBucket(s.rate, s.burst)
		s.byIP[key] = bucket
	}
	s.mu.Unlock()

	return bucket.allow()
}
