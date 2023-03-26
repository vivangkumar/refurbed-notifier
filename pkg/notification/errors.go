package notification

import (
	"fmt"
	"time"
)

// requestError is the internal error type returned
// when a Notify request fails.
//
// It encapsulates the underlying error, along with
// the message that failed and if the error can be retried
// or not.
//
// Callers should test errors for the IsRetryable and
// Message methods using errors.As.

type requestError struct {
	err       error
	msg       Message
	retryable bool
}

func newRequestError(status int, msg Message, retryable bool) error {
	return requestError{
		err:       fmt.Errorf("request failed with status: %d", status),
		msg:       msg,
		retryable: retryable,
	}
}

func (re requestError) Error() string {
	return re.err.Error()
}

// IsRetryable determines if an error can be retried.
func (re requestError) IsRetryable() bool {
	return re.retryable
}

// Message returns the message that failed when
// a Notify attempt was made.
func (re requestError) Message() Message {
	return re.msg
}

// enqueueError is the internal error type for
// enqueuing messages.
//
// Callers should test errors for IsTemporary
// and retry the call again.
type enqueueError struct {
	err error
}

func newEnqueueError(err error) error {
	return enqueueError{err: err}
}

// Error implements the error interface.
func (er enqueueError) Error() string {
	return er.err.Error()
}

// IsTemporary returns if the error is temporary in nature.
func (er enqueueError) IsTemporary() bool {
	return true
}

// RetryAfter returns the time duration after which enqueueing
// can be tried again.
func (er enqueueError) RetryAfter() time.Duration {
	return defaultEnqueueRetryDuration
}
