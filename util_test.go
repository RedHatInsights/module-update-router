package main

import (
	"os"
	"testing"
)

func TestDefaultEnv(t *testing.T) {
	tests := []struct {
		desc  string
		input struct {
			key, defaultValue string
		}
		want string
	}{
		{
			desc: "falls back to default value",
			input: struct{ key, defaultValue string }{
				key:          "__NONEXIST_KEY",
				defaultValue: "__DEFAULT_VALUE",
			},
			want: "__DEFAULT_VALUE",
		},
		{
			desc: "value exists",
			input: struct{ key, defaultValue string }{
				key:          "__EXISTING_KEY",
				defaultValue: "__DEFAULT_VALUE",
			},
			want: "VALUE",
		},
	}

	if err := os.Setenv("__EXISTING_KEY", "VALUE"); err != nil {
		t.Fatal(err)
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			got := DefaultEnv(test.input.key, test.input.defaultValue)

			if got != test.want {
				t.Errorf("%+v != %+v", got, test.want)
			}
		})
	}
}
