package main

import (
	"io"
	"net/http"

	"github.com/Khaz713/Tag_and_Seek/internal/pubsub"
)

func decodeRequestGob[T any](w http.ResponseWriter, r *http.Request) (T, bool) {
	if r.Header.Get("Game-Client") != "TagAndSeek" {
		http.Error(w, "Unauthorized Client", http.StatusUnauthorized)
		var empty T
		return empty, false
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		var empty T
		return empty, false
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		var empty T
		return empty, false
	}
	defer r.Body.Close()

	var req T
	req, err = pubsub.DecodeGob[T](bodyBytes)
	if err != nil {
		http.Error(w, "Failed to decode body", http.StatusBadRequest)
		var empty T
		return empty, false
	}
	return req, true
}

func writeResponseGob[T any](w http.ResponseWriter, statusCode int, data T) {
	resp, err := pubsub.EncodeGob[T](data)
	if err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.WriteHeader(statusCode)
	_, err = w.Write(resp)
	if err != nil {
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
		return
	}
}
