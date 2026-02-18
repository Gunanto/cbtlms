package app

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"cbtlms/internal/app/apiresp"
)

const csrfCookieName = "cbtlms_csrf"
const csrfHeaderName = "X-CSRF-Token"

type rateBucket struct {
	Count      int
	WindowEnds time.Time
}

type IPRateLimiter struct {
	mu     sync.Mutex
	max    int
	window time.Duration
	store  map[string]rateBucket
}

func NewIPRateLimiter(max int, window time.Duration) *IPRateLimiter {
	if max <= 0 {
		max = 60
	}
	if window <= 0 {
		window = time.Minute
	}
	return &IPRateLimiter{
		max:    max,
		window: window,
		store:  make(map[string]rateBucket),
	}
}

func (l *IPRateLimiter) Allow(key string) bool {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()

	b := l.store[key]
	if now.After(b.WindowEnds) {
		b = rateBucket{Count: 0, WindowEnds: now.Add(l.window)}
	}
	if b.Count >= l.max {
		l.store[key] = b
		return false
	}
	b.Count++
	l.store[key] = b
	return true
}

func RateLimitMiddleware(l *IPRateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := strings.TrimSpace(r.RemoteAddr)
			key := ip + "|" + r.Method + "|" + r.URL.Path
			if !l.Allow(key) {
				apiresp.WriteLegacy(w, r, http.StatusTooManyRequests, false, nil, "rate limit exceeded")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func CSRFMiddleware(enforced bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !enforced {
				next.ServeHTTP(w, r)
				return
			}
			switch r.Method {
			case http.MethodGet, http.MethodHead, http.MethodOptions:
				next.ServeHTTP(w, r)
				return
			}

			c, err := r.Cookie(csrfCookieName)
			if err != nil || strings.TrimSpace(c.Value) == "" {
				apiresp.WriteLegacy(w, r, http.StatusForbidden, false, nil, "csrf token missing")
				return
			}
			h := strings.TrimSpace(r.Header.Get(csrfHeaderName))
			if h == "" || h != c.Value {
				apiresp.WriteLegacy(w, r, http.StatusForbidden, false, nil, "csrf token invalid")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
