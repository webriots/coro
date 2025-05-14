package coro

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

// multiError implements unwrapping to multiple errors
type multiError struct {
	errs []error
}

func (m *multiError) Error() string {
	return "multiple errors"
}

func (m *multiError) Unwrap() []error {
	return m.errs
}

// selfReferentialError creates a circular reference to test the seen error detection
type selfReferentialError struct {
	err error
	msg string
}

func (s *selfReferentialError) Error() string {
	return s.msg
}

func (s *selfReferentialError) Unwrap() error {
	return s.err
}

func TestDebugStringWithMultipleErrors(t *testing.T) {
	r := require.New(t)

	// Create an error that unwraps to multiple errors
	innerErr1 := errors.New("inner error 1")
	innerErr2 := errors.New("inner error 2")
	multiErr := &multiError{errs: []error{innerErr1, innerErr2}}

	// Create a panicError with this value
	pErr := &panicError{
		value: multiErr,
		stack: []byte("mock stack"),
	}

	// Test DebugString
	debugStr := pErr.DebugString()
	r.Contains(debugStr, "multiple errors")
	r.Contains(debugStr, "inner error 1")
	r.Contains(debugStr, "inner error 2")
	r.Contains(debugStr, "mock stack")
}

func TestDebugStringWithCircularReference(t *testing.T) {
	r := require.New(t)

	// Create an error with a circular reference
	selfErr := &selfReferentialError{msg: "self error"}
	selfErr.err = selfErr // circular reference

	// Create a panicError with this value
	pErr := &panicError{
		value: selfErr,
		stack: []byte("mock stack"),
	}

	// Test DebugString
	debugStr := pErr.DebugString()
	r.Contains(debugStr, "self error")
	r.Contains(debugStr, "mock stack")
	// Should not cause an infinite loop due to seen tracking
}

func TestPanicErrorUnwrapNonError(t *testing.T) {
	r := require.New(t)

	// Create a panicError with a non-error value
	pErr := &panicError{
		value: "not an error",
		stack: []byte("mock stack"),
	}

	// Test Unwrap returns nil for non-error values
	r.Nil(pErr.Unwrap())
}

func TestPanicErrorMethods(t *testing.T) {
	r := require.New(t)

	// Test with error value
	errValue := fmt.Errorf("test error")
	pErr := &panicError{
		value: errValue,
		stack: []byte("mock stack"),
	}

	r.Equal("test error", pErr.Error())
	r.Contains(pErr.ErrorWithStack(), "test error")
	r.Contains(pErr.ErrorWithStack(), "mock stack")
	r.Equal(errValue, pErr.Unwrap())
}
