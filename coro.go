package coro

import (
	"errors"
	"fmt"
	"unsafe"
)

var (
	ErrCanceled = errors.New("coro: coroutine canceled")
	_           unsafe.Pointer
)

type coroutine struct{}

//go:linkname newcoro runtime.newcoro
func newcoro(func(*coroutine)) *coroutine

//go:linkname coroswitch runtime.coroswitch
func coroswitch(*coroutine)

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
