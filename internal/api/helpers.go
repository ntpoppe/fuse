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

func decodeJSON(w http.ResponseWriter, r *http.Request, maxBytes int64, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	return json.NewDecoder(r.Body).Decode(dst)
}

func decodeJSONError(w http.ResponseWriter, err error) bool {
	var maxErr *http.MaxBytesError
	if errors.As(err, &maxErr) {
		writeAPIError(w, http.StatusRequestEntityTooLarge, "request body too large")
		return true
	}
	return false
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
	case errors.Is(err, fuseerr.ErrReadOnly):
		status = http.StatusBadRequest
	}
	writeAPIError(w, status, err.Error())
}

func writeFederatedError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, fuseerr.ErrNotFound):
		writeAPIError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, fuseerr.ErrQueryRowLimit):
		writeAPIError(w, http.StatusBadRequest, err.Error())
	default:
		status := http.StatusBadRequest
		var legErr fuseerr.LegExecutionError
		if errors.As(err, &legErr) {
			status = http.StatusInternalServerError
		}
		writeAPIError(w, status, err.Error())
	}
}
