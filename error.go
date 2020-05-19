package main

import (
	"encoding/json"
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
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	log.Error(r)
	http.Error(w, string(data), code)
}
