package api

import (
	"encoding/json"
	"net/http"

	connectionmanager "github.com/ntpoppe/fuse/internal/connection_manager"
	"github.com/ntpoppe/fuse/internal/executor"
	"github.com/ntpoppe/fuse/internal/storage"
)

const jsonContentType = "application/json"

type Handler struct {
	cm    *connectionmanager.ConnectionManager
	store *storage.Store
	exec  *executor.Executor
}

type connectionPayload struct {
	ID     string `json:"id"`
	Driver string `json:"driver"`
	Host   string `json:"host"`
}

type queryPayload struct {
	ID  string `json:"id"`
	SQL string `json:"sql"`
}

func NewRouter(
	cm *connectionmanager.ConnectionManager,
	store *storage.Store,
	exec *executor.Executor,
) *http.ServeMux {
	router := http.ServeMux{}

	h := &Handler{
		cm:    cm,
		store: store,
		exec:  exec,
	}

	router.HandleFunc("GET /health", h.GetHealth)
	router.HandleFunc("POST /api/connections", h.PostConnections)
	router.HandleFunc("POST /api/query", h.PostQuery)

	return &router
}

func (h *Handler) GetHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", jsonContentType)
	w.WriteHeader(http.StatusOK)

	response := map[string]string{"status": "ok"}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *Handler) PostConnections(w http.ResponseWriter, r *http.Request) {
	var payload connectionPayload

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid JSON structure payload", http.StatusBadRequest)
		return
	}

	if err := h.cm.RegisterConnection(payload.ID, payload.Driver, payload.Host); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	connRecord := storage.SavedConnection{
		ID:     payload.ID,
		Driver: payload.Driver,
		Host:   payload.Host,
	}
	if err := h.store.SaveConnection(r.Context(), connRecord); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *Handler) PostQuery(w http.ResponseWriter, r *http.Request) {
	var payload queryPayload

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid JSON structure payload", http.StatusBadRequest)
		return
	}

	if payload.ID == "" || payload.SQL == "" {
		http.Error(w, "Missing mandatory 'id' or 'sql' body parameters", http.StatusBadRequest)
		return
	}

	results, err := h.exec.ExecuteQuery(r.Context(), payload.ID, payload.SQL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", jsonContentType)
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(results); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
