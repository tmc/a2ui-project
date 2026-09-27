package a2uischema

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Validation failures. A [ValidationError] wraps one of these.
var (
	// ErrInvalidMessage reports a message that is malformed or is
	// missing a required field.
	ErrInvalidMessage = errors.New("a2uischema: invalid message")

	// ErrVersionMismatch reports a message for a protocol version
	// other than [a2ui.Version].
	ErrVersionMismatch = errors.New("a2uischema: version mismatch")

	// ErrUnknownComponent reports a component type that the catalog
	// does not define.
	ErrUnknownComponent = errors.New("a2uischema: unknown component")

	// ErrUnknownFunction reports a function that the catalog does not
	// define.
	ErrUnknownFunction = errors.New("a2uischema: unknown function")

	// ErrInvalidTree reports a surface whose components do not form a
	// tree: a duplicate id, a missing root, a reference to an unknown
	// component, a cycle or an orphaned component.
	ErrInvalidTree = errors.New("a2uischema: invalid component tree")

	// ErrNotAllowed reports a component whose parent or child is not
	// permitted by the allowedParents or allowedChildren of the catalog.
	ErrNotAllowed = errors.New("a2uischema: component not allowed")
)

// A ValidationError describes a validation failure.
type ValidationError struct {
	// Path is the JSON Pointer (RFC 6901) of the offending value in
	// the validated messages, taken as a JSON array, such as
	// /0/updateComponents/components/3/child. When a single message
	// object is validated, the pointer is relative to that object.
	Path string

	// Err is the class of failure, one of the Err variables in this
	// package, possibly wrapping the underlying error.
	Err error

	msg string // detail, without the package prefix
}

// Error returns the detailed message, prefixed with "a2uischema: ".
func (e *ValidationError) Error() string {
	return "a2uischema: " + e.msg
}

// Unwrap returns e.Err.
func (e *ValidationError) Unwrap() error {
	return e.Err
}

// invalid returns a *ValidationError of class err at the JSON pointer
// path, relative to the value being validated.
func invalid(err error, path, msg string) *ValidationError {
	return &ValidationError{Path: path, Err: err, msg: msg}
}

// within reports err as occurring at the JSON pointer path, which is
// relative to the enclosing value, adding context to its message.
// The context may be empty. An err that is not a *ValidationError is
// reported as an [ErrInvalidMessage].
func within(err error, path, context string) error {
	if err == nil {
		return nil
	}
	e, ok := err.(*ValidationError)
	if !ok {
		e = invalid(fmt.Errorf("%w: %w", ErrInvalidMessage, err), "", err.Error())
	}
	e.Path = path + e.Path
	if context != "" {
		e.msg = context + ": " + e.msg
	}
	return e
}

// pointer returns the JSON pointer of the reference tokens.
func pointer(tokens ...any) string {
	var b strings.Builder
	for _, t := range tokens {
		b.WriteByte('/')
		switch t := t.(type) {
		case int:
			b.WriteString(strconv.Itoa(t))
		case string:
			b.WriteString(strings.NewReplacer("~", "~0", "/", "~1").Replace(t))
		}
	}
	return b.String()
}
