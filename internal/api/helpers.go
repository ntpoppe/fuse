package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/ntpoppe/fuse/internal/fuseerr"
)

type errorResponse struct {
	Error string `json:"error"`
}

func decodeJSON(r *http.Request, dst any) error {
	return json.NewDecoder(r.Body).Decode(dst)
}

func writeJSON(w http.ResponseWriter, status int, body any) error {
	w.Header().Set("Content-Type", ContentTypeJSON)
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(body)
}

func writeAPIError(w http.ResponseWriter, status int, message string) {
	_ = writeJSON(w, status, errorResponse{Error: message})
}

func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, fuseerr.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, fuseerr.ErrAlreadyExists):
		status = http.StatusBadRequest
	case errors.Is(err, fuseerr.ErrQueryRowLimit):
		status = http.StatusBadRequest
	}
	writeAPIError(w, status, err.Error())
}
