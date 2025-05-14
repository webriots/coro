package coro

import (
	"errors"
	"fmt"
	"unsafe"
)

var (
	// ErrCanceled is returned when a coroutine is canceled or when
	// yield/suspend is called on a completed or canceled coroutine.
	ErrCanceled = errors.New("coro: coroutine canceled")
	_           unsafe.Pointer
)

// coroutine represents a native Go coroutine instance. It's an opaque
// struct used by the runtime functions.
type coroutine struct{}

//go:linkname newcoro runtime.newcoro
func newcoro(func(*coroutine)) *coroutine

//go:linkname coroswitch runtime.coroswitch
func coroswitch(*coroutine)

// New creates a new coroutine with the provided function.
//
// Parameters:
//   - fn: A function that will be executed as a coroutine. The
//     function receives two parameters: a 'yield' function that
//     returns a value to the caller and pauses execution, and a
//     'suspend' function that pauses execution without returning a
//     value. Both functions return the value passed to resume when
//     execution continues.
//
// Returns:
//   - resume: A function used to pass values to the coroutine and
//     resume its execution. It returns the value yielded by the
//     coroutine and a boolean indicating whether the coroutine is
//     still running.
//   - cancel: A function used to cancel the coroutine's execution. If
//     the coroutine is running, it will panic with ErrCanceled when
//     it next yields or suspends.
//
// The generic type parameters allow for strongly typed coroutines:
//   - In: The type of values passed to the coroutine via resume
//   - Out: The type of values returned from the coroutine via yield
func New[In, Out any](
	fn func(func(Out) In, func() In) Out,
) (resume func(In) (Out, bool), cancel func()) {
	var (
		c    *coroutine
		in   In
		out  Out
		done bool
		perr error
	)

	c = newcoro(func(c *coroutine) {
		defer func() {
			if !done {
				if p := recover(); p != nil {
					perr = newPanicError(p)
				}
				done = true
			}
		}()

		yield := func(val Out) In {
			if done {
				panic(ErrCanceled)
			}
			out = val
			coroswitch(c)
			if perr != nil {
				panic(perr)
			}
			return in
		}

		suspend := func() In {
			if done {
				panic(ErrCanceled)
			}
			coroswitch(c)
			if perr != nil {
				panic(perr)
			}
			return in
		}

		if perr == nil {
			out = fn(yield, suspend)
		}
	})

	resume = func(val In) (Out, bool) {
		if perr != nil {
			panic(perr)
		}
		if done {
			var zero Out
			return zero, false
		}
		in = val
		coroswitch(c)
		if perr != nil {
			panic(perr)
		}
		return out, !done
	}

	cancel = func() {
		if done {
			return
		}
		canceled := fmt.Errorf("%w", ErrCanceled)
		perr = canceled
		coroswitch(c)
		if perr != nil && perr != canceled {
			panic(perr)
		}
	}

	return
}
