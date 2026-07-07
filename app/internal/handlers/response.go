package handlers

import "net/http"

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
