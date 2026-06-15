package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ntpoppe/fuse/internal/api"
	connectionmanager "github.com/ntpoppe/fuse/internal/connection_manager"
	"github.com/ntpoppe/fuse/internal/config"
	"github.com/ntpoppe/fuse/internal/driver"
	"github.com/ntpoppe/fuse/internal/executor"
	"github.com/ntpoppe/fuse/internal/registry"
	"github.com/ntpoppe/fuse/internal/runtime"
	"github.com/ntpoppe/fuse/internal/storage"
	"github.com/ntpoppe/fuse/internal/testutil"
)

type apiEnv struct {
	handler http.Handler
	store   *storage.Store
	cm      *connectionmanager.ConnectionManager
}

func newAPIEnv(t *testing.T) apiEnv {
	return newAPIEnvWithMaxRows(t, config.DefaultMaxQueryRows)
}

func newAPIEnvWithMaxRows(t *testing.T, maxRows int) apiEnv {
	return newAPIEnvWithHTTPProfile(t, maxRows, runtime.HTTPProfile{AllowConnectionChanges: true})
}

func newAPIEnvWithHTTPProfile(t *testing.T, maxRows int, httpProfile runtime.HTTPProfile) apiEnv {
	t.Helper()

	store := storage.NewStore(testutil.OpenSQLiteMemory(t))
	if err := store.InitializeSchema(); err != nil {
		t.Fatalf("initialize schema: %v", err)
	}

	reg := registry.NewRegistry()
	cm := connectionmanager.NewConnectionManager(reg)
	exec := executor.NewExecutor(reg, maxRows)
	fedExec := executor.NewFederatedExecutor(reg, maxRows)

	if httpProfile.MaxBodyBytes == 0 {
		httpProfile.MaxBodyBytes = 1 << 20
	}

	return apiEnv{
		handler: api.NewRouter(cm, store, exec, fedExec, httpProfile),
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

	return doRequest(t, handler, method, path, body, api.ContentTypeJSON)
}

func assertStatus(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rec.Code != want {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, want, rec.Body.String())
	}
}


func assertJSONErrorContains(t *testing.T, rec *httptest.ResponseRecorder, want string) {
	t.Helper()

	var body struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error body: %v; raw = %q", err, rec.Body.String())
	}
	if !strings.Contains(body.Error, want) {
		t.Fatalf("error = %q, want substring %q", body.Error, want)
	}
}

func registerSQLiteConnection(t *testing.T, env apiEnv, id, dbPath string) {
	t.Helper()

	rec := doJSON(t, env.handler, http.MethodPost, api.PathConnections, map[string]string{
		"id":     id,
		"driver": driver.DriverSQLite,
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
	rec := doRequest(t, env.handler, http.MethodGet, api.PathHealth, nil, "")

	assertStatus(t, rec, http.StatusOK)

	if got := rec.Header().Get("Content-Type"); got != api.ContentTypeJSON {
		t.Fatalf("content-type = %q, want %s", got, api.ContentTypeJSON)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("status = %q, want ok", body["status"])
	}
}

func TestHandler_GetConnections(t *testing.T) {
	t.Parallel()

	env := newAPIEnv(t)
	registerMockConnection(t, env, "conn1")

	rec := doRequest(t, env.handler, http.MethodGet, api.PathConnections, nil, "")
	assertStatus(t, rec, http.StatusOK)

	var connections []map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&connections); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if len(connections) != 1 || connections[0]["id"] != "conn1" {
		t.Fatalf("connections = %+v, want one conn1 record", connections)
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
			bodySubstr: "invalid JSON payload",
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
			bodySubstr: "ping connection",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			env := newAPIEnv(t)
			rec := doJSON(t, env.handler, http.MethodPost, api.PathConnections, tt.setup(t, env))
			assertStatus(t, rec, tt.wantStatus)

			if tt.bodySubstr != "" {
				assertJSONErrorContains(t, rec, tt.bodySubstr)
			}
			if tt.verify != nil {
				tt.verify(t, env)
			}
		})
	}
}

func TestHandler_PostConnection_FixedConnections(t *testing.T) {
	t.Parallel()

	env := newAPIEnvWithHTTPProfile(t, config.DefaultMaxQueryRows, runtime.HTTPProfile{AllowConnectionChanges: false, MaxBodyBytes: 1 << 20})
	driverName := testutil.RegisterNamedMockDriver(t, "demo-blocked", false)

	rec := doJSON(t, env.handler, http.MethodPost, api.PathConnections, map[string]string{
		"id": "conn1", "driver": driverName, "host": "localhost:3306",
	})
	assertStatus(t, rec, http.StatusForbidden)
	assertJSONErrorContains(t, rec, "connection changes")
}

func TestHandler_DeleteConnection_FixedConnections(t *testing.T) {
	t.Parallel()

	env := newAPIEnvWithHTTPProfile(t, config.DefaultMaxQueryRows, runtime.HTTPProfile{AllowConnectionChanges: false, MaxBodyBytes: 1 << 20})

	rec := doRequest(t, env.handler, http.MethodDelete, api.PathConnections+"/shop", nil, "")
	assertStatus(t, rec, http.StatusForbidden)
	assertJSONErrorContains(t, rec, "connection changes")
}

func TestHandler_PostConnection_DuplicateID(t *testing.T) {
	env := newAPIEnv(t)
	driver := testutil.RegisterNamedMockDriver(t, "duplicate", false)
	payload := map[string]string{
		"id": "conn1", "driver": driver, "host": "localhost:3306",
	}

	rec := doJSON(t, env.handler, http.MethodPost, api.PathConnections, payload)
	assertStatus(t, rec, http.StatusCreated)

	rec = doJSON(t, env.handler, http.MethodPost, api.PathConnections, payload)
	assertStatus(t, rec, http.StatusBadRequest)
	assertJSONErrorContains(t, rec, "already exists")
}

func registerMockConnection(t *testing.T, env apiEnv, id string) {
	t.Helper()

	driver := testutil.RegisterNamedMockDriver(t, id, false)
	rec := doJSON(t, env.handler, http.MethodPost, api.PathConnections, map[string]string{
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
			bodySubstr: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			env := newAPIEnv(t)
			id := tt.setup(t, env)

			rec := doRequest(t, env.handler, http.MethodDelete, api.PathConnections+"/"+id, nil, "")
			assertStatus(t, rec, tt.wantStatus)

			if tt.bodySubstr != "" {
				assertJSONErrorContains(t, rec, tt.bodySubstr)
			}
			if tt.verify != nil {
				tt.verify(t, env, id)
			}
		})
	}
}

func TestHandler_PostQuery(t *testing.T) {
	t.Parallel()

	validFederatedSQL := `SELECT u.id, u.name, o.total FROM billing.users u JOIN analytics.orders o ON u.id = o.user_id WHERE u.active = 1 LIMIT 100`

	tests := []struct {
		name       string
		setup      func(t *testing.T, env apiEnv) any
		wantStatus int
		bodySubstr string
		verify     func(t *testing.T, rec *httptest.ResponseRecorder)
	}{
		{
			name: "single success",
			setup: func(t *testing.T, env apiEnv) any {
				dbPath := testutil.SeedSQLiteFile(t, `
					CREATE TABLE items (
						id INTEGER PRIMARY KEY,
						name TEXT NOT NULL
					);
					INSERT INTO items (name) VALUES ('alpha'), ('beta');
				`)
				registerSQLiteConnection(t, env, "query_conn", dbPath)
				return map[string]string{
					"id":  "query_conn",
					"sql": "SELECT id, name FROM items ORDER BY id ASC",
				}
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
			name: "federated success",
			setup: func(t *testing.T, env apiEnv) any {
				registerFederatedSQLiteConnections(t, env)
				return map[string]string{"sql": validFederatedSQL}
			},
			wantStatus: http.StatusOK,
			verify: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var results []map[string]any
				if err := json.NewDecoder(rec.Body).Decode(&results); err != nil {
					t.Fatalf("decode response: %v", err)
				}
				if len(results) != 2 || results[0]["name"] != "alice" {
					t.Fatalf("results = %+v, want two rows starting with alice", results)
				}
			},
		},
		{
			name:       "invalid json",
			setup:      func(*testing.T, apiEnv) any { return "not-json" },
			wantStatus: http.StatusBadRequest,
			bodySubstr: "invalid JSON payload",
		},
		{
			name:       "missing sql",
			setup:      func(*testing.T, apiEnv) any { return map[string]string{"id": "conn1"} },
			wantStatus: http.StatusBadRequest,
			bodySubstr: "missing required field: sql",
		},
		{
			name:       "empty sql",
			setup:      func(*testing.T, apiEnv) any { return map[string]string{"id": "", "sql": ""} },
			wantStatus: http.StatusBadRequest,
			bodySubstr: "missing required field: sql",
		},
		{
			name:       "federated unqualified sql",
			setup:      func(*testing.T, apiEnv) any { return map[string]string{"sql": "SELECT u.id FROM users u"} },
			wantStatus: http.StatusBadRequest,
			bodySubstr: "connection_id.table",
		},
		{
			name: "single read-only violation",
			setup: func(t *testing.T, env apiEnv) any {
				dbPath := testutil.SeedSQLiteFile(t, `CREATE TABLE items (id INTEGER PRIMARY KEY);`)
				registerSQLiteConnection(t, env, "query_conn", dbPath)
				return map[string]string{
					"id":  "query_conn",
					"sql": "DELETE FROM items",
				}
			},
			wantStatus: http.StatusBadRequest,
			bodySubstr: "read-only violation",
		},
		{
			name: "single invalid sql",
			setup: func(t *testing.T, env apiEnv) any {
				dbPath := testutil.SeedSQLiteFile(t, `CREATE TABLE items (id INTEGER PRIMARY KEY);`)
				registerSQLiteConnection(t, env, "query_conn", dbPath)
				return map[string]string{
					"id":  "query_conn",
					"sql": "SELECT * FROM missing_table",
				}
			},
			wantStatus: http.StatusInternalServerError,
			bodySubstr: "query:",
		},
		{
			name:       "single unknown connection",
			setup:      func(*testing.T, apiEnv) any { return map[string]string{"id": "missing_conn", "sql": "SELECT 1"} },
			wantStatus: http.StatusNotFound,
			bodySubstr: "not found",
		},
		{
			name: "federated unknown connection",
			setup: func(t *testing.T, env apiEnv) any {
				registerMockConnection(t, env, "billing")
				return map[string]string{"sql": validFederatedSQL}
			},
			wantStatus: http.StatusNotFound,
			bodySubstr: "analytics",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			env := newAPIEnv(t)
			rec := doJSON(t, env.handler, http.MethodPost, api.PathQuery, tt.setup(t, env))
			assertStatus(t, rec, tt.wantStatus)

			if tt.bodySubstr != "" {
				assertJSONErrorContains(t, rec, tt.bodySubstr)
			}
			if tt.verify != nil {
				tt.verify(t, rec)
			}
		})
	}
}

func registerFederatedSQLiteConnections(t *testing.T, env apiEnv) {
	t.Helper()

	billingPath := testutil.SeedSQLiteFile(t, `
CREATE TABLE users (
	id INTEGER PRIMARY KEY,
	name TEXT NOT NULL,
	active INTEGER NOT NULL
);
INSERT INTO users (id, name, active) VALUES (1, 'alice', 1), (2, 'bob', 1);
`)
	analyticsPath := testutil.SeedSQLiteFile(t, `
CREATE TABLE orders (
	user_id INTEGER NOT NULL,
	total REAL NOT NULL
);
INSERT INTO orders (user_id, total) VALUES (1, 10.5), (2, 20.0);
`)

	registerSQLiteConnection(t, env, "billing", billingPath)
	registerSQLiteConnection(t, env, "analytics", analyticsPath)
}

type failingStore struct {
	inner *storage.Store
}

func (s *failingStore) GetAllConnections(ctx context.Context) ([]storage.SavedConnection, error) {
	return s.inner.GetAllConnections(ctx)
}

func (s *failingStore) SaveConnection(context.Context, storage.SavedConnection) error {
	return errors.New("forced storage failure")
}

func (s *failingStore) GetConnection(ctx context.Context, id string) (storage.SavedConnection, bool, error) {
	return s.inner.GetConnection(ctx, id)
}

func (s *failingStore) RemoveConnection(ctx context.Context, id string) error {
	return s.inner.RemoveConnection(ctx, id)
}

func TestHandler_PostConnection_StorageRollback(t *testing.T) {
	store := storage.NewStore(testutil.OpenSQLiteMemory(t))
	if err := store.InitializeSchema(); err != nil {
		t.Fatalf("initialize schema: %v", err)
	}

	reg := registry.NewRegistry()
	cm := connectionmanager.NewConnectionManager(reg)
	handler := api.NewRouter(cm, &failingStore{inner: store}, executor.NewExecutor(reg, config.DefaultMaxQueryRows), executor.NewFederatedExecutor(reg, config.DefaultMaxQueryRows), runtime.HTTPProfile{AllowConnectionChanges: true, MaxBodyBytes: 1 << 20})

	driverName := testutil.RegisterNamedMockDriver(t, "rollback", false)
	rec := doJSON(t, handler, http.MethodPost, api.PathConnections, map[string]string{
		"id": "conn1", "driver": driverName, "host": "localhost:3306",
	})
	assertStatus(t, rec, http.StatusInternalServerError)

	if _, found := reg.Fetch("conn1"); found {
		t.Fatal("expected registry entry to be rolled back after storage failure")
	}
}

func TestHandler_PostQuery_RowLimitExceeded(t *testing.T) {
	env := newAPIEnvWithMaxRows(t, 1)
	dbPath := testutil.SeedSQLiteFile(t, `
CREATE TABLE items (
	id INTEGER PRIMARY KEY,
	name TEXT NOT NULL
);
INSERT INTO items (name) VALUES ('alpha'), ('beta');
`)
	registerSQLiteConnection(t, env, "limited", dbPath)

	rec := doJSON(t, env.handler, http.MethodPost, api.PathQuery, map[string]string{
		"id":  "limited",
		"sql": "SELECT id, name FROM items ORDER BY id ASC",
	})
	assertStatus(t, rec, http.StatusBadRequest)
	assertJSONErrorContains(t, rec, "query returned more than 1 rows")
}

func TestHandler_PostQuery_FederatedRowLimitExceeded(t *testing.T) {
	env := newAPIEnvWithMaxRows(t, 1)
	registerFederatedSQLiteConnections(t, env)

	rec := doJSON(t, env.handler, http.MethodPost, api.PathQuery, map[string]string{
		"sql": `SELECT u.id, u.name, o.total FROM billing.users u JOIN analytics.orders o ON u.id = o.user_id WHERE u.active = 1`,
	})
	assertStatus(t, rec, http.StatusBadRequest)
	assertJSONErrorContains(t, rec, "query returned more than 1 rows")
}
