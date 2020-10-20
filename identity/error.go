package identity

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// ErrMissingIdentityValue occurs when a request context is missing a value for
// the Identity.
var ErrMissingIdentityValue = fmt.Errorf("identity: nil value found in request context")

// TypeCastError represents a failed attempt at casting a type.
type TypeCastError struct {
	from, to interface{}
}

func (e TypeCastError) Error() string {
	return fmt.Sprintf("identity: cannot cast %T as %T", e.from, e.to)
}

// formatError converts a basic HTTP status code and message into a JSON API
// error object, serializes it to JSON and writes it to w.
func formatJSONError(w http.ResponseWriter, code int, msg string) {
	r := map[string]interface{}{
		"errors": []map[string]interface{}{
			{
				"status": http.StatusText(code),
				"title":  msg,
			},
		},
	}

	data, err := json.Marshal(&r)
	if err != nil {
		writeError(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

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
