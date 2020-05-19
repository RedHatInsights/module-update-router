package main

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
			input: request{http.MethodGet, "/api/v1/channel?module=insights-core", "", map[string]string{"X-Rh-Identity": base64.StdEncoding.EncodeToString([]byte(`{ "identity": { "account_number": "540155", "type": "User", "internal": { "org_id": "1979710" } } }`))}},
			want:  response{http.StatusOK, `{"url":"/testing"}`},
		},
		{
			desc:  "want /release",
			input: request{http.MethodGet, "/api/v1/channel?module=insights-core", "", map[string]string{"X-Rh-Identity": base64.StdEncoding.EncodeToString([]byte(`{ "identity": { "account_number": "540156", "type": "User", "internal": { "org_id": "1979710" } } }`))}},
			want:  response{http.StatusOK, `{"url":"/release"}`},
		},
	}

	// Bootstrap a server and seed the database
	srv, err := NewServer(":8080", "file::memory:?cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()
	srv.db.Load("insights-core,540155")

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			reader := strings.NewReader(test.input.body)
			req := httptest.NewRequest(test.input.method, test.input.url, reader)
			for k, v := range test.input.headers {
				req.Header.Add(k, v)
			}
			rr := httptest.NewRecorder()
			srv.ServeHTTP(rr, req)

			if rr.Code != test.want.code {
				t.Errorf("%v != %v", rr.Code, test.want.code)
			}

			if rr.Body.String() != test.want.body {
				t.Errorf("%+v != %+v", rr.Body.String(), test.want.body)
			}
		})
	}
}
