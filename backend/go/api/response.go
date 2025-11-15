// backend/go/api/response.go
package api

import (
	"encoding/json"
	"net/http"
)

// JSON helper: use in handlers instead of directly encoding
func JSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// best-effort: write plain text if encoding fails
		http.Error(w, "encode error", http.StatusInternalServerError)
	}
}

// ErrorJSON: structured error payload
func ErrorJSON(w http.ResponseWriter, status int, msg string) {
	JSON(w, status, map[string]string{"error": msg})
}
