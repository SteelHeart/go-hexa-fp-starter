package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiter throttles per client.
//
// ⚠️ In memory, hence PER INSTANCE: behind N replicas the effective limit is
// multiplied by N (see SECURITY.md, what this starter does not provide).
type RateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	rps      rate.Limit
	burst    int
	ttl      time.Duration
}

type visitor struct {
	limiter *rate.Limiter
	seen    time.Time
}

// NewRateLimiter builds a limiter. `ttl` is how long an idle client is kept
// before being forgotten, so the table does not grow without end.
func NewRateLimiter(rps float64, burst int, ttl time.Duration) *RateLimiter {
	return &RateLimiter{
		visitors: make(map[string]*visitor),
		rps:      rate.Limit(rps),
		burst:    burst,
		ttl:      ttl,
	}
}

// Middleware returns the throttling middleware.
func (l *RateLimiter) Middleware() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !l.allow(clientKey(r)) {
				w.Header().Set("Retry-After", "1")
				http.Error(w, `{"title":"Too Many Requests","status":429}`, http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (l *RateLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	v, found := l.visitors[key]
	if !found {
		// Eviction happens BEFORE insertion, and the new visitor is born
		// stamped. The reverse order — insert, then evict — deleted the visitor
		// just created: its `seen` still held the zero date, and
		// `now.Sub(zero)` exceeds any TTL.
		//
		// The consequence stayed invisible until a test looked for it: the
		// table stayed empty, every request started over with a fresh limiter,
		// and throttling throttled NOTHING. No symptom — a limiter that lets
		// everything through behaves exactly like a lightly used service.
		l.evictLocked(now)
		v = &visitor{limiter: rate.NewLimiter(l.rps, l.burst), seen: now}
		l.visitors[key] = v
	}
	v.seen = now
	return v.limiter.Allow()
}

// evictLocked drops idle clients. Called under lock, and only when a new client
// shows up: no cleanup goroutine to shut down.
func (l *RateLimiter) evictLocked(now time.Time) {
	for key, v := range l.visitors {
		if now.Sub(v.seen) > l.ttl {
			delete(l.visitors, key)
		}
	}
}

// clientKey identifies the caller. RemoteAddr only: an unvalidated
// X-Forwarded-For header would be forgeable, so the limit would be bypassable.
// The trusted proxy will have to rewrite RemoteAddr itself.
func clientKey(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
