package postgres

import (
	"errors"
	"reflect"
)

const (
	sqlStateUniqueViolation = "23505"
)

func MapUniqueConstraint(err error, m map[string]error) error {
	if c, ok := isUniqueViolation(err); ok {
		if mapped := m[c]; mapped != nil {
			return mapped
		}
	}
	return err
}

func isUniqueViolation(err error) (string, bool) {
	return isSQLState(err, sqlStateUniqueViolation)
}

func isSQLState(err error, want string) (string, bool) {
	return walkErrors(err, func(e error) (string, bool) {
		// lib/pq style: SQLState() string
		type sqlStater interface {
			SQLState() string
		}
		if s, ok := e.(sqlStater); ok && s.SQLState() == want {
			return extractConstraint(e), true
		}

		// pgconn.PgError style: field Code == "23505"
		if code, ok := extractStringField(e, "Code"); ok && code == want {
			return extractConstraint(e), true
		}

		return "", false
	})
}

func walkErrors(err error, fn func(error) (string, bool)) (string, bool) {
	if err == nil {
		return "", false
	}
	if v, ok := fn(err); ok {
		return v, true
	}

	type unwrapperMulti interface {
		Unwrap() []error
	}
	if m, ok := err.(unwrapperMulti); ok {
		for _, child := range m.Unwrap() {
			if v, ok := walkErrors(child, fn); ok {
				return v, true
			}
		}
		return "", false
	}

	if u := errors.Unwrap(err); u != nil {
		return walkErrors(u, fn)
	}
	return "", false
}

func extractConstraint(err error) string {
	if v, ok := extractStringField(err, "ConstraintName"); ok && v != "" {
		return v
	}
	if v, ok := extractStringField(err, "Constraint"); ok && v != "" {
		return v
	}
	return ""
}

func extractStringField(err error, field string) (string, bool) {
	rv := reflect.ValueOf(err)
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return "", false
		}
		rv = rv.Elem()
	}
	if !rv.IsValid() || rv.Kind() != reflect.Struct {
		return "", false
	}

	f := rv.FieldByName(field)
	if !f.IsValid() || f.Kind() != reflect.String {
		return "", false
	}
	return f.String(), true
}
