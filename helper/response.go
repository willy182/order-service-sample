package helper

import (
	"encoding/json"
	"net/http"
)

// WriteJSON mengirim response JSON dengan status code
func WriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// WriteErrorJSON mengirim response JSON untuk error
func WriteErrorJSON(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, map[string]string{"error": message})
}
