package main

import (
	"testing"
)

func TestDBCount(t *testing.T) {
	tests := []struct {
		desc  string
		input struct{ moduleName, accountID string }
		want  int
	}{
		{
			desc:  "",
			input: struct{ moduleName, accountID string }{"modfoo", "1"},
			want:  1,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			db, err := Open("sqlite3", "file::memory:?cache=shared")
			if err != nil {
				t.Fatal(err)
			}
			if err := db.Insert(test.input.moduleName, test.input.accountID); err != nil {
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
