package main

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestRouter(t *testing.T) {
	type request struct {
		method, url, body string
		headers           map[string]string
	}
	type response struct {
		code int
		body string
	}

	tests := []struct {
		desc  string
		input request
		want  response
	}{
		{
			desc:  "ping",
			input: request{http.MethodGet, "/ping", "", nil},
			want:  response{http.StatusOK, "OK"},
		},
		{
			desc:  "want /testing",
			input: request{http.MethodGet, "/api/module-update-router/v1/channel?module=insights-core", "", map[string]string{"X-Rh-Identity": base64.StdEncoding.EncodeToString([]byte(`{ "identity": { "account_number": "540155", "type": "User", "internal": { "org_id": "1979710" } } }`))}},
			want:  response{http.StatusOK, `{"url":"/testing"}`},
		},
		{
			desc:  "want /release",
			input: request{http.MethodGet, "/api/module-update-router/v1/channel?module=insights-core", "", map[string]string{"X-Rh-Identity": base64.StdEncoding.EncodeToString([]byte(`{ "identity": { "account_number": "540156", "type": "User", "internal": { "org_id": "1979710" } } }`))}},
			want:  response{http.StatusOK, `{"url":"/release"}`},
		},
	}

	// Bootstrap a server and seed the database
	db, err := Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	db.Load("insights-core,540155")

	srv, err := NewServer(":8080", "/api/module-update-router/v1", db)
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			reader := strings.NewReader(test.input.body)
			req := httptest.NewRequest(test.input.method, test.input.url, reader)
			for k, v := range test.input.headers {
				req.Header.Add(k, v)
			}
			rr := httptest.NewRecorder()
			srv.ServeHTTP(rr, req)
			got := response{rr.Code, rr.Body.String()}

			if !cmp.Equal(got, test.want, cmp.AllowUnexported(response{})) {
				t.Errorf("%#v != %#v", got, test.want)
			}
		})
	}
}
