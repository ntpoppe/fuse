package runtime_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ntpoppe/fuse/internal/config"
	"github.com/ntpoppe/fuse/internal/runtime"
)

func TestFromConfig_Production(t *testing.T) {
	t.Parallel()

	cfg := config.NewConfig()
	p := runtime.FromConfig(cfg)

	if p.MaxQueryRows != config.DefaultMaxQueryRows {
		t.Fatalf("MaxQueryRows = %d, want %d", p.MaxQueryRows, config.DefaultMaxQueryRows)
	}
	if !p.HTTP.AllowConnectionChanges {
		t.Fatal("expected connection changes allowed in production profile")
	}
	if p.HTTP.QueryTimeout != 0 {
		t.Fatalf("QueryTimeout = %v, want 0", p.HTTP.QueryTimeout)
	}
	if p.HTTP.RateLimit.RequestsPerSecond != 0 {
		t.Fatal("expected rate limit disabled in production profile")
	}
}

func TestFromConfig_Demo(t *testing.T) {
	t.Parallel()

	cfg := config.NewConfig()
	cfg.DemoMode = true
	cfg.MaxQueryRows = config.DefaultDemoMaxQueryRows

	p := runtime.FromConfig(cfg)

	if p.HTTP.AllowConnectionChanges {
		t.Fatal("expected connection changes disabled in demo profile")
	}
	if p.HTTP.QueryTimeout == 0 {
		t.Fatal("expected query timeout in demo profile")
	}
	if p.HTTP.RateLimit.RequestsPerSecond <= 0 {
		t.Fatal("expected rate limit in demo profile")
	}
	if p.HTTP.MaxBodyBytes != 64<<10 {
		t.Fatalf("MaxBodyBytes = %d, want 65536", p.HTTP.MaxBodyBytes)
	}
}

func TestWithCORS_AllowedOrigin(t *testing.T) {
	t.Parallel()

	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	handler := runtime.WrapHandler(inner, runtime.Profile{
		HTTP: runtime.HTTPProfile{CORSOrigins: []string{"http://localhost:8080"}},
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("Origin", "http://localhost:8080")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:8080" {
		t.Fatalf("allow-origin = %q", got)
	}
	if !called {
		t.Fatal("inner handler not called")
	}
}

func TestWithCORS_Preflight(t *testing.T) {
	t.Parallel()

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("inner handler should not run for OPTIONS")
	})

	handler := runtime.WrapHandler(inner, runtime.Profile{
		HTTP: runtime.HTTPProfile{CORSOrigins: []string{"http://localhost:8080"}},
	})

	req := httptest.NewRequest(http.MethodOptions, "/api/query", nil)
	req.Header.Set("Origin", "http://localhost:8080")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
}

func TestWithRateLimit(t *testing.T) {
	t.Parallel()

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := runtime.WrapHandler(inner, runtime.Profile{
		HTTP: runtime.HTTPProfile{
			RateLimit: runtime.RateLimit{RequestsPerSecond: 1, Burst: 1},
		},
	})

	req := func() *http.Request {
		return httptest.NewRequest(http.MethodPost, "/api/query", nil)
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req())
	if rec.Code != http.StatusOK {
		t.Fatalf("first request status = %d, want 200", rec.Code)
	}

	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req())
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("second request status = %d, want 429", rec.Code)
	}
}
