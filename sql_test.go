package main

import (
	"database/sql"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestNewNullString(t *testing.T) {
	tests := []struct {
		description string
		input       *string
		want        sql.NullString
	}{
		{
			description: "nil pointer",
			input:       nil,
			want:        sql.NullString{},
		},
		{
			description: "zero-value",
			input:       func() *string { s := ""; return &s }(),
			want:        sql.NullString{},
		},
		{
			description: "full string",
			input:       func() *string { s := "hello world"; return &s }(),
			want:        sql.NullString{Valid: true, String: "hello world"},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			got := NewNullString(test.input)

			if !cmp.Equal(got, test.want) {
				t.Errorf("%#v != %#v", got, test.want)
			}
		})
	}
}
