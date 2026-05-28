package errors

import "fmt"

// Error is the canonical error type. Always construct via New or Wrap.
// Code is a stable machine-readable string (e.g. "okr.plan.invalid_period")
// that maps 1:1 to an i18n message key.
type Error struct {
	Kind    Kind   // category for HTTP status + logging
	Code    string // stable code, dot-separated: <source>.<resource>.<reason>
	Message string // human-readable Chinese message for fallback (i18n key takes priority)
	Cause   error  // underlying cause (may be nil)
}

func New(kind Kind, code, message string) *Error {
	return &Error{Kind: kind, Code: code, Message: message}
}

func Wrap(kind Kind, code, message string, cause error) *Error {
	return &Error{Kind: kind, Code: code, Message: message, Cause: cause}
}

func (e *Error) Error() string {
	if e.Message == "" {
		return e.Code
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap supports errors.Is / errors.As.
func (e *Error) Unwrap() error { return e.Cause }
