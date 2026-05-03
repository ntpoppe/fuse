package api

import (
	"encoding/json"
	"net/http"
)

const jsonContentType = "application/json"

func NewRouter() *http.ServeMux {
	router := http.ServeMux{}
	router.HandleFunc("GET /health", getHealth)

	return &router
}

func getHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", jsonContentType)
	w.WriteHeader(http.StatusOK)

	response := map[string]string{"key": "value"}
	err := json.NewEncoder(w).Encode(response)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
