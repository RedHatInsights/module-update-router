package main

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestDBCount(t *testing.T) {
	tests := []struct {
		desc  string
		input struct{ query, moduleName, accountID string }
		want  int
	}{
		{
			desc:  "",
			input: struct{ query, moduleName, accountID string }{`INSERT INTO accounts_modules (account_id, module_name) VALUES ('%s', '%s');`, "modfoo", "1"},
			want:  1,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			db, err := Open("sqlite3", "file::memory:?cache=shared")
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			if err := db.Migrate(false); err != nil {
				t.Fatal(err)
			}
			if err := db.seedData([]byte(fmt.Sprintf(test.input.query, test.input.accountID, test.input.moduleName))); err != nil {
				t.Fatal(err)
			}

			got, err := db.Count(test.input.moduleName, test.input.accountID)
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Errorf("%+v != %+v", got, test.want)
			}
		})
	}
}

func TestDBInsertEvents(t *testing.T) {
	type record struct {
		phase       string
		startedAt   time.Time
		exit        int
		exception   sql.NullString
		endedAt     time.Time
		machineID   string
		coreVersion string
		corePath    string
	}

	tests := []struct {
		desc  string
		input record
	}{
		{
			desc: "basic event",
			input: record{
				phase:       "pre_update",
				startedAt:   time.Now(),
				exit:        1,
				exception:   sql.NullString{String: "OSPermissionError", Valid: true},
				endedAt:     time.Now().Add(164),
				machineID:   "fd475f2c-544f-4dd7-b53f-209df3290504",
				coreVersion: "3.0.156",
				corePath:    "/etc/rpm/insights.egg",
			},
		},
		{
			desc: "null exception",
			input: record{
				phase:       "pre_update",
				startedAt:   time.Now(),
				exit:        0,
				exception:   sql.NullString{String: "", Valid: false},
				endedAt:     time.Now().Add(164),
				machineID:   "fd475f2c-544f-4dd7-b53f-209df3290504",
				coreVersion: "3.0.156",
				corePath:    "/etc/rpm/insights.egg",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			db, err := Open("sqlite3", "file::memory:?cache=shared")
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			if err := db.Migrate(false); err != nil {
				t.Fatal(err)
			}

			if err := db.InsertEvents(test.input.phase, test.input.startedAt, test.input.exit, test.input.exception, test.input.endedAt, test.input.machineID, test.input.coreVersion, test.input.corePath); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestDBGetEvents(t *testing.T) {
	tests := []struct {
		desc  string
		input struct {
			query  string
			limit  int
			offset int
		}
		want []map[string]interface{}
	}{
		{
			desc: "limit 1",
			input: struct {
				query  string
				limit  int
				offset int
			}{
				query:  `INSERT INTO events (event_id, phase, started_at, exit, exception, ended_at, machine_id, core_version, core_path) VALUES ("af3b8e13-6b65-45d8-8310-a45e0821bd62", "pre_update", "2020-07-15T17:16:55+00:00", 1, "OSError", "2020-07-15T17:17:37+00:00", "a9ab0a44-1241-43ae-9c02-1850acf0c36c", "3.0.156", "/etc/insights-client/rpm.egg");`,
				limit:  1,
				offset: 0,
			},
			want: []map[string]interface{}{
				{
					"event_id":     "af3b8e13-6b65-45d8-8310-a45e0821bd62",
					"phase":        "pre_update",
					"started_at":   time.Date(2020, time.July, 15, 17, 16, 55, 0, time.UTC),
					"exit":         1,
					"exception":    "OSError",
					"ended_at":     time.Date(2020, time.July, 15, 17, 17, 37, 0, time.UTC),
					"machine_id":   "a9ab0a44-1241-43ae-9c02-1850acf0c36c",
					"core_version": "3.0.156",
					"core_path":    "/etc/insights-client/rpm.egg",
				},
			},
		},
		{
			desc: "limit 0",
			input: struct {
				query  string
				limit  int
				offset int
			}{
				query:  `INSERT INTO events (event_id, phase, started_at, exit, exception, ended_at, machine_id, core_version, core_path) VALUES ("af3b8e13-6b65-45d8-8310-a45e0821bd62", "pre_update", "2020-07-15T17:16:55+00:00", 1, NULL, "2020-07-15T17:17:37+00:00", "a9ab0a44-1241-43ae-9c02-1850acf0c36c", "3.0.156", "/etc/insights-client/rpm.egg");`,
				limit:  0,
				offset: 0,
			},
			want: []map[string]interface{}{},
		},
		{
			desc: "limit -1",
			input: struct {
				query  string
				limit  int
				offset int
			}{
				query:  `INSERT INTO events (event_id, phase, started_at, exit, exception, ended_at, machine_id, core_version, core_path) VALUES ("af3b8e13-6b65-45d8-8310-a45e0821bd62", "pre_update", "2020-07-15T17:16:55+00:00", 1, NULL, "2020-07-15T17:17:37+00:00", "a9ab0a44-1241-43ae-9c02-1850acf0c36c", "3.0.156", "/etc/insights-client/rpm.egg");`,
				limit:  -1,
				offset: 0,
			},
			want: []map[string]interface{}{
				{
					"event_id":     "af3b8e13-6b65-45d8-8310-a45e0821bd62",
					"phase":        "pre_update",
					"started_at":   time.Date(2020, time.July, 15, 17, 16, 55, 0, time.UTC),
					"exit":         1,
					"ended_at":     time.Date(2020, time.July, 15, 17, 17, 37, 0, time.UTC),
					"machine_id":   "a9ab0a44-1241-43ae-9c02-1850acf0c36c",
					"core_version": "3.0.156",
					"core_path":    "/etc/insights-client/rpm.egg",
				},
			},
		},
		{
			desc: "NULL core_path",
			input: struct {
				query  string
				limit  int
				offset int
			}{
				query:  `INSERT INTO events (event_id, phase, started_at, exit, exception, ended_at, machine_id, core_version, core_path) VALUES ("af3b8e13-6b65-45d8-8310-a45e0821bd62", "pre_update", "2020-07-15T17:16:55+00:00", 1, NULL, "2020-07-15T17:17:37+00:00", "a9ab0a44-1241-43ae-9c02-1850acf0c36c", "3.0.156", NULL);`,
				limit:  1,
				offset: 0,
			},
			want: []map[string]interface{}{
				{
					"event_id":     "af3b8e13-6b65-45d8-8310-a45e0821bd62",
					"phase":        "pre_update",
					"started_at":   time.Date(2020, time.July, 15, 17, 16, 55, 0, time.UTC),
					"exit":         1,
					"ended_at":     time.Date(2020, time.July, 15, 17, 17, 37, 0, time.UTC),
					"machine_id":   "a9ab0a44-1241-43ae-9c02-1850acf0c36c",
					"core_version": "3.0.156",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			db, err := Open("sqlite3", "file::memory:?cache=shared")
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			if err := db.Migrate(false); err != nil {
				t.Fatal(err)
			}
			if err := db.seedData([]byte(test.input.query)); err != nil {
				t.Fatal(err)
			}

			got, err := db.GetEvents(test.input.limit, test.input.offset)
			if err != nil {
				t.Fatal(err)
			}
			if !cmp.Equal(got, test.want) {
				t.Errorf("%v", cmp.Diff(got, test.want))
			}
		})
	}
}

func TestDeleteEvents(t *testing.T) {
	tests := []struct {
		description string
		input       struct {
			seed []string
			date time.Time
		}
		want      int64
		wantError error
	}{
		{
			description: "3 events, 3 older",
			input: struct {
				seed []string
				date time.Time
			}{
				seed: []string{
					`INSERT INTO events (event_id, phase, started_at, exit, exception, ended_at, machine_id, core_version, core_path) VALUES ("a775eb95-baa0-48ef-80a5-438adfefca85", "pre_update", "2020-07-15T17:16:55+00:00", 1, NULL, "2020-07-15T17:17:37+00:00", "a9ab0a44-1241-43ae-9c02-1850acf0c36c", "3.0.156", NULL);`,
					`INSERT INTO events (event_id, phase, started_at, exit, exception, ended_at, machine_id, core_version, core_path) VALUES ("6d7d9b1b-60bf-4523-b5df-db6c9a2e3e4b", "pre_update", "2020-07-15T17:18:55+00:00", 1, NULL, "2020-07-15T17:19:37+00:00", "a9ab0a44-1241-43ae-9c02-1850acf0c36c", "3.0.156", NULL);`,
					`INSERT INTO events (event_id, phase, started_at, exit, exception, ended_at, machine_id, core_version, core_path) VALUES ("7f4cfe4b-415a-478e-98d7-ec232a8cf181", "pre_update", "2020-07-15T17:20:55+00:00", 1, NULL, "2020-07-15T17:21:37+00:00", "a9ab0a44-1241-43ae-9c02-1850acf0c36c", "3.0.156", NULL);`,
				},
				date: time.Date(2020, time.July, 15, 17, 21, 00, 00, time.UTC),
			},
			want: 3,
		},
		{
			description: "3 events, 1 older",
			input: struct {
				seed []string
				date time.Time
			}{
				seed: []string{
					`INSERT INTO events (event_id, phase, started_at, exit, exception, ended_at, machine_id, core_version, core_path) VALUES ("a775eb95-baa0-48ef-80a5-438adfefca85", "pre_update", "2020-07-15T17:16:55+00:00", 1, NULL, "2020-07-15T17:17:37+00:00", "a9ab0a44-1241-43ae-9c02-1850acf0c36c", "3.0.156", NULL);`,
					`INSERT INTO events (event_id, phase, started_at, exit, exception, ended_at, machine_id, core_version, core_path) VALUES ("6d7d9b1b-60bf-4523-b5df-db6c9a2e3e4b", "pre_update", "2020-07-15T17:18:55+00:00", 1, NULL, "2020-07-15T17:19:37+00:00", "a9ab0a44-1241-43ae-9c02-1850acf0c36c", "3.0.156", NULL);`,
					`INSERT INTO events (event_id, phase, started_at, exit, exception, ended_at, machine_id, core_version, core_path) VALUES ("7f4cfe4b-415a-478e-98d7-ec232a8cf181", "pre_update", "2020-07-15T17:20:55+00:00", 1, NULL, "2020-07-15T17:21:37+00:00", "a9ab0a44-1241-43ae-9c02-1850acf0c36c", "3.0.156", NULL);`,
				},
				date: time.Date(2020, time.July, 15, 17, 17, 00, 00, time.UTC),
			},
			want: 1,
		},
		{
			description: "3 events, 0 older",
			input: struct {
				seed []string
				date time.Time
			}{
				seed: []string{
					`INSERT INTO events (event_id, phase, started_at, exit, exception, ended_at, machine_id, core_version, core_path) VALUES ("a775eb95-baa0-48ef-80a5-438adfefca85", "pre_update", "2020-07-15T17:16:55+00:00", 1, NULL, "2020-07-15T17:17:37+00:00", "a9ab0a44-1241-43ae-9c02-1850acf0c36c", "3.0.156", NULL);`,
					`INSERT INTO events (event_id, phase, started_at, exit, exception, ended_at, machine_id, core_version, core_path) VALUES ("6d7d9b1b-60bf-4523-b5df-db6c9a2e3e4b", "pre_update", "2020-07-15T17:18:55+00:00", 1, NULL, "2020-07-15T17:19:37+00:00", "a9ab0a44-1241-43ae-9c02-1850acf0c36c", "3.0.156", NULL);`,
					`INSERT INTO events (event_id, phase, started_at, exit, exception, ended_at, machine_id, core_version, core_path) VALUES ("7f4cfe4b-415a-478e-98d7-ec232a8cf181", "pre_update", "2020-07-15T17:20:55+00:00", 1, NULL, "2020-07-15T17:21:37+00:00", "a9ab0a44-1241-43ae-9c02-1850acf0c36c", "3.0.156", NULL);`,
				},
				date: time.Date(2020, time.July, 15, 17, 15, 00, 00, time.UTC),
			},
			want: 0,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			db, err := Open("sqlite3", "file::memory:?cache=shared")
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			if err := db.Migrate(false); err != nil {
				t.Fatal(err)
			}
			for _, q := range test.input.seed {
				if err := db.seedData([]byte(q)); err != nil {
					t.Fatal(err)
				}
			}

			got, err := db.DeleteEvents(test.input.date)

			if test.wantError != nil {
				if !cmp.Equal(err, test.wantError, cmpopts.EquateErrors()) {
					t.Errorf("%#v != %#v", err, test.wantError)
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				if !cmp.Equal(got, test.want) {
					t.Errorf("%#v != %#v", got, test.want)
				}
			}
		})
	}
}
