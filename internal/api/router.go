package api

import (
	"net/http"

	connectionmanager "github.com/ntpoppe/fuse/internal/connection_manager"
	"github.com/ntpoppe/fuse/internal/executor"
	"github.com/ntpoppe/fuse/internal/storage"
)

type Handler struct {
	cm      *connectionmanager.ConnectionManager
	store   *storage.Store
	exec    *executor.Executor
	fedExec *executor.FederatedExecutor
}

type connectionPayload struct {
	ID     string `json:"id"`
	Driver string `json:"driver"`
	Host   string `json:"host"`
}

type connectionResponse struct {
	ID     string `json:"id"`
	Driver string `json:"driver"`
	Host   string `json:"host"`
}

type queryPayload struct {
	ID  string `json:"id"`
	SQL string `json:"sql"`
}

type federatedQueryPayload struct {
	SQL string `json:"sql"`
}

func NewRouter(
	cm *connectionmanager.ConnectionManager,
	store *storage.Store,
	exec *executor.Executor,
	fedExec *executor.FederatedExecutor,
) *http.ServeMux {
	router := http.ServeMux{}

	h := &Handler{
		cm:      cm,
		store:   store,
		exec:    exec,
		fedExec: fedExec,
	}

	router.HandleFunc("GET "+PathHealth, h.GetHealth)
	router.HandleFunc("GET "+PathConnections, h.GetConnections)
	router.HandleFunc("POST "+PathConnections, h.PostConnection)
	router.HandleFunc("DELETE "+PathConnectionByID, h.DeleteConnection)
	router.HandleFunc("POST "+PathQuery, h.PostQuery)
	router.HandleFunc("POST "+PathFederatedQuery, h.PostFederatedQuery)

	return &router
}

func (h *Handler) GetHealth(w http.ResponseWriter, r *http.Request) {
	if err := writeJSON(w, http.StatusOK, map[string]string{fieldStatus: healthStatusOK}); err != nil {
		writeAPIError(w, http.StatusInternalServerError, err.Error())
	}
}

func (h *Handler) GetConnections(w http.ResponseWriter, r *http.Request) {
	connections, err := h.store.GetAllConnections(r.Context())
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := make([]connectionResponse, 0, len(connections))
	for _, conn := range connections {
		response = append(response, connectionResponse{
			ID:     conn.ID,
			Driver: conn.Driver,
			Host:   conn.Host,
		})
	}

	if err := writeJSON(w, http.StatusOK, response); err != nil {
		writeAPIError(w, http.StatusInternalServerError, err.Error())
	}
}

func (h *Handler) PostConnection(w http.ResponseWriter, r *http.Request) {
	var payload connectionPayload
	if err := decodeJSON(r, &payload); err != nil {
		writeAPIError(w, http.StatusBadRequest, errInvalidJSON)
		return
	}

	if payload.ID == "" || payload.Driver == "" || payload.Host == "" {
		writeAPIError(w, http.StatusBadRequest, errMissingConnFields)
		return
	}

	if err := h.cm.RegisterConnection(payload.ID, payload.Driver, payload.Host); err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}

	connRecord := storage.SavedConnection{
		ID:     payload.ID,
		Driver: payload.Driver,
		Host:   payload.Host,
	}
	if err := h.store.SaveConnection(r.Context(), connRecord); err != nil {
		_ = h.cm.RemoveConnection(payload.ID)
		writeAPIError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *Handler) DeleteConnection(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeAPIError(w, http.StatusBadRequest, errMissingConnectionID)
		return
	}

	saved, _, err := h.store.GetConnection(r.Context(), id)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := h.cm.RemoveConnection(id); err != nil {
		writeError(w, err)
		return
	}

	if err := h.store.RemoveConnection(r.Context(), id); err != nil {
		if saved.ID != "" {
			_ = h.cm.RegisterConnection(saved.ID, saved.Driver, saved.Host)
		}
		writeAPIError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) PostQuery(w http.ResponseWriter, r *http.Request) {
	var payload queryPayload
	if err := decodeJSON(r, &payload); err != nil {
		writeAPIError(w, http.StatusBadRequest, errInvalidJSON)
		return
	}

	if payload.ID == "" || payload.SQL == "" {
		writeAPIError(w, http.StatusBadRequest, errMissingQueryFields)
		return
	}

	results, err := h.exec.ExecuteQuery(r.Context(), payload.ID, payload.SQL)
	if err != nil {
		writeError(w, err)
		return
	}

	if err := writeJSON(w, http.StatusOK, results); err != nil {
		writeAPIError(w, http.StatusInternalServerError, err.Error())
	}
}

func (h *Handler) PostFederatedQuery(w http.ResponseWriter, r *http.Request) {
	var payload federatedQueryPayload
	if err := decodeJSON(r, &payload); err != nil {
		writeAPIError(w, http.StatusBadRequest, errInvalidJSON)
		return
	}

	if payload.SQL == "" {
		writeAPIError(w, http.StatusBadRequest, errMissingFederatedSQL)
		return
	}

	results, err := h.fedExec.ExecuteFederatedQuery(r.Context(), payload.SQL)
	if err != nil {
		writeFederatedError(w, err)
		return
	}

	if err := writeJSON(w, http.StatusOK, results); err != nil {
		writeAPIError(w, http.StatusInternalServerError, err.Error())
	}
}
