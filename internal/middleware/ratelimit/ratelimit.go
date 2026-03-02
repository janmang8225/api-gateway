package ratelimit

import (
	"net"
	"net/http"
	"sync"
	"time"
)

type bucket struct {
	tokens     float64
	maxTokens  float64
	refillRate float64
	lastRefill time.Time
	mu         sync.Mutex
}

func newBucket(maxTokens float64, refillRate float64) *bucket {
	return &bucket{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

func (b *bucket) allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.lastRefill).Seconds()
	b.lastRefill = now

	b.tokens = b.tokens + elapsed*b.refillRate
	if b.tokens > b.maxTokens {
		b.tokens = b.maxTokens
	}

	if b.tokens >= 1.0 {
		b.tokens -= 1.0
		return true
	}

	return false
}

type RateLimiter struct {
	buckets    map[string]*bucket
	mu         sync.Mutex
	maxTokens  float64
	refillRate float64
}

func NewRateLimiter(maxTokens float64, refillRate float64) *RateLimiter {
	return &RateLimiter{
		buckets:    make(map[string]*bucket),
		maxTokens:  maxTokens,
		refillRate: refillRate,
	}
}

func (rl *RateLimiter) getBucket(ip string) *bucket {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	b, exists := rl.buckets[ip]
	if !exists {
		b = newBucket(rl.maxTokens, rl.refillRate)
		rl.buckets[ip] = b
	}

	return b
}

func extractIP(remoteAddr string) string {
	ip, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		// if splitting fails, use as-is
		return remoteAddr
	}
	return ip
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := extractIP(r.RemoteAddr)

		b := rl.getBucket(ip)

		if !b.allow() {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte("429 Too Many Requests"))
			return
		}

		next.ServeHTTP(w, r)
	})
}