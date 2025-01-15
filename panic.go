package coro

import (
	"errors"
	"fmt"
	"runtime/debug"
	"strings"
)

type panicError struct {
	value any
	stack []byte
}

func (p *panicError) Error() string {
	return fmt.Sprintf("%v", p.value)
}

func (p *panicError) ErrorWithStack() string {
	return fmt.Sprintf("%v\n\n%s", p.value, p.stack)
}

func (p *panicError) Unwrap() error {
	err, ok := p.value.(error)
	if !ok {
		return nil
	}
	return err
}

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

func newPanicError(v any) error {
	return &panicError{
		value: v,
		stack: debug.Stack(),
	}
}
