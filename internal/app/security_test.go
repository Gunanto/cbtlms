package app

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIPRateLimiterAllow(t *testing.T) {
	l := NewIPRateLimiter(2, 0)
	if !l.Allow("k") || !l.Allow("k") {
		t.Fatalf("first two requests should pass")
	}
	if l.Allow("k") {
		t.Fatalf("third request should be blocked")
	}
}

func TestCSRFMiddlewareEnforced(t *testing.T) {
	mw := CSRFMiddleware(true)
	next := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/attempts/1/submit", nil)
	req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: "abc"})
	req.Header.Set(csrfHeaderName, "abc")
	w := httptest.NewRecorder()
	next.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestCSRFMiddlewareRejectsMissingToken(t *testing.T) {
	mw := CSRFMiddleware(true)
	next := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/attempts/1/submit", nil)
	w := httptest.NewRecorder()
	next.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}
