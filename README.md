# coro

[![Go Reference](https://pkg.go.dev/badge/github.com/webriots/coro.svg)](https://pkg.go.dev/github.com/webriots/coro)
[![Go Report Card](https://goreportcard.com/badge/github.com/webriots/coro)](https://goreportcard.com/report/github.com/webriots/coro)
[![Coverage Status](https://coveralls.io/repos/github/webriots/coro/badge.svg?branch=main)](https://coveralls.io/github/webriots/coro?branch=main)


A Go implementation of coroutines that provides cooperative multitasking using Go's native coroutine support.

## Features

- Create and manage coroutines that can yield control flow
- Strongly typed inputs and outputs with generics
- Support for yielding values and suspending execution
- Complete panic handling and propagation
- Cancellation mechanism for coroutines

## Installation

```shell
go get github.com/webriots/coro
```

> [!IMPORTANT]
> The `-ldflags=-checklinkname=0` flag is required when building and testing this library since it uses the `//go:linkname` directive to access internal Go runtime functions. As of Go 1.23, accessing internal symbols requires this flag as an "escape hatch" to bypass the new [package handshake requirement](https://github.com/golang/go/issues/67401).

## Requirements

- Go 1.23.1 or later

## Quick Start

### Basic Example

```go
package main

import (
	"fmt"

	"github.com/webriots/coro"
)

func main() {
	// Create a coroutine that yields values
	resume, cancel := coro.New(func(yield func(string) int, suspend func() int) string {
		// Yield a value and wait for input
		input := yield("first value")
		fmt.Printf("Received: %d\n", input)

		// Yield another value
		input = yield("second value")
		fmt.Printf("Received: %d\n", input)

		// Return final value
		return "final value"
	})
	defer cancel() // Always cancel to avoid leaks

	// Start the coroutine and get the first yielded value
	value, running := resume(0)
	fmt.Printf("Coroutine yielded: %s, still running: %t\n", value, running)
	// Output: Coroutine yielded: first value, still running: true

	// Resume the coroutine with a value and get the next yielded value
	value, running = resume(1)
	fmt.Printf("Coroutine yielded: %s, still running: %t\n", value, running)
	// Output: Coroutine yielded: second value, still running: true

	// Resume one last time to get the return value
	value, running = resume(2)
	fmt.Printf("Coroutine returned: %s, still running: %t\n", value, running)
	// Output: Coroutine returned: final value, still running: false
}
```

### Generator Example

```go
package main

import (
	"fmt"

	"github.com/webriots/coro"
)

// Create a generator that yields numbers in a sequence
func createSequence(max int) func() (int, bool) {
	resume, cancel := coro.New(func(yield func(int) struct{}, suspend func() struct{}) int {
		for i := 0; i < max; i++ {
			yield(i)
		}
		return -1 // Final return value
	})

	// Return a simple generator function
	return func() (int, bool) {
		value, running := resume(struct{}{})
		if !running {
			cancel() // Clean up resources
		}
		return value, running
	}
}

func main() {
	// Create a generator that yields numbers 0-9
	nextValue := createSequence(10)

	// Iterate through all values
	for {
		value, hasMore := nextValue()
		if !hasMore {
			break
		}
		fmt.Printf("Generated: %d\n", value)
	}
}
```

### Suspension Example

```go
package main

import (
	"fmt"

	"github.com/webriots/coro"
)

func main() {
	resume, cancel := coro.New(func(yield func(string) int, suspend func() int) string {
		// Suspend execution without yielding a value
		input := suspend()
		fmt.Printf("Resumed with: %d\n", input)

		// Yield a value and suspend
		input = yield("yielded value")
		fmt.Printf("Resumed with: %d\n", input)

		return "done"
	})
	defer cancel()

	// First resume starts the coroutine, which immediately suspends
	value, running := resume(0)
	fmt.Printf("Value: %q, Running: %t\n", value, running)
	// Output: Value: "", Running: true

	// Resume the suspended coroutine
	value, running = resume(1)
	fmt.Printf("Value: %q, Running: %t\n", value, running)
	// Output: Value: "yielded value", Running: true

	// Final resume to get the return value
	value, running = resume(2)
	fmt.Printf("Value: %q, Running: %t\n", value, running)
	// Output: Value: "done", Running: false
}
```

### Cancellation

```go
package main

import (
	"fmt"

	"github.com/webriots/coro"
)

func main() {
	resume, cancel := coro.New(func(yield func(string) int, suspend func() int) string {
		// Use defer to catch cancellation
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("Coroutine was cancelled: %v\n", r)
			}
		}()

		// Yield a value
		yield("before cancel")

		// This will never be reached if the coroutine is cancelled
		fmt.Println("This won't be executed if cancelled")
		return "completed"
	})

	// Start the coroutine
	value, running := resume(0)
	fmt.Printf("Value: %q, Running: %t\n", value, running)

	// Cancel the coroutine
	cancel()

	// Attempting to resume will panic with ErrCanceled
	// If you want to handle this, you need to wrap in a recover()
	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("Panic when resuming cancelled coroutine: %v\n", r)
			}
		}()
		resume(1)
	}()
}
```

## Detailed Usage

### Coroutine Creation

Coroutines are created using the `New` function, which returns a pair of functions: `resume` and `cancel`. The provided function to `New` is executed as a coroutine, and receives two control flow functions: `yield` and `suspend`.

```go
resume, cancel := coro.New[In, Out](fn func(yield func(Out) In, suspend func() In) Out)
```

Parameters:

| Parameter | Description |
| --------- | ----------- |
| `In` | Type parameter for values passed to the coroutine when resuming |
| `Out` | Type parameter for values yielded from the coroutine |
| `fn` | Function to execute as a coroutine |
| `yield` | Function to return a value to the caller and pause execution |
| `suspend` | Function to pause execution without returning a value |

Returns:

| Function | Description |
| -------- | ----------- |
| `resume(In) (Out, bool)` | Function to resume the coroutine with a value and get its yielded value and status |
| `cancel()` | Function to cancel the coroutine's execution |

### Resuming a Coroutine

The `resume` function is used to start and continue a coroutine's execution:

```go
value, running := resume(inputValue)
```

Parameters:

| Parameter | Description |
| --------- | ----------- |
| `inputValue` | Value of type `In` to pass to the coroutine |

Returns:

| Value | Description |
| ----- | ----------- |
| `value` | The value yielded by the coroutine (of type `Out`) or the coroutine's final return value |
| `running` | Boolean indicating whether the coroutine is still running (`true`) or has completed (`false`) |

### Yielding Values

The `yield` function allows a coroutine to return a value to the caller and pause execution until resumed:

```go
inputValue := yield(outputValue)
```

When `yield` is called:
1. The coroutine pauses execution and returns control to the caller
2. The `outputValue` is passed to the caller via the `resume` function
3. When the coroutine is resumed, `yield` returns the value passed to `resume`

### Suspending Execution

The `suspend` function pauses a coroutine without yielding a value:

```go
inputValue := suspend()
```

When `suspend` is called:
1. The coroutine pauses execution and returns control to the caller
2. No value is returned to the caller (the zero value of `Out` is returned from `resume`)
3. When the coroutine is resumed, `suspend` returns the value passed to `resume`

### Cancellation

The `cancel` function is used to terminate a coroutine early:

```go
cancel()
```

When a coroutine is canceled:
1. If the coroutine is currently suspended, resuming it will cause a panic with `ErrCanceled`
2. If the coroutine calls `yield` or `suspend` after being canceled, it will panic with `ErrCanceled`
3. The panic can be caught inside the coroutine with a deferred recover block

### Error Handling

Panics within a coroutine are captured and propagated through the `resume` function:

```go
defer func() {
    if r := recover(); r != nil {
        // Handle the panic
        fmt.Printf("Coroutine panicked: %v\n", r)
    }
}()

resume, cancel := coro.New(...)
defer cancel()

// This will panic if the coroutine panics
resume(inputValue)
```

Additionally, a special `panicError` type is used to wrap panics and provide stack trace information. You can access the stack trace using the error's methods:

```go
defer func() {
    if r := recover(); r != nil {
        if pe, ok := r.(interface{ DebugString() string }); ok {
            // Get detailed error information with stack trace
            errorInfo := pe.DebugString()
            fmt.Println(errorInfo)
        }
    }
}()
```

### Type Safety

The `New` function uses generics for type safety:

```go
// A coroutine that takes integers and yields strings
resume, cancel := coro.New[int, string](func(yield func(string) int, suspend func() int) string {
    // yield returns string and receives int
    input := yield("hello")
    return "final"
})

// Type-safe usage
value, _ := resume(42) // value is of type string
```

### Best Practices

1. **Always defer `cancel()`** to ensure proper cleanup when you're done with a coroutine:
   ```go
   resume, cancel := coro.New(...)
   defer cancel() // Prevents resource leaks
   ```

2. **Handle eventual completion** by checking the boolean return value from `resume`:
   ```go
   value, running := resume(input)
   if !running {
       // Coroutine has completed
   }
   ```

3. **Avoid letting `yield` and `suspend` escape the coroutine** function. If these functions escape and are called after the coroutine is done, they will panic.

4. **Use recover within coroutines** to handle cancellation gracefully:
   ```go
   resume, cancel := coro.New(func(yield func(string) int, suspend func() int) string {
       defer func() {
           if r := recover(); r != nil {
               // Handle cancellation or other panics
           }
       }()
       // Coroutine logic
   })
   ```

5. **Build higher-level abstractions** like generators, iterators, or state machines on top of the core coroutine functionality.

## Notes

- Always call `cancel()` when you're done with a coroutine to prevent resource leaks
- Coroutines can propagate panics through the `resume` function

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the [MIT License](LICENSE).
