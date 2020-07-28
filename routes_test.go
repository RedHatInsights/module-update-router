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
			desc:  "POST /event - want CREATED - date format Z",
			input: request{http.MethodPost, "/api/module-update-router/v1/event", `{"phase": "pre_update", "started_at": "2020-06-19T11:18:03Z", "exit": 0, "ended_at": "2020-06-19T11:19:03Z", "machine_id": "60654767-dfba-47af-8bca-cb2d1d01d9a6", "core_version": "3.0.156", "core_path": "/etc/rpm/insights.egg"}`, map[string]string{"X-Rh-Identity": base64.StdEncoding.EncodeToString([]byte(`{ "identity": { "account_number": "540155", "type": "User", "internal": { "org_id": "1979710" } } }`))}},
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
		{
			desc: "GET /event - limit 1",
			input: request{
				method: http.MethodGet,
				url:    "/api/module-update-router/v1/event?limit=1",
				body:   ``,
				headers: map[string]string{
					"X-Rh-Identity": base64.StdEncoding.EncodeToString([]byte(`{ "identity": { "account_number": "540155", "type": "User", "internal": { "org_id": "1979710" } } }`)),
				},
			},
			want: response{
				code: http.StatusOK,
				body: `[{"core_path":"/etc/insights-client/rpm.egg","core_version":"3.0.156","ended_at":"2020-07-15T17:17:37Z","event_id":"af3b8e13-6b65-45d8-8310-a45e0821bd62","exception":{"String":"","Valid":false},"exit":1,"machine_id":"a9ab0a44-1241-43ae-9c02-1850acf0c36c","phase":"pre_update","started_at":"2020-06-19T11:18:03Z"}]`,
			},
		},
		{
			desc: "GET /event - omit limit, omit offset",
			input: request{
				method: http.MethodGet,
				url:    "/api/module-update-router/v1/event",
				body:   ``,
				headers: map[string]string{
					"X-Rh-Identity": base64.StdEncoding.EncodeToString([]byte(`{ "identity": { "account_number": "540155", "type": "User", "internal": { "org_id": "1979710" } } }`)),
				},
			},
			want: response{
				code: http.StatusOK,
				body: `[{"core_path":"/etc/insights-client/rpm.egg","core_version":"3.0.156","ended_at":"2020-07-15T17:17:37Z","event_id":"af3b8e13-6b65-45d8-8310-a45e0821bd62","exception":{"String":"","Valid":false},"exit":1,"machine_id":"a9ab0a44-1241-43ae-9c02-1850acf0c36c","phase":"pre_update","started_at":"2020-06-19T11:18:03Z"},{"core_path":"/var/lib/insights/latest.egg","core_version":"3.0.156","ended_at":"2020-07-21T13:02:31Z","event_id":"89d9352c-0f53-49c0-9f7c-27a9ee3e2dff","exception":{"String":"OSError","Valid":true},"exit":1,"machine_id":"21f3e7da-6e33-41dd-b25f-0eab2242ae27","phase":"pre_update","started_at":"2020-07-21T13:01:04Z"}]`,
			},
		},
		{
			desc: "GET /event - limit 1, offset 1",
			input: request{
				method: http.MethodGet,
				url:    "/api/module-update-router/v1/event?offset=1&limit=1",
				body:   ``,
				headers: map[string]string{
					"X-Rh-Identity": base64.StdEncoding.EncodeToString([]byte(`{ "identity": { "account_number": "540155", "type": "User", "internal": { "org_id": "1979710" } } }`)),
				},
			},
			want: response{
				code: http.StatusOK,
				body: `[{"core_path":"/var/lib/insights/latest.egg","core_version":"3.0.156","ended_at":"2020-07-21T13:02:31Z","event_id":"89d9352c-0f53-49c0-9f7c-27a9ee3e2dff","exception":{"String":"OSError","Valid":true},"exit":1,"machine_id":"21f3e7da-6e33-41dd-b25f-0eab2242ae27","phase":"pre_update","started_at":"2020-07-21T13:01:04Z"}]`,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			// Bootstrap a server and seed the database
			db, err := Open("sqlite3", "file::memory:?cache=shared")
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			if err := db.Migrate(false); err != nil {
				t.Fatal(err)
			}
			db.seedData([]byte(`INSERT INTO accounts_modules (account_id, module_name) VALUES ('540155', 'insights-core');`))
			db.seedData([]byte(`INSERT INTO accounts_events (account_id) VALUES ('540155');`))
			db.seedData([]byte(`INSERT INTO events (event_id, phase, started_at, exit, exception, ended_at, machine_id, core_version, core_path)
			VALUES ("af3b8e13-6b65-45d8-8310-a45e0821bd62", "pre_update", "2020-06-19T11:18:03Z", 1, NULL, "2020-07-15T17:17:37Z", "a9ab0a44-1241-43ae-9c02-1850acf0c36c", "3.0.156", "/etc/insights-client/rpm.egg");`))
			db.seedData([]byte(`INSERT INTO events (event_id, phase, started_at, exit, exception, ended_at, machine_id, core_version, core_path)
			VALUES ("89d9352c-0f53-49c0-9f7c-27a9ee3e2dff", "pre_update", "2020-07-21T13:01:04Z", 1, "OSError", "2020-07-21T13:02:31Z", "21f3e7da-6e33-41dd-b25f-0eab2242ae27", "3.0.156", "/var/lib/insights/latest.egg");`))

			srv, err := NewServer(":8080", []string{"/api/module-update-router/v1"}, db, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer srv.Close()

			reader := strings.NewReader(test.input.body)
			req := httptest.NewRequest(test.input.method, test.input.url, reader)
			for k, v := range test.input.headers {
				req.Header.Add(k, v)
			}
			rr := httptest.NewRecorder()
			srv.ServeHTTP(rr, req)
			got := response{rr.Code, rr.Body.String()}

			if !cmp.Equal(got, test.want, cmp.AllowUnexported(response{})) {
				t.Errorf("\ngot:  %+v\nwant: %+v", got, test.want)
			}
		})
	}
}
