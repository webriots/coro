package coro

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCoroutineYield(t *testing.T) {
	r := require.New(t)

	resume, cancel := New(func(yield func(string) int, suspend func() int) string {
		input := yield("first")
		r.Equal(1, input)

		input = yield("second")
		r.Equal(2, input)

		return "done"
	})
	defer cancel()

	out, running := resume(0)
	r.True(running)
	r.Equal("first", out)

	out, running = resume(1)
	r.True(running)
	r.Equal("second", out)

	out, running = resume(2)
	r.False(running)
	r.Equal("done", out)

	out, running = resume(3)
	r.False(running)
	r.Equal("", out)
}

func TestCoroutineSuspend(t *testing.T) {
	r := require.New(t)

	resume, cancel := New(func(yield func(string) int, suspend func() int) string {
		input := suspend()
		r.Equal(1, input)

		input = yield("yielded")
		r.Equal(2, input)

		return "done"
	})
	defer cancel()

	out, running := resume(0)
	r.True(running)
	r.Equal("", out)

	out, running = resume(1)
	r.True(running)
	r.Equal("yielded", out)

	out, running = resume(2)
	r.False(running)
	r.Equal("done", out)

	out, running = resume(3)
	r.False(running)
	r.Equal("", out)
}

func TestCoroutinePanicOnlyRecovery(t *testing.T) {
	r := require.New(t)

	resume, cancel := New(func(yield func(string) int, suspend func() int) string {
		panic("test panic")
	})
	defer cancel()

	r.PanicsWithError("test panic", func() {
		resume(0)
	})
}

func TestCoroutinePanicRecovery(t *testing.T) {
	r := require.New(t)

	resume, _ := New(func(yield func(string) int, suspend func() int) string {
		input := yield("first")
		r.Equal(1, input)
		panic("test panic")
	})

	out, running := resume(0)
	r.True(running)
	r.Equal("first", out)

	r.PanicsWithError("test panic", func() {
		resume(1)
	})
}

func TestCoroutineCancel(t *testing.T) {
	r := require.New(t)

	returned := false
	defer func() { r.True(returned) }()

	resume, cancel := New(func(yield func(string) int, suspend func() int) string {
		defer func() {
			returned = true
			p := recover()
			err, ok := p.(error)
			r.NotNil(err)
			r.True(ok)
			r.ErrorIs(err, ErrCanceled)
		}()

		_ = yield("before cancel")
		panic("should not reach here")
	})

	out, running := resume(0)
	r.True(running)
	r.Equal("before cancel", out)

	cancel()

	r.PanicsWithError(ErrCanceled.Error(), func() {
		resume(0)
	})
}

func TestCoroutinePanicInYield(t *testing.T) {
	r := require.New(t)

	returned := false
	defer func() { r.True(returned) }()

	resume, _ := New(func(yield func(string) int, suspend func() int) string {
		defer func() { returned = true }()
		yield("first yield")
		panic("panic after yield")
	})

	out, running := resume(0)
	r.True(running)
	r.Equal("first yield", out)

	r.PanicsWithError("panic after yield", func() {
		resume(0)
	})
}

func TestCoroutineResumeAfterCompletion(t *testing.T) {
	r := require.New(t)

	resume, _ := New(func(yield func(string) int, suspend func() int) string {
		return "completed"
	})

	out, running := resume(0)
	r.False(running)
	r.Equal("completed", out)

	out, running = resume(0)
	r.False(running)
	r.Equal("", out)
}

func TestCoroutineMultipleCancels(t *testing.T) {
	r := require.New(t)

	returned := false
	defer func() { r.True(returned) }()

	resume, cancel := New(func(yield func(string) int, suspend func() int) string {
		defer func() {
			returned = true
			p := recover()
			err, ok := p.(error)
			r.NotNil(err)
			r.True(ok)
			r.ErrorIs(err, ErrCanceled)
		}()

		yield("before cancel")
		r.Fail("coroutine should have been canceled")
		panic("should not reach here")
	})

	out, running := resume(0)
	r.True(running)
	r.Equal("before cancel", out)

	cancel()
	cancel()
	cancel()
}

func TestCoroutineYieldEscaped(t *testing.T) {
	r := require.New(t)

	var yieldEscaped func(string) int

	resume, _ := New(func(yield func(string) int, suspend func() int) string {
		yieldEscaped = yield
		yield("first yield")
		return "done"
	})

	out, running := resume(0)
	r.True(running)
	r.Equal("first yield", out)

	out, running = resume(1)
	r.False(running)
	r.Equal("done", out)

	r.PanicsWithError(ErrCanceled.Error(), func() {
		yieldEscaped("already done")
	})
}

func TestCoroutineSuspendEscaped(t *testing.T) {
	r := require.New(t)

	var suspendEscaped func() int

	resume, _ := New(func(yield func(string) int, suspend func() int) string {
		suspendEscaped = suspend
		yield("first yield")
		return "done"
	})

	out, running := resume(0)
	r.True(running)
	r.Equal("first yield", out)

	out, running = resume(1)
	r.False(running)
	r.Equal("done", out)

	r.PanicsWithError(ErrCanceled.Error(), func() {
		suspendEscaped()
	})
}

func TestCoroutineYieldEscapedCancel(t *testing.T) {
	r := require.New(t)

	returned := false
	defer func() { r.True(returned) }()

	var yieldEscaped func(string) int

	resume, cancel := New(func(yield func(string) int, suspend func() int) string {
		defer func() { returned = true }()
		yieldEscaped = yield
		r.PanicsWithError(ErrCanceled.Error(), func() {
			yield("first yield")
		})
		return "done"
	})

	out, running := resume(0)
	r.True(running)
	r.Equal("first yield", out)

	cancel()

	r.PanicsWithError(ErrCanceled.Error(), func() {
		yieldEscaped("already done")
	})
}

func TestCoroutineSuspendEscapedCancel(t *testing.T) {
	r := require.New(t)

	returned := false
	defer func() { r.True(returned) }()

	var suspendEscaped func() int

	resume, cancel := New(func(yield func(string) int, suspend func() int) string {
		defer func() { returned = true }()
		suspendEscaped = suspend
		r.Panics(func() { suspend() })
		return "done"
	})

	out, running := resume(0)
	r.True(running)
	r.Equal("", out)

	cancel()

	r.PanicsWithError(ErrCanceled.Error(), func() {
		suspendEscaped()
	})
}

func TestCoroutineCancelWithPanic(t *testing.T) {
	r := require.New(t)

	returned := false
	defer func() { r.True(returned) }()

	resume, cancel := New(func(yield func(string) int, suspend func() int) string {
		defer func() {
			returned = true
			p := recover()
			err, ok := p.(error)
			r.NotNil(err)
			r.True(ok)
			r.ErrorIs(err, ErrCanceled)
		}()

		_ = suspend()
		panic("panic after suspend")
	})

	out, running := resume(0)
	r.True(running)
	r.Equal("", out)

	cancel()

	r.PanicsWithError(ErrCanceled.Error(), func() {
		resume(0)
	})

	cancel()
}

func TestResumeAfterCoroutinePanic(t *testing.T) {
	r := require.New(t)

	resume, cancel := New(func(yield func(string) int, suspend func() int) string {
		panic("test panic")
	})

	r.PanicsWithError("test panic", func() {
		resume(0)
	})

	r.PanicsWithError("test panic", func() {
		resume(1)
	})

	cancel()
}

func TestCoroutineCancelBeforeResume(t *testing.T) {
	r := require.New(t)

	resume, cancel := New(func(yield func(string) int, suspend func() int) string {
		r.Fail("coroutine should not start")
		panic("should not reach here")
	})

	cancel()

	r.PanicsWithError(ErrCanceled.Error(), func() {
		resume(0)
	})
}

func TestCancelDuringCoroutinePanic(t *testing.T) {
	r := require.New(t)

	returned := false
	defer func() { r.True(returned) }()

	resume, cancel := New(func(yield func(string) int, suspend func() int) string {
		// Simulate the coro recovering a panic and then panic'ing again.
		defer func() {
			returned = true
			panic("deferred error")
		}()
		r.Panics(func() { yield("before panic") })
		return ""
	})

	out, running := resume(0)
	r.True(running)
	r.Equal("before panic", out)

	r.PanicsWithError("deferred error", cancel)
}

func TestDebugString(t *testing.T) {
	r := require.New(t)

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
		r.NotNil(p)

		err, ok := p.(interface{ DebugString() string })
		r.True(ok)

		msg := err.DebugString()

		var (
			lineNums     []int
			lineNumRegex = regexp.MustCompile(`:(\d+) \+`)
		)

		for _, line := range strings.Split(msg, "\n") {
			if strings.Contains(line, "coro_test.go:") {
				matches := lineNumRegex.FindStringSubmatch(line)
				r.Len(matches, 2)

				lineNum, err := strconv.Atoi(matches[1])
				r.NoError(err)

				lineNums = append(lineNums, lineNum)
			}
		}

		r.Len(lineNums, 2)
		r.Equal(3, lineNums[0]-lineNums[1])
	}()

	resume(0)
}
