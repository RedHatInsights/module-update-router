package main

import (
	"database/sql"
)

// NewNullString creates a sql.NullString from s, returning an empty NullString
// if s is nil or a zero-value string.
func NewNullString(s *string) sql.NullString {
	if s == nil || *s == "" {
		return sql.NullString{}
	}
	return sql.NullString{Valid: true, String: *s}
}
