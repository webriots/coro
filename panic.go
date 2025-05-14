package coro

import (
	"errors"
	"fmt"
	"runtime/debug"
	"strings"
)

// panicError wraps a panic value with its stack trace. It implements
// the error interface and provides additional methods for debugging
// and error unwrapping.
type panicError struct {
	value any
	stack []byte
}

// Error returns the string representation of the panic value.
func (p *panicError) Error() string {
	return fmt.Sprintf("%v", p.value)
}

// ErrorWithStack returns the panic value along with its stack trace.
func (p *panicError) ErrorWithStack() string {
	return fmt.Sprintf("%v\n\n%s", p.value, p.stack)
}

// Unwrap returns the underlying error if the panic value was an
// error. This allows for errors.Is() and errors.As() compatibility.
func (p *panicError) Unwrap() error {
	err, ok := p.value.(error)
	if !ok {
		return nil
	}
	return err
}

// DebugString returns a detailed error message that includes the
// panic value and stack trace information, as well as any unwrapped
// errors. This is useful for comprehensive debugging of nested
// errors.
func (p *panicError) DebugString() string {
	var sb strings.Builder
	seen := make(map[error]bool)

	var unwrap func(error)
	unwrap = func(e error) {
		if e == nil || seen[e] {
			return
		}
		seen[e] = true

		if p, ok := e.(*panicError); ok {
			sb.WriteString(p.ErrorWithStack())
		} else {
			sb.WriteString(e.Error())
		}

		if unwrapper, ok := e.(interface{ Unwrap() []error }); ok {
			for _, ue := range unwrapper.Unwrap() {
				unwrap(ue)
			}
		} else if ue := errors.Unwrap(e); ue != nil {
			unwrap(ue)
		}
	}

	unwrap(p)
	return sb.String()
}

// newPanicError creates a new panicError that wraps the provided
// panic value with a stack trace captured at the call site.
func newPanicError(v any) error {
	return &panicError{
		value: v,
		stack: debug.Stack(),
	}
}
