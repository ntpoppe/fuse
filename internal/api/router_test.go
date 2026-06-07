package api_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ntpoppe/fuse/internal/api"
	connectionmanager "github.com/ntpoppe/fuse/internal/connection_manager"
	"github.com/ntpoppe/fuse/internal/executor"
	"github.com/ntpoppe/fuse/internal/registry"
	"github.com/ntpoppe/fuse/internal/storage"

	_ "modernc.org/sqlite"
)

type testEnv struct {
	router *http.ServeMux
	store  *storage.Store
	cm     *connectionmanager.ConnectionManager
}

type mockDriver struct{ failPing bool }
type mockConn struct{ failPing bool }

func (d *mockDriver) Open(name string) (driver.Conn, error) {
	return &mockConn{failPing: d.failPing}, nil
}

func (c *mockConn) Ping(ctx context.Context) error {
	if c.failPing {
		return errors.New("network destination unreachable")
	}
	return nil
}

func (c *mockConn) Prepare(query string) (driver.Stmt, error) { return nil, nil }
func (c *mockConn) Close() error                              { return nil }
func (c *mockConn) Begin() (driver.Tx, error)                 { return nil, nil }

func setupTestEnv(t *testing.T) testEnv {
	t.Helper()

	stateDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test sqlite db: %v", err)
	}

	store := storage.NewStore(stateDB)
	if err := store.InitializeSchema(); err != nil {
		t.Fatalf("failed to init test storage schema: %v", err)
	}

	reg := registry.NewRegistry()
	cm := connectionmanager.NewConnectionManager(reg)
	exec := executor.NewExecutor(reg)

	return testEnv{
		router: api.NewRouter(cm, store, exec),
		store:  store,
		cm:     cm,
	}
}

func setupTestRouter(t *testing.T) *http.ServeMux {
	t.Helper()
	return setupTestEnv(t).router
}

func createSeededSQLiteDB(t *testing.T) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "target.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("failed to open seed sqlite db: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE items (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL
		);
		INSERT INTO items (name) VALUES ('alpha'), ('beta');
	`)
	if err != nil {
		t.Fatalf("failed to seed sqlite db: %v", err)
	}

	return path
}

func TestHealthEndpoint_StatusOK(t *testing.T) {
	router := setupTestRouter(t)

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestHealthEndpoint_ContentType(t *testing.T) {
	router := setupTestRouter(t)

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	ct := rec.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", ct)
	}
}

func TestHealthEndpoint_Body(t *testing.T) {
	router := setupTestRouter(t)

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

func TestPostConnection_Success(t *testing.T) {
	sql.Register("mock_api_healthy", &mockDriver{failPing: false})

	env := setupTestEnv(t)
	body := `{"id":"conn1","driver":"mock_api_healthy","host":"localhost:3306"}`

	req := httptest.NewRequest("POST", "/api/connections", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	env.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	connections, err := env.store.GetAllConnections(ctx)
	if err != nil {
		t.Fatalf("failed to load saved connections: %v", err)
	}
	if len(connections) != 1 {
		t.Fatalf("expected 1 saved connection, got %d", len(connections))
	}
	if connections[0].ID != "conn1" || connections[0].Driver != "mock_api_healthy" {
		t.Errorf("unexpected saved connection: %+v", connections[0])
	}
}

func TestPostConnection_InvalidJSON(t *testing.T) {
	router := setupTestRouter(t)

	req := httptest.NewRequest("POST", "/api/connections", strings.NewReader("{invalid"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Invalid JSON structure payload") {
		t.Errorf("expected invalid JSON error, got %q", rec.Body.String())
	}
}

func TestPostConnection_InvalidDriver(t *testing.T) {
	router := setupTestRouter(t)
	body := `{"id":"conn1","driver":"non_existent_driver","host":"localhost:9999"}`

	req := httptest.NewRequest("POST", "/api/connections", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func TestPostConnection_UnreachableHost(t *testing.T) {
	sql.Register("mock_api_unreachable", &mockDriver{failPing: true})

	router := setupTestRouter(t)
	body := `{"id":"conn1","driver":"mock_api_unreachable","host":"localhost:3306"}`

	req := httptest.NewRequest("POST", "/api/connections", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "failed to ping") {
		t.Errorf("expected ping failure error, got %q", rec.Body.String())
	}
}

func TestPostConnection_DuplicateID(t *testing.T) {
	sql.Register("mock_api_duplicate", &mockDriver{failPing: false})

	router := setupTestRouter(t)
	body := `{"id":"conn1","driver":"mock_api_duplicate","host":"localhost:3306"}`

	req := httptest.NewRequest("POST", "/api/connections", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected first registration to succeed with 201, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest("POST", "/api/connections", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for duplicate id, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "already exists") {
		t.Errorf("expected duplicate id error, got %q", rec.Body.String())
	}
}

func registerSQLiteConnection(t *testing.T, env testEnv, id string, dbPath string) {
	t.Helper()

	connBodyBytes, err := json.Marshal(map[string]string{
		"id":     id,
		"driver": "sqlite",
		"host":   dbPath,
	})
	if err != nil {
		t.Fatalf("failed to marshal connection payload: %v", err)
	}

	req := httptest.NewRequest("POST", "/api/connections", strings.NewReader(string(connBodyBytes)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	env.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("failed to register connection: %d %s", rec.Code, rec.Body.String())
	}

	t.Cleanup(func() {
		if err := env.cm.RemoveConnection(id); err != nil {
			t.Errorf("failed to remove test connection %q: %v", id, err)
		}
	})
}

func TestPostQuery_Success(t *testing.T) {
	env := setupTestEnv(t)
	dbPath := createSeededSQLiteDB(t)
	registerSQLiteConnection(t, env, "query_conn", dbPath)

	queryBody := `{"id":"query_conn","sql":"SELECT id, name FROM items ORDER BY id ASC"}`
	req := httptest.NewRequest("POST", "/api/query", strings.NewReader(queryBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	env.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", ct)
	}

	var results []map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&results); err != nil {
		t.Fatalf("failed to decode query response: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(results))
	}
	if results[0]["name"] != "alpha" {
		t.Errorf("expected first row name 'alpha', got %v", results[0]["name"])
	}
}

func TestPostQuery_InvalidJSON(t *testing.T) {
	router := setupTestRouter(t)

	req := httptest.NewRequest("POST", "/api/query", strings.NewReader("not-json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Invalid JSON structure payload") {
		t.Errorf("expected invalid JSON error, got %q", rec.Body.String())
	}
}

func TestPostQuery_MissingFields(t *testing.T) {
	router := setupTestRouter(t)

	tests := []struct {
		name string
		body string
	}{
		{name: "missing id", body: `{"sql":"SELECT 1"}`},
		{name: "missing sql", body: `{"id":"conn1"}`},
		{name: "empty fields", body: `{"id":"","sql":""}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/query", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("expected status 400, got %d", rec.Code)
			}
			if !strings.Contains(rec.Body.String(), "Missing mandatory") {
				t.Errorf("expected missing fields error, got %q", rec.Body.String())
			}
		})
	}
}

func TestPostQuery_InvalidSQL(t *testing.T) {
	env := setupTestEnv(t)
	dbPath := createSeededSQLiteDB(t)
	registerSQLiteConnection(t, env, "query_conn", dbPath)

	queryBody := `{"id":"query_conn","sql":"SELECT * FROM missing_table"}`
	req := httptest.NewRequest("POST", "/api/query", strings.NewReader(queryBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	env.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "error querying") {
		t.Errorf("expected query error, got %q", rec.Body.String())
	}
}

func TestPostQuery_UnknownConnection(t *testing.T) {
	router := setupTestRouter(t)
	body := `{"id":"missing_conn","sql":"SELECT 1"}`

	req := httptest.NewRequest("POST", "/api/query", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "does not exist in registry") {
		t.Errorf("expected unknown connection error, got %q", rec.Body.String())
	}
}
