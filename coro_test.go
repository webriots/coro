package coro

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func TestCoroutineYield(t *testing.T) {
	resume, cancel := New(func(yield func(string) int, suspend func() int) string {
		input := yield("first")
		if input != 1 {
			t.Errorf("Expected input to be 1, got %d", input)
		}

		input = yield("second")
		if input != 2 {
			t.Errorf("Expected input to be 2, got %d", input)
		}

		return "done"
	})
	defer cancel()

	out, running := resume(0)
	if !running {
		t.Error("Expected coroutine to be running")
	}
	if out != "first" {
		t.Errorf("Expected output to be 'first', got '%s'", out)
	}

	out, running = resume(1)
	if !running {
		t.Error("Expected coroutine to be running")
	}
	if out != "second" {
		t.Errorf("Expected output to be 'second', got '%s'", out)
	}

	out, running = resume(2)
	if running {
		t.Error("Expected coroutine to be completed")
	}
	if out != "done" {
		t.Errorf("Expected output to be 'done', got '%s'", out)
	}

	out, running = resume(3)
	if running {
		t.Error("Expected coroutine to be completed")
	}
	if out != "" {
		t.Errorf("Expected output to be empty, got '%s'", out)
	}
}

func TestCoroutineSuspend(t *testing.T) {
	resume, cancel := New(func(yield func(string) int, suspend func() int) string {
		input := suspend()
		if input != 1 {
			t.Errorf("Expected input to be 1, got %d", input)
		}

		input = yield("yielded")
		if input != 2 {
			t.Errorf("Expected input to be 2, got %d", input)
		}

		return "done"
	})
	defer cancel()

	out, running := resume(0)
	if !running {
		t.Error("Expected coroutine to be running")
	}
	if out != "" {
		t.Errorf("Expected output to be empty, got '%s'", out)
	}

	out, running = resume(1)
	if !running {
		t.Error("Expected coroutine to be running")
	}
	if out != "yielded" {
		t.Errorf("Expected output to be 'yielded', got '%s'", out)
	}

	out, running = resume(2)
	if running {
		t.Error("Expected coroutine to be completed")
	}
	if out != "done" {
		t.Errorf("Expected output to be 'done', got '%s'", out)
	}

	out, running = resume(3)
	if running {
		t.Error("Expected coroutine to be completed")
	}
	if out != "" {
		t.Errorf("Expected output to be empty, got '%s'", out)
	}
}

func TestCoroutinePanicOnlyRecovery(t *testing.T) {
	resume, cancel := New(func(yield func(string) int, suspend func() int) string {
		panic("test panic")
	})
	defer cancel()

	func() {
		defer func() {
			r := recover()
			if r == nil {
				t.Error("Expected panic but got none")
			}
			err, ok := r.(error)
			if !ok {
				t.Errorf("Expected error type from panic, got %T", r)
			}
			if err.Error() != "test panic" {
				t.Errorf("Expected panic message 'test panic', got '%s'", err.Error())
			}
		}()
		resume(0)
	}()
}

func TestCoroutinePanicRecovery(t *testing.T) {
	resume, _ := New(func(yield func(string) int, suspend func() int) string {
		input := yield("first")
		if input != 1 {
			t.Errorf("Expected input to be 1, got %d", input)
		}
		panic("test panic")
	})

	out, running := resume(0)
	if !running {
		t.Error("Expected coroutine to be running")
	}
	if out != "first" {
		t.Errorf("Expected output to be 'first', got '%s'", out)
	}

	func() {
		defer func() {
			r := recover()
			if r == nil {
				t.Error("Expected panic but got none")
			}
			err, ok := r.(error)
			if !ok {
				t.Errorf("Expected error type from panic, got %T", r)
			}
			if err.Error() != "test panic" {
				t.Errorf("Expected panic message 'test panic', got '%s'", err.Error())
			}
		}()
		resume(1)
	}()
}

func TestCoroutineCancel(t *testing.T) {
	returned := false
	defer func() {
		if !returned {
			t.Error("Expected returned to be true")
		}
	}()

	resume, cancel := New(func(yield func(string) int, suspend func() int) string {
		defer func() {
			returned = true
			p := recover()
			if p == nil {
				t.Error("Expected panic but got none")
			}
			err, ok := p.(error)
			if !ok {
				t.Errorf("Expected error type from panic, got %T", p)
			}
			if !errors.Is(err, ErrCanceled) {
				t.Errorf("Expected error to be ErrCanceled, got '%v'", err)
			}
		}()

		_ = yield("before cancel")
		panic("should not reach here")
	})

	out, running := resume(0)
	if !running {
		t.Error("Expected coroutine to be running")
	}
	if out != "before cancel" {
		t.Errorf("Expected output to be 'before cancel', got '%s'", out)
	}

	cancel()

	func() {
		defer func() {
			r := recover()
			if r == nil {
				t.Error("Expected panic but got none")
			}
			err, ok := r.(error)
			if !ok {
				t.Errorf("Expected error type from panic, got %T", r)
			}
			if err.Error() != ErrCanceled.Error() {
				t.Errorf("Expected panic message '%s', got '%s'", ErrCanceled.Error(), err.Error())
			}
		}()
		resume(0)
	}()
}

func TestCoroutinePanicInYield(t *testing.T) {
	returned := false
	defer func() {
		if !returned {
			t.Error("Expected returned to be true")
		}
	}()

	resume, _ := New(func(yield func(string) int, suspend func() int) string {
		defer func() { returned = true }()
		yield("first yield")
		panic("panic after yield")
	})

	out, running := resume(0)
	if !running {
		t.Error("Expected coroutine to be running")
	}
	if out != "first yield" {
		t.Errorf("Expected output to be 'first yield', got '%s'", out)
	}

	func() {
		defer func() {
			r := recover()
			if r == nil {
				t.Error("Expected panic but got none")
			}
			err, ok := r.(error)
			if !ok {
				t.Errorf("Expected error type from panic, got %T", r)
			}
			if err.Error() != "panic after yield" {
				t.Errorf("Expected panic message 'panic after yield', got '%s'", err.Error())
			}
		}()
		resume(0)
	}()
}

func TestCoroutineResumeAfterCompletion(t *testing.T) {
	resume, _ := New(func(yield func(string) int, suspend func() int) string {
		return "completed"
	})

	out, running := resume(0)
	if running {
		t.Error("Expected coroutine to be completed")
	}
	if out != "completed" {
		t.Errorf("Expected output to be 'completed', got '%s'", out)
	}

	out, running = resume(0)
	if running {
		t.Error("Expected coroutine to be completed")
	}
	if out != "" {
		t.Errorf("Expected output to be empty, got '%s'", out)
	}
}

func TestCoroutineMultipleCancels(t *testing.T) {
	returned := false
	defer func() {
		if !returned {
			t.Error("Expected returned to be true")
		}
	}()

	resume, cancel := New(func(yield func(string) int, suspend func() int) string {
		defer func() {
			returned = true
			p := recover()
			if p == nil {
				t.Error("Expected panic but got none")
			}
			err, ok := p.(error)
			if !ok {
				t.Errorf("Expected error type from panic, got %T", p)
			}
			if !errors.Is(err, ErrCanceled) {
				t.Errorf("Expected error to be ErrCanceled, got '%v'", err)
			}
		}()

		yield("before cancel")
		t.Error("coroutine should have been canceled")
		panic("should not reach here")
	})

	out, running := resume(0)
	if !running {
		t.Error("Expected coroutine to be running")
	}
	if out != "before cancel" {
		t.Errorf("Expected output to be 'before cancel', got '%s'", out)
	}

	cancel()
	cancel()
	cancel()
}

func TestCoroutineYieldEscaped(t *testing.T) {
	var yieldEscaped func(string) int

	resume, _ := New(func(yield func(string) int, suspend func() int) string {
		yieldEscaped = yield
		yield("first yield")
		return "done"
	})

	out, running := resume(0)
	if !running {
		t.Error("Expected coroutine to be running")
	}
	if out != "first yield" {
		t.Errorf("Expected output to be 'first yield', got '%s'", out)
	}

	out, running = resume(1)
	if running {
		t.Error("Expected coroutine to be completed")
	}
	if out != "done" {
		t.Errorf("Expected output to be 'done', got '%s'", out)
	}

	func() {
		defer func() {
			r := recover()
			if r == nil {
				t.Error("Expected panic but got none")
			}
			err, ok := r.(error)
			if !ok {
				t.Errorf("Expected error type from panic, got %T", r)
			}
			if err.Error() != ErrCanceled.Error() {
				t.Errorf("Expected panic message '%s', got '%s'", ErrCanceled.Error(), err.Error())
			}
		}()
		yieldEscaped("already done")
	}()
}

func TestCoroutineSuspendEscaped(t *testing.T) {
	var suspendEscaped func() int

	resume, _ := New(func(yield func(string) int, suspend func() int) string {
		suspendEscaped = suspend
		yield("first yield")
		return "done"
	})

	out, running := resume(0)
	if !running {
		t.Error("Expected coroutine to be running")
	}
	if out != "first yield" {
		t.Errorf("Expected output to be 'first yield', got '%s'", out)
	}

	out, running = resume(1)
	if running {
		t.Error("Expected coroutine to be completed")
	}
	if out != "done" {
		t.Errorf("Expected output to be 'done', got '%s'", out)
	}

	func() {
		defer func() {
			r := recover()
			if r == nil {
				t.Error("Expected panic but got none")
			}
			err, ok := r.(error)
			if !ok {
				t.Errorf("Expected error type from panic, got %T", r)
			}
			if err.Error() != ErrCanceled.Error() {
				t.Errorf("Expected panic message '%s', got '%s'", ErrCanceled.Error(), err.Error())
			}
		}()
		suspendEscaped()
	}()
}

func TestCoroutineYieldEscapedCancel(t *testing.T) {
	returned := false
	defer func() {
		if !returned {
			t.Error("Expected returned to be true")
		}
	}()

	var yieldEscaped func(string) int

	resume, cancel := New(func(yield func(string) int, suspend func() int) string {
		defer func() { returned = true }()
		yieldEscaped = yield
		func() {
			defer func() {
				r := recover()
				if r == nil {
					t.Error("Expected panic but got none")
				}
				err, ok := r.(error)
				if !ok {
					t.Errorf("Expected error type from panic, got %T", r)
				}
				if err.Error() != ErrCanceled.Error() {
					t.Errorf("Expected panic message '%s', got '%s'", ErrCanceled.Error(), err.Error())
				}
			}()
			yield("first yield")
		}()
		return "done"
	})

	out, running := resume(0)
	if !running {
		t.Error("Expected coroutine to be running")
	}
	if out != "first yield" {
		t.Errorf("Expected output to be 'first yield', got '%s'", out)
	}

	cancel()

	func() {
		defer func() {
			r := recover()
			if r == nil {
				t.Error("Expected panic but got none")
			}
			err, ok := r.(error)
			if !ok {
				t.Errorf("Expected error type from panic, got %T", r)
			}
			if err.Error() != ErrCanceled.Error() {
				t.Errorf("Expected panic message '%s', got '%s'", ErrCanceled.Error(), err.Error())
			}
		}()
		yieldEscaped("already done")
	}()
}

func TestCoroutineSuspendEscapedCancel(t *testing.T) {
	returned := false
	defer func() {
		if !returned {
			t.Error("Expected returned to be true")
		}
	}()

	var suspendEscaped func() int

	resume, cancel := New(func(yield func(string) int, suspend func() int) string {
		defer func() { returned = true }()
		suspendEscaped = suspend
		func() {
			defer func() {
				r := recover()
				if r == nil {
					t.Error("Expected panic but got none")
				}
			}()
			suspend()
		}()
		return "done"
	})

	out, running := resume(0)
	if !running {
		t.Error("Expected coroutine to be running")
	}
	if out != "" {
		t.Errorf("Expected output to be empty, got '%s'", out)
	}

	cancel()

	func() {
		defer func() {
			r := recover()
			if r == nil {
				t.Error("Expected panic but got none")
			}
			err, ok := r.(error)
			if !ok {
				t.Errorf("Expected error type from panic, got %T", r)
			}
			if err.Error() != ErrCanceled.Error() {
				t.Errorf("Expected panic message '%s', got '%s'", ErrCanceled.Error(), err.Error())
			}
		}()
		suspendEscaped()
	}()
}

func TestCoroutineCancelWithPanic(t *testing.T) {
	returned := false
	defer func() {
		if !returned {
			t.Error("Expected returned to be true")
		}
	}()

	resume, cancel := New(func(yield func(string) int, suspend func() int) string {
		defer func() {
			returned = true
			p := recover()
			if p == nil {
				t.Error("Expected panic but got none")
			}
			err, ok := p.(error)
			if !ok {
				t.Errorf("Expected error type from panic, got %T", p)
			}
			if !errors.Is(err, ErrCanceled) {
				t.Errorf("Expected error to be ErrCanceled, got '%v'", err)
			}
		}()

		_ = suspend()
		panic("panic after suspend")
	})

	out, running := resume(0)
	if !running {
		t.Error("Expected coroutine to be running")
	}
	if out != "" {
		t.Errorf("Expected output to be empty, got '%s'", out)
	}

	cancel()

	func() {
		defer func() {
			r := recover()
			if r == nil {
				t.Error("Expected panic but got none")
			}
			err, ok := r.(error)
			if !ok {
				t.Errorf("Expected error type from panic, got %T", r)
			}
			if err.Error() != ErrCanceled.Error() {
				t.Errorf("Expected panic message '%s', got '%s'", ErrCanceled.Error(), err.Error())
			}
		}()
		resume(0)
	}()

	cancel()
}

func TestResumeAfterCoroutinePanic(t *testing.T) {
	resume, cancel := New(func(yield func(string) int, suspend func() int) string {
		panic("test panic")
	})

	func() {
		defer func() {
			r := recover()
			if r == nil {
				t.Error("Expected panic but got none")
			}
			err, ok := r.(error)
			if !ok {
				t.Errorf("Expected error type from panic, got %T", r)
			}
			if err.Error() != "test panic" {
				t.Errorf("Expected panic message 'test panic', got '%s'", err.Error())
			}
		}()
		resume(0)
	}()

	func() {
		defer func() {
			r := recover()
			if r == nil {
				t.Error("Expected panic but got none")
			}
			err, ok := r.(error)
			if !ok {
				t.Errorf("Expected error type from panic, got %T", r)
			}
			if err.Error() != "test panic" {
				t.Errorf("Expected panic message 'test panic', got '%s'", err.Error())
			}
		}()
		resume(1)
	}()

	cancel()
}

func TestCoroutineCancelBeforeResume(t *testing.T) {
	resume, cancel := New(func(yield func(string) int, suspend func() int) string {
		t.Error("coroutine should not start")
		panic("should not reach here")
	})

	cancel()

	func() {
		defer func() {
			r := recover()
			if r == nil {
				t.Error("Expected panic but got none")
			}
			err, ok := r.(error)
			if !ok {
				t.Errorf("Expected error type from panic, got %T", r)
			}
			if err.Error() != ErrCanceled.Error() {
				t.Errorf("Expected panic message '%s', got '%s'", ErrCanceled.Error(), err.Error())
			}
		}()
		resume(0)
	}()
}

func TestCancelDuringCoroutinePanic(t *testing.T) {
	returned := false
	defer func() {
		if !returned {
			t.Error("Expected returned to be true")
		}
	}()

	resume, cancel := New(func(yield func(string) int, suspend func() int) string {
		// Simulate the coro recovering a panic and then panic'ing again.
		defer func() {
			returned = true
			panic("deferred error")
		}()
		func() {
			defer func() {
				r := recover()
				if r == nil {
					t.Error("Expected panic but got none")
				}
			}()
			yield("before panic")
		}()
		return ""
	})

	out, running := resume(0)
	if !running {
		t.Error("Expected coroutine to be running")
	}
	if out != "before panic" {
		t.Errorf("Expected output to be 'before panic', got '%s'", out)
	}

	func() {
		defer func() {
			r := recover()
			if r == nil {
				t.Error("Expected panic but got none")
			}
			err, ok := r.(error)
			if !ok {
				t.Errorf("Expected error type from panic, got %T", r)
			}
			if err.Error() != "deferred error" {
				t.Errorf("Expected panic message 'deferred error', got '%s'", err.Error())
			}
		}()
		cancel()
	}()
}

func TestDebugString(t *testing.T) {
	resume, _ := New(func(yield func(string) int, suspend func() int) string {
		resume, cancel := New(func(yield func(string) int, suspend func() int) string {
			panic("test panic")
		})
		defer cancel()
		resume(0)
		return ""
	})

	defer func() {
		p := recover()
		if p == nil {
			t.Error("Expected panic but got none")
			return
		}

		err, ok := p.(interface{ DebugString() string })
		if !ok {
			t.Errorf("Expected error with DebugString method, got %T", p)
			return
		}

		msg := err.DebugString()

		var (
			lineNums     []int
			lineNumRegex = regexp.MustCompile(`:(\d+) \+`)
		)

		for _, line := range strings.Split(msg, "\n") {
			if strings.Contains(line, "coro_test.go:") {
				matches := lineNumRegex.FindStringSubmatch(line)
				if len(matches) != 2 {
					t.Errorf("Expected 2 matches, got %d", len(matches))
					continue
				}

				lineNum, err := strconv.Atoi(matches[1])
				if err != nil {
					t.Errorf("Error converting line number: %v", err)
					continue
				}

				lineNums = append(lineNums, lineNum)
			}
		}

		if len(lineNums) != 2 {
			t.Errorf("Expected 2 line numbers, got %d", len(lineNums))
			return
		}
		if lineNums[0]-lineNums[1] != 3 {
			t.Errorf("Expected line difference of 3, got %d", lineNums[0]-lineNums[1])
		}
	}()

	resume(0)
}
