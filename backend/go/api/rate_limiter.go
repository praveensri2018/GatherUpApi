// backend/go/api/rate_limiter.go
package api

import (
	"net/http"
	"sync"
	"time"
)

type simpleBucket struct {
	last time.Time
	mu   sync.Mutex
}

var authBuckets = map[string]*simpleBucket{}
var authBucketsMu sync.Mutex

// Limit to N requests per IP per window (very simple)
func RateLimitAuth(window time.Duration, max int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr // production: use real IP parsing / X-Forwarded-For
			authBucketsMu.Lock()
			b, ok := authBuckets[ip]
			if !ok {
				b = &simpleBucket{last: time.Now()}
				authBuckets[ip] = b
			}
			authBucketsMu.Unlock()

			b.mu.Lock()
			if time.Since(b.last) < window {
				// simple: allow 1 per window; extend per needs
				b.mu.Unlock()
				ErrorJSON(w, http.StatusTooManyRequests, "rate limit exceeded")
				return
			}
			b.last = time.Now()
			b.mu.Unlock()

			next.ServeHTTP(w, r)
		})
	}
}
