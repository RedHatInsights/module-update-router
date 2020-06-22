package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"
)

// formatError converts a basic HTTP status code and message into a JSON API
// error object, serializes it to JSON and writes it to w.
func formatJSONError(w http.ResponseWriter, code int, msg string) {
	type response struct {
		Errors []struct {
			Status string `json:"status"`
			Title  string `json:"title"`
		} `json:"errors"`
	}

	r := response{
		Errors: []struct {
			Status string "json:\"status\""
			Title  string "json:\"title\""
		}{},
	}

	r.Errors = append(r.Errors, struct {
		Status string "json:\"status\""
		Title  string "json:\"title\""
	}{
		Status: http.StatusText(code),
		Title:  msg,
	})

	data, err := json.Marshal(&r)
	if err != nil {
		log.Error(err)
		writeError(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	log.Error(r)
	writeError(w, string(data), code)
}

// writeError replies to the request with the specified error message and HTTP
// code.
func writeError(w http.ResponseWriter, error string, code int) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)
	fmt.Fprint(w, error)
}
