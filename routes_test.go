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
			desc:  "GET /ping - want OK",
			input: request{http.MethodGet, "/ping", "", nil},
			want:  response{http.StatusOK, "OK"},
		},
		{
			desc:  "GET /channel - want /testing",
			input: request{http.MethodGet, "/api/module-update-router/v1/channel?module=insights-core", "", map[string]string{"X-Rh-Identity": base64.StdEncoding.EncodeToString([]byte(`{ "identity": { "account_number": "540155", "type": "User", "internal": { "org_id": "1979710" } } }`))}},
			want:  response{http.StatusOK, `{"url":"/testing"}`},
		},
		{
			desc:  "GET /channel - want /release",
			input: request{http.MethodGet, "/api/module-update-router/v1/channel?module=insights-core", "", map[string]string{"X-Rh-Identity": base64.StdEncoding.EncodeToString([]byte(`{ "identity": { "account_number": "540156", "type": "User", "internal": { "org_id": "1979710" } } }`))}},
			want:  response{http.StatusOK, `{"url":"/release"}`},
		},
		{
			desc:  "POST /event - want CREATED",
			input: request{http.MethodPost, "/api/module-update-router/v1/event", `{"phase": "pre_update", "started_at": "2020-06-19T11:18:03-04:00", "exit": 1, "exception": "OSPermissionError", "ended_at": "2020-06-19T11:19:03-04:00", "machine_id": "60654767-dfba-47af-8bca-cb2d1d01d9a6", "core_version": "3.0.156", "core_path": "/etc/rpm/insights.egg"}`, map[string]string{"X-Rh-Identity": base64.StdEncoding.EncodeToString([]byte(`{ "identity": { "account_number": "540155", "type": "User", "internal": { "org_id": "1979710" } } }`))}},
			want:  response{http.StatusCreated, ""},
		},
		{
			desc:  "POST /event - want CREATED - exception is null",
			input: request{http.MethodPost, "/api/module-update-router/v1/event", `{"phase": "pre_update", "started_at": "2020-06-19T11:18:03-04:00", "exit": 0, "exception": null, "ended_at": "2020-06-19T11:19:03-04:00", "machine_id": "60654767-dfba-47af-8bca-cb2d1d01d9a6", "core_version": "3.0.156", "core_path": "/etc/rpm/insights.egg"}`, map[string]string{"X-Rh-Identity": base64.StdEncoding.EncodeToString([]byte(`{ "identity": { "account_number": "540155", "type": "User", "internal": { "org_id": "1979710" } } }`))}},
			want:  response{http.StatusCreated, ""},
		},
		{
			desc:  "POST /event - want CREATED - exception is omitted",
			input: request{http.MethodPost, "/api/module-update-router/v1/event", `{"phase": "pre_update", "started_at": "2020-06-19T11:18:03-04:00", "exit": 0, "ended_at": "2020-06-19T11:19:03-04:00", "machine_id": "60654767-dfba-47af-8bca-cb2d1d01d9a6", "core_version": "3.0.156", "core_path": "/etc/rpm/insights.egg"}`, map[string]string{"X-Rh-Identity": base64.StdEncoding.EncodeToString([]byte(`{ "identity": { "account_number": "540155", "type": "User", "internal": { "org_id": "1979710" } } }`))}},
			want:  response{http.StatusCreated, ""},
		},
		{
			desc:  "POST /event - want BAD REQUEST - exit is null",
			input: request{http.MethodPost, "/api/module-update-router/v1/event", `{"phase": "pre_update", "started_at": "2020-06-19T11:18:03-04:00", "exit": null, "exception": "OSPermissionError", "ended_at": "2020-06-19T11:19:03-04:00", "machine_id": "60654767-dfba-47af-8bca-cb2d1d01d9a6", "core_version": "3.0.156", "core_path": "/etc/rpm/insights.egg"}`, map[string]string{"X-Rh-Identity": base64.StdEncoding.EncodeToString([]byte(`{ "identity": { "account_number": "540155", "type": "User", "internal": { "org_id": "1979710" } } }`))}},
			want:  response{http.StatusBadRequest, `{"errors":[{"status":"Bad Request","title":"missing required *int field: 'exit'"}]}`},
		},
		{
			desc:  "POST /event - want BAD REQUEST - exit is omitted",
			input: request{http.MethodPost, "/api/module-update-router/v1/event", `{"phase": "pre_update", "started_at": "2020-06-19T11:18:03-04:00", "exception": "OSPermissionError", "ended_at": "2020-06-19T11:19:03-04:00", "machine_id": "60654767-dfba-47af-8bca-cb2d1d01d9a6", "core_version": "3.0.156", "core_path": "/etc/rpm/insights.egg"}`, map[string]string{"X-Rh-Identity": base64.StdEncoding.EncodeToString([]byte(`{ "identity": { "account_number": "540155", "type": "User", "internal": { "org_id": "1979710" } } }`))}},
			want:  response{http.StatusBadRequest, `{"errors":[{"status":"Bad Request","title":"missing required *int field: 'exit'"}]}`},
		},
	}

	// Bootstrap a server and seed the database
	db, err := Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Migrate(false); err != nil {
		t.Fatal(err)
	}
	db.seedData([]byte(`INSERT INTO accounts_modules (account_id, module_name) VALUES ('540155', 'insights-core');`))

	srv, err := NewServer(":8080", []string{"/api/module-update-router/v1"}, db)
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
				t.Errorf("%+v != %+v", got, test.want)
			}
		})
	}
}
