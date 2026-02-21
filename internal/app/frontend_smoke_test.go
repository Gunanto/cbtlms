package app

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestFrontendSmokePublicRoutes(t *testing.T) {
	restore := chdirToRepoRoot(t)
	defer restore()

	router := NewRouter(Config{
		CSRFEnforced:        false,
		AuthRateLimitPerMin: 60,
	}, nil)

	tests := []struct {
		name       string
		method     string
		target     string
		wantStatus int
	}{
		{name: "home", method: http.MethodGet, target: "/", wantStatus: http.StatusOK},
		{name: "login", method: http.MethodGet, target: "/login", wantStatus: http.StatusOK},
		{name: "ujian_pick", method: http.MethodGet, target: "/ujian", wantStatus: http.StatusOK},
		{name: "simulasi_redirect", method: http.MethodGet, target: "/simulasi", wantStatus: http.StatusFound},
		{name: "authoring", method: http.MethodGet, target: "/authoring", wantStatus: http.StatusOK},
		{name: "admin", method: http.MethodGet, target: "/admin", wantStatus: http.StatusOK},
		{name: "attempt_page", method: http.MethodGet, target: "/ujian/123", wantStatus: http.StatusOK},
		{name: "result_page", method: http.MethodGet, target: "/hasil/123", wantStatus: http.StatusOK},
		{name: "healthz", method: http.MethodGet, target: "/healthz", wantStatus: http.StatusOK},
		{name: "static_css", method: http.MethodGet, target: "/static/css/app.css?v=test", wantStatus: http.StatusOK},
		{name: "static_js", method: http.MethodGet, target: "/static/js/app.js?v=test", wantStatus: http.StatusOK},
		{name: "auth_me_unauthorized", method: http.MethodGet, target: "/api/v1/auth/me", wantStatus: http.StatusUnauthorized},
		{name: "login_invalid_body", method: http.MethodPost, target: "/api/v1/auth/login-password", wantStatus: http.StatusBadRequest},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.target, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			if w.Code != tc.wantStatus {
				t.Fatalf("%s %s: got status %d, want %d", tc.method, tc.target, w.Code, tc.wantStatus)
			}
		})
	}
}

func chdirToRepoRoot(t *testing.T) func() {
	t.Helper()

	start, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	dir := start
	for {
		if fileExists(filepath.Join(dir, "go.mod")) && fileExists(filepath.Join(dir, "web", "templates", "layout", "base.html")) {
			if err := os.Chdir(dir); err != nil {
				t.Fatalf("chdir to repo root %s: %v", dir, err)
			}
			return func() {
				_ = os.Chdir(start)
			}
		}

		next := filepath.Dir(dir)
		if next == dir {
			t.Fatalf("repo root not found from %s", start)
		}
		dir = next
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
