package a2uistream

import "errors"

// ErrInvalidPayload reports an A2UI payload that cannot be used as
// A2UI 1.x messages: it is empty, is not JSON, is for another protocol
// version or does not decode as a 1.x message.
var ErrInvalidPayload = errors.New("a2uistream: invalid payload")

// A payloadError is an [ErrInvalidPayload] with its own message.
type payloadError struct {
	msg string
	err error // underlying error, or nil
}

func (e *payloadError) Error() string {
	return e.msg
}

func (e *payloadError) Unwrap() []error {
	if e.err == nil {
		return []error{ErrInvalidPayload}
	}
	return []error{ErrInvalidPayload, e.err}
}
