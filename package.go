// Package coro provides an implementation of coroutines for Go,
// enabling cooperative multitasking with lightweight functions that
// can suspend their execution and resume later. Coroutines offer a
// powerful control flow mechanism for implementing generators,
// iterators, and asynchronous patterns with explicit yield points.
//
// A coroutine is created using the New function, which returns resume
// and cancel functions. The resume function is used to pass values to
// the coroutine and receive values from it, while the cancel function
// is used to terminate a coroutine's execution early.
//
// Within a coroutine function, the yield parameter allows returning a
// value to the caller while pausing execution, and the suspend
// parameter allows pausing execution without returning a value. Both
// functions allow receiving values from the caller when execution
// resumes.
//
// The package handles panics within coroutines appropriately,
// wrapping and propagating them to the caller with helpful stack
// traces. It also ensures that escaped yield and suspend functions
// cannot be misused after coroutine completion.
package coro
