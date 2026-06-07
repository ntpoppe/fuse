package api_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ntpoppe/fuse/internal/api"
	connectionmanager "github.com/ntpoppe/fuse/internal/connection_manager"
	"github.com/ntpoppe/fuse/internal/executor"
	"github.com/ntpoppe/fuse/internal/registry"
	"github.com/ntpoppe/fuse/internal/storage"
	"github.com/ntpoppe/fuse/internal/testutil"
)

type apiEnv struct {
	router *http.ServeMux
	store  *storage.Store
	cm     *connectionmanager.ConnectionManager
}

func newAPIEnv(t *testing.T) apiEnv {
	t.Helper()

	store := storage.NewStore(testutil.OpenSQLiteMemory(t))
	if err := store.InitializeSchema(); err != nil {
		t.Fatalf("initialize schema: %v", err)
	}

	reg := registry.NewRegistry()
	cm := connectionmanager.NewConnectionManager(reg)
	exec := executor.NewExecutor(reg)

	return apiEnv{
		router: api.NewRouter(cm, store, exec),
		store:  store,
		cm:     cm,
	}
}

func doRequest(t *testing.T, handler http.Handler, method, path string, body io.Reader, contentType string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, path, body)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func doJSON(t *testing.T, handler http.Handler, method, path string, payload any) *httptest.ResponseRecorder {
	t.Helper()

	var body io.Reader
	if payload != nil {
		switch v := payload.(type) {
		case string:
			body = strings.NewReader(v)
		default:
			data, err := json.Marshal(v)
			if err != nil {
				t.Fatalf("marshal request body: %v", err)
			}
			body = strings.NewReader(string(data))
		}
	}

	return doRequest(t, handler, method, path, body, "application/json")
}

func assertStatus(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rec.Code != want {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, want, rec.Body.String())
	}
}

func assertBodyContains(t *testing.T, rec *httptest.ResponseRecorder, want string) {
	t.Helper()
	if !strings.Contains(rec.Body.String(), want) {
		t.Fatalf("body = %q, want substring %q", rec.Body.String(), want)
	}
}


func registerSQLiteConnection(t *testing.T, env apiEnv, id, dbPath string) {
	t.Helper()

	rec := doJSON(t, env.router, http.MethodPost, "/api/connections", map[string]string{
		"id":     id,
		"driver": "sqlite",
		"host":   dbPath,
	})
	assertStatus(t, rec, http.StatusCreated)

	t.Cleanup(func() {
		if err := env.cm.RemoveConnection(id); err != nil {
			t.Errorf("remove test connection %q: %v", id, err)
		}
	})
}

func TestHandler_Health(t *testing.T) {
	t.Parallel()

	env := newAPIEnv(t)
	rec := doRequest(t, env.router, http.MethodGet, "/health", nil, "")

	assertStatus(t, rec, http.StatusOK)

	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("content-type = %q, want application/json", got)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("status = %q, want ok", body["status"])
	}
}

func TestHandler_PostConnection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setup      func(t *testing.T, env apiEnv) any
		wantStatus int
		bodySubstr string
		verify     func(t *testing.T, env apiEnv)
	}{
		{
			name: "success",
			setup: func(t *testing.T, env apiEnv) any {
				driver := testutil.RegisterNamedMockDriver(t, "healthy", false)
				return map[string]string{
					"id": "conn1", "driver": driver, "host": "localhost:3306",
				}
			},
			wantStatus: http.StatusCreated,
			verify: func(t *testing.T, env apiEnv) {
				connections, err := env.store.GetAllConnections(testutil.Context(t))
				if err != nil {
					t.Fatalf("load saved connections: %v", err)
				}
				if len(connections) != 1 || connections[0].ID != "conn1" {
					t.Fatalf("saved connections = %+v, want one conn1 record", connections)
				}
			},
		},
		{
			name:       "invalid json",
			setup:      func(*testing.T, apiEnv) any { return "{invalid" },
			wantStatus: http.StatusBadRequest,
			bodySubstr: "Invalid JSON structure payload",
		},
		{
			name: "invalid driver",
			setup: func(*testing.T, apiEnv) any {
				return map[string]string{
					"id": "conn1", "driver": "non_existent_driver", "host": "localhost:9999",
				}
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "unreachable host",
			setup: func(t *testing.T, env apiEnv) any {
				driver := testutil.RegisterNamedMockDriver(t, "unreachable", true)
				return map[string]string{
					"id": "conn1", "driver": driver, "host": "localhost:3306",
				}
			},
			wantStatus: http.StatusBadRequest,
			bodySubstr: "failed to ping",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			env := newAPIEnv(t)
			rec := doJSON(t, env.router, http.MethodPost, "/api/connections", tt.setup(t, env))
			assertStatus(t, rec, tt.wantStatus)

			if tt.bodySubstr != "" {
				assertBodyContains(t, rec, tt.bodySubstr)
			}
			if tt.verify != nil {
				tt.verify(t, env)
			}
		})
	}
}

func TestHandler_PostConnection_DuplicateID(t *testing.T) {
	env := newAPIEnv(t)
	driver := testutil.RegisterNamedMockDriver(t, "duplicate", false)
	payload := map[string]string{
		"id": "conn1", "driver": driver, "host": "localhost:3306",
	}

	rec := doJSON(t, env.router, http.MethodPost, "/api/connections", payload)
	assertStatus(t, rec, http.StatusCreated)

	rec = doJSON(t, env.router, http.MethodPost, "/api/connections", payload)
	assertStatus(t, rec, http.StatusBadRequest)
	assertBodyContains(t, rec, "already exists")
}

func registerMockConnection(t *testing.T, env apiEnv, id string) {
	t.Helper()

	driver := testutil.RegisterNamedMockDriver(t, "conn", false)
	rec := doJSON(t, env.router, http.MethodPost, "/api/connections", map[string]string{
		"id": id, "driver": driver, "host": "localhost:3306",
	})
	assertStatus(t, rec, http.StatusCreated)
}

func TestHandler_DeleteConnection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setup      func(t *testing.T, env apiEnv) string
		wantStatus int
		bodySubstr string
		verify     func(t *testing.T, env apiEnv, id string)
	}{
		{
			name: "success",
			setup: func(t *testing.T, env apiEnv) string {
				id := "conn1"
				registerMockConnection(t, env, id)
				return id
			},
			wantStatus: http.StatusNoContent,
			verify: func(t *testing.T, env apiEnv, id string) {
				if err := env.cm.RemoveConnection(id); err == nil {
					t.Fatal("expected connection to be removed from registry")
				}

				connections, err := env.store.GetAllConnections(testutil.Context(t))
				if err != nil {
					t.Fatalf("load saved connections: %v", err)
				}
				if len(connections) != 0 {
					t.Fatalf("saved connections = %+v, want empty", connections)
				}
			},
		},
		{
			name:       "unknown connection",
			setup:      func(*testing.T, apiEnv) string { return "missing" },
			wantStatus: http.StatusNotFound,
			bodySubstr: "does not exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			env := newAPIEnv(t)
			id := tt.setup(t, env)

			rec := doRequest(t, env.router, http.MethodDelete, "/api/connections/"+id, nil, "")
			assertStatus(t, rec, tt.wantStatus)

			if tt.bodySubstr != "" {
				assertBodyContains(t, rec, tt.bodySubstr)
			}
			if tt.verify != nil {
				tt.verify(t, env, id)
			}
		})
	}
}

func TestHandler_PostQuery(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setup      func(t *testing.T, env apiEnv) string
		wantStatus int
		bodySubstr string
		verify     func(t *testing.T, rec *httptest.ResponseRecorder)
	}{
		{
			name: "success",
			setup: func(t *testing.T, env apiEnv) string {
				dbPath := testutil.SeedSQLiteFile(t, `
					CREATE TABLE items (
						id INTEGER PRIMARY KEY,
						name TEXT NOT NULL
					);
					INSERT INTO items (name) VALUES ('alpha'), ('beta');
				`)
				registerSQLiteConnection(t, env, "query_conn", dbPath)
				return `{"id":"query_conn","sql":"SELECT id, name FROM items ORDER BY id ASC"}`
			},
			wantStatus: http.StatusOK,
			verify: func(t *testing.T, rec *httptest.ResponseRecorder) {
				if got := rec.Header().Get("Content-Type"); got != "application/json" {
					t.Fatalf("content-type = %q, want application/json", got)
				}

				var results []map[string]any
				if err := json.NewDecoder(rec.Body).Decode(&results); err != nil {
					t.Fatalf("decode response: %v", err)
				}
				if len(results) != 2 || results[0]["name"] != "alpha" {
					t.Fatalf("results = %+v, want two rows starting with alpha", results)
				}
			},
		},
		{
			name:       "invalid json",
			setup:      func(*testing.T, apiEnv) string { return "not-json" },
			wantStatus: http.StatusBadRequest,
			bodySubstr: "Invalid JSON structure payload",
		},
		{
			name:       "missing id",
			setup:      func(*testing.T, apiEnv) string { return `{"sql":"SELECT 1"}` },
			wantStatus: http.StatusBadRequest,
			bodySubstr: "Missing mandatory",
		},
		{
			name:       "missing sql",
			setup:      func(*testing.T, apiEnv) string { return `{"id":"conn1"}` },
			wantStatus: http.StatusBadRequest,
			bodySubstr: "Missing mandatory",
		},
		{
			name:       "empty fields",
			setup:      func(*testing.T, apiEnv) string { return `{"id":"","sql":""}` },
			wantStatus: http.StatusBadRequest,
			bodySubstr: "Missing mandatory",
		},
		{
			name: "invalid sql",
			setup: func(t *testing.T, env apiEnv) string {
				dbPath := testutil.SeedSQLiteFile(t, `CREATE TABLE items (id INTEGER PRIMARY KEY);`)
				registerSQLiteConnection(t, env, "query_conn", dbPath)
				return `{"id":"query_conn","sql":"SELECT * FROM missing_table"}`
			},
			wantStatus: http.StatusInternalServerError,
			bodySubstr: "error querying",
		},
		{
			name:       "unknown connection",
			setup:      func(*testing.T, apiEnv) string { return `{"id":"missing_conn","sql":"SELECT 1"}` },
			wantStatus: http.StatusInternalServerError,
			bodySubstr: "does not exist in registry",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			env := newAPIEnv(t)
			rec := doJSON(t, env.router, http.MethodPost, "/api/query", tt.setup(t, env))
			assertStatus(t, rec, tt.wantStatus)

			if tt.bodySubstr != "" {
				assertBodyContains(t, rec, tt.bodySubstr)
			}
			if tt.verify != nil {
				tt.verify(t, rec)
			}
		})
	}
}
