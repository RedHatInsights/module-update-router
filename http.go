package main

import (
	"bytes"
	"fmt"
	"net/http"
)

// responseRecorder records status code and body from an http.ResponseWriter.
type responseRecorder struct {
	http.ResponseWriter
	Code int
	Body *bytes.Buffer
}

// newResponseRecorder creates a new responseRecorder, wrapping the given
// http.ResponseWriter w.
func newResponseRecorder(w http.ResponseWriter) *responseRecorder {
	return &responseRecorder{
		ResponseWriter: w,
		Code:           0,
		Body:           new(bytes.Buffer),
	}
}

func (r *responseRecorder) WriteHeader(code int) {
	r.Code = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *responseRecorder) Write(buf []byte) (int, error) {
	if r.Code == 0 {
		r.WriteHeader(http.StatusOK)
	}
	if r.Body != nil {
		r.Body.Write(buf)
	}
	return r.ResponseWriter.Write(buf)
}

func (r *responseRecorder) String() string {
	return fmt.Sprintf("%v %v", r.Code, r.Body.String())
}
