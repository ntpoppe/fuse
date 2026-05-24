package api

import (
	"encoding/json"
	"net/http"

	connectionmanager "github.com/ntpoppe/fuse/internal/connection_manager"
)

const jsonContentType = "application/json"

func NewRouter(cm *connectionmanager.ConnectionManager) *http.ServeMux {
	router := http.ServeMux{}
	router.HandleFunc("GET /health", getHealth)

	return &router
}

func getHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", jsonContentType)
	w.WriteHeader(http.StatusOK)

	response := map[string]string{"status": "ok"}
	err := json.NewEncoder(w).Encode(response)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func postTestQuery(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", jsonContentType)
	w.WriteHeader(http.StatusOK)

}
