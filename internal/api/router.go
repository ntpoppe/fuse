package api

import (
	"net/http"

	connectionmanager "github.com/ntpoppe/fuse/internal/connection_manager"
	"github.com/ntpoppe/fuse/internal/executor"
	"github.com/ntpoppe/fuse/internal/storage"
)

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

	router.HandleFunc("GET "+PathHealth, h.GetHealth)
	router.HandleFunc("POST "+PathConnections, h.PostConnection)
	router.HandleFunc("DELETE "+PathConnectionByID, h.DeleteConnection)
	router.HandleFunc("POST "+PathQuery, h.PostQuery)

	return &router
}

func (h *Handler) GetHealth(w http.ResponseWriter, r *http.Request) {
	if err := writeJSON(w, http.StatusOK, map[string]string{fieldStatus: healthStatusOK}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *Handler) PostConnection(w http.ResponseWriter, r *http.Request) {
	var payload connectionPayload
	if err := decodeJSON(r, &payload); err != nil {
		http.Error(w, errInvalidJSON, http.StatusBadRequest)
		return
	}

	if payload.ID == "" || payload.Driver == "" || payload.Host == "" {
		http.Error(w, errMissingConnFields, http.StatusBadRequest)
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
		_ = h.cm.RemoveConnection(payload.ID)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *Handler) DeleteConnection(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, errMissingConnectionID, http.StatusBadRequest)
		return
	}

	if err := h.cm.RemoveConnection(id); err != nil {
		writeError(w, err)
		return
	}

	if err := h.store.RemoveConnection(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) PostQuery(w http.ResponseWriter, r *http.Request) {
	var payload queryPayload
	if err := decodeJSON(r, &payload); err != nil {
		http.Error(w, errInvalidJSON, http.StatusBadRequest)
		return
	}

	if payload.ID == "" || payload.SQL == "" {
		http.Error(w, errMissingQueryFields, http.StatusBadRequest)
		return
	}

	results, err := h.exec.ExecuteQuery(r.Context(), payload.ID, payload.SQL)
	if err != nil {
		writeError(w, err)
		return
	}

	if err := writeJSON(w, http.StatusOK, results); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
