package api

import (
	"math"
	"net/http"
	"sync"
	"time"
)

// tokenBucket per client
type tokenBucket struct {
	mu       sync.Mutex
	tokens   float64
	last     time.Time
	rate     float64
	capacity float64
}

var buckets sync.Map // map[string]*tokenBucket

// RateLimitAuth returns middleware that limits requests per client (ip or forwarded).
// rate: tokens per second, capacity: burst capacity.
func RateLimitAuth(rate float64, capacity float64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := clientIP(r)
			now := time.Now()
			v, _ := buckets.LoadOrStore(ip, &tokenBucket{
				tokens:   capacity,
				last:     now,
				rate:     rate,
				capacity: capacity,
			})
			b := v.(*tokenBucket)

			b.mu.Lock()
			elapsed := now.Sub(b.last).Seconds()
			b.tokens = math.Min(b.capacity, b.tokens+elapsed*b.rate)
			b.last = now
			if b.tokens < 1 {
				b.mu.Unlock()
				ErrorJSON(w, http.StatusTooManyRequests, "rate limit exceeded")
				return
			}
			b.tokens -= 1
			b.mu.Unlock()

			next.ServeHTTP(w, r)
		})
	}
}
