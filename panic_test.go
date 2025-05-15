package coro

import (
	"errors"
	"fmt"
	"strings"
	"testing"
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
	if !strings.Contains(debugStr, "multiple errors") {
		t.Error("DebugString should contain 'multiple errors'")
	}
	if !strings.Contains(debugStr, "inner error 1") {
		t.Error("DebugString should contain 'inner error 1'")
	}
	if !strings.Contains(debugStr, "inner error 2") {
		t.Error("DebugString should contain 'inner error 2'")
	}
	if !strings.Contains(debugStr, "mock stack") {
		t.Error("DebugString should contain 'mock stack'")
	}
}

func TestDebugStringWithCircularReference(t *testing.T) {
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
	if !strings.Contains(debugStr, "self error") {
		t.Error("DebugString should contain 'self error'")
	}
	if !strings.Contains(debugStr, "mock stack") {
		t.Error("DebugString should contain 'mock stack'")
	}
	// Should not cause an infinite loop due to seen tracking
}

func TestPanicErrorUnwrapNonError(t *testing.T) {
	// Create a panicError with a non-error value
	pErr := &panicError{
		value: "not an error",
		stack: []byte("mock stack"),
	}

	// Test Unwrap returns nil for non-error values
	if pErr.Unwrap() != nil {
		t.Error("Expected Unwrap to return nil for non-error values")
	}
}

func TestPanicErrorMethods(t *testing.T) {
	// Test with error value
	errValue := fmt.Errorf("test error")
	pErr := &panicError{
		value: errValue,
		stack: []byte("mock stack"),
	}

	if pErr.Error() != "test error" {
		t.Errorf("Expected Error() to return 'test error', got '%s'", pErr.Error())
	}
	if !strings.Contains(pErr.ErrorWithStack(), "test error") {
		t.Error("ErrorWithStack() should contain 'test error'")
	}
	if !strings.Contains(pErr.ErrorWithStack(), "mock stack") {
		t.Error("ErrorWithStack() should contain 'mock stack'")
	}
	if pErr.Unwrap() != errValue {
		t.Errorf("Expected Unwrap() to return original error, got %v", pErr.Unwrap())
	}
}
