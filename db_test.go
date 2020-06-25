package main

import (
	"database/sql"
	"fmt"
	"testing"
	"time"
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
	tests := []struct {
		desc  string
		input struct {
			phase       string
			startedAt   time.Time
			exit        int
			exception   sql.NullString
			endedAt     time.Time
			machineID   string
			coreVersion string
		}
	}{
		{
			desc: "basic event",
			input: struct {
				phase       string
				startedAt   time.Time
				exit        int
				exception   sql.NullString
				endedAt     time.Time
				machineID   string
				coreVersion string
			}{
				phase:       "pre_update",
				startedAt:   time.Now(),
				exit:        1,
				exception:   sql.NullString{String: "OSPermissionError", Valid: true},
				endedAt:     time.Now().Add(164),
				machineID:   "fd475f2c-544f-4dd7-b53f-209df3290504",
				coreVersion: "3.0.156",
			},
		},
		{
			desc: "null exception",
			input: struct {
				phase       string
				startedAt   time.Time
				exit        int
				exception   sql.NullString
				endedAt     time.Time
				machineID   string
				coreVersion string
			}{
				phase:       "pre_update",
				startedAt:   time.Now(),
				exit:        0,
				exception:   sql.NullString{String: "", Valid: false},
				endedAt:     time.Now().Add(164),
				machineID:   "fd475f2c-544f-4dd7-b53f-209df3290504",
				coreVersion: "3.0.156",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			db, err := Open("sqlite3", "file::memory:?cache=shared")
			if err != nil {
				t.Fatal(err)
			}
			if err := db.Migrate(false); err != nil {
				t.Fatal(err)
			}

			if err := db.InsertEvents(test.input.phase, test.input.startedAt, test.input.exit, test.input.exception, test.input.endedAt, test.input.machineID, test.input.coreVersion); err != nil {
				t.Error(err)
			}
		})
	}
}
