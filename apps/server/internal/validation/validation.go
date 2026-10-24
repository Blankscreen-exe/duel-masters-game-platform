// Package validation holds the error every feature returns for input that
// breaks its rules, so the HTTP layer answers all of them the same way.
package validation

import (
	"errors"
	"fmt"
)

// Error reports one invalid field and what is wrong with it, in words the
// player can act on, such as "must be at least 8 characters".
type Error struct {
	Field  string
	Reason string
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s %s", e.Field, e.Reason)
}

// New returns an *Error for field.
func New(field, reason string) *Error {
	return &Error{Field: field, Reason: reason}
}

// As reports whether err is, or wraps, an *Error, and returns it.
func As(err error) (*Error, bool) {
	var invalid *Error
	ok := errors.As(err, &invalid)
	return invalid, ok
}
