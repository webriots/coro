```go
package main

import (
	"fmt"

	"github.com/webriots/coro"
)

func main() {
	resume, cancel := coro.New(func(yield func(string) int, suspend func() int) string {
		yield("hello")
		yield("world")
		return "!"
	})
	defer cancel()

	for i := 0; i < 4; i++ {
		s, ok := resume(0)
		fmt.Printf("%q %v\n", s, ok)
	}
}
```

```sh
$ go run hello.go
"hello" true
"world" true
"!" false
"" false
$
```

[Playground](https://go.dev/play/p/jhp-WqZVaOT)
