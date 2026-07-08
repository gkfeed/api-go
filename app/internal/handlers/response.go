package handlers

import (
	"encoding/json"
	"net/http"
)

const noAuthMessage = "No authentication provided"

func basicAuthUserName(w http.ResponseWriter, r *http.Request) (string, bool) {
	userName, _, ok := r.BasicAuth()
	if ok {
		return userName, true
	}

	if _, err := w.Write([]byte(noAuthMessage)); err != nil {
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
	}
	return "", false
}

func respondJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
