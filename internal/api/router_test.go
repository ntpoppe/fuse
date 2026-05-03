package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ntpoppe/fuse/internal/api"
)

func TestHealthEndpoint_StatusOK(t *testing.T) {
	router := api.NewRouter()
	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestHealthEndpoint_ContentType(t *testing.T) {
	router := api.NewRouter()
	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	ct := rec.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", ct)
	}
}

func TestHealthEndpoint_Body(t *testing.T) {
	router := api.NewRouter()
	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if body["status"] != "ok" {
		t.Errorf("expected status 'ok', got %q", body["status"])
	}
}
