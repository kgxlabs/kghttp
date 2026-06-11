# kgbuf

`kgbuf` is a small buffered I/O package for the `kgx` stack. It is intentionally minimal: the goal is to provide just enough buffered reading behavior for the surrounding packages while keeping the implementation easy to read.

It is similar in spirit to parts of Go's `bufio`, but smaller and tailored to this project.

## Status

`kgbuf` is available, but still early.

| API | Status | Description |
|-----|--------|-------------|
| `Read([]byte)` | Implemented | Reads bytes into a caller-provided buffer |
| `ReadBytes(delim []byte)` | Implemented | Reads through the next delimiter and returns the consumed bytes |
| `ReadString(delim string)` | Implemented | Reads through the next delimiter and returns the consumed string |
| `Peek(n int)` | Implemented | Reads the next `n` bytes into the internal buffer without consuming them |
| `Buffered()` | Implemented | Returns the number of bytes currently buffered and unread |
| `Size()` | Implemented | Returns the current internal buffer capacity |
| `Reset(io.Reader)` | Implemented | Reuses the reader with a new underlying `io.Reader` |

## Usage

```go
package main

import (
	"fmt"
	"strings"

	"github.com/Kaung-HtetKyaw/kgx/kgbuf"
)

func main() {
	r := kgbuf.NewReader(strings.NewReader("hello\nworld\n"))

	line, err := r.ReadString("\n")
	if err != nil {
		panic(err)
	}

	fmt.Print(line)
}
```

Read bytes into a caller-provided buffer:

```go
package main

import (
	"fmt"
	"strings"

	"github.com/Kaung-HtetKyaw/kgx/kgbuf"
)

func main() {
	r := kgbuf.NewReader(strings.NewReader("hello world"))
	p := make([]byte, 5)

	n, err := r.Read(p)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%d %q\n", n, p[:n])
}
```

Read bytes through a delimiter:

```go
package main

import (
	"fmt"
	"strings"

	"github.com/Kaung-HtetKyaw/kgx/kgbuf"
)

func main() {
	r := kgbuf.NewReader(strings.NewReader("hello\nworld\n"))

	line, err := r.ReadBytes([]byte("\n"))
	if err != nil {
		panic(err)
	}

	fmt.Printf("%q\n", line)
}
```

Peek ahead without consuming bytes:

```go
package main

import (
	"fmt"
	"strings"

	"github.com/Kaung-HtetKyaw/kgx/kgbuf"
)

func main() {
	r := kgbuf.NewReader(strings.NewReader("hello world"))

	b, err := r.Peek(5)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%q buffered=%d\n", b, r.Buffered())
}
```

## API overview

| Type / function | Role |
|-----------------|------|
| `Reader` | Wraps an `io.Reader` with an internal buffer |
| `NewReader(io.Reader)` | Creates a new buffered reader |
| `NewReaderSize(io.Reader, int)` | Creates a new buffered reader with a custom internal buffer size |
| `Reader.Read([]byte)` | Reads up to the caller-provided buffer size |
| `Reader.ReadBytes(delim []byte)` | Reads until `delim` is found and includes `delim` in the returned bytes |
| `Reader.ReadString(delim string)` | Reads until `delim` is found and includes `delim` in the returned string |
| `Reader.Peek(n int)` | Buffers and returns the next `n` bytes without advancing the read cursor |
| `Reader.Buffered()` | Reports how many bytes are currently buffered |
| `Reader.Size()` | Reports the current buffer capacity |
| `Reader.Reset(io.Reader)` | Clears buffered state and switches to a new underlying reader |

## Behavior today

- The reader keeps an internal buffer and grows it when needed.
- Consumed bytes are compacted when enough of the buffer has been read.
- `Read` fills the provided byte slice from buffered data and the underlying reader.
- `Read` returns the number of bytes copied into the provided slice.
- `Read` returns `0, nil` when no more bytes are available.
- `ReadBytes` and `ReadString` return the delimiter as part of the returned value.
- If the delimiter is not found before EOF, `ReadBytes` returns an empty slice and no error.
- If the delimiter is not found before EOF, `ReadString` currently returns an empty string and no error.
- `Peek(0)` returns an empty slice and no error.
- `Peek(n)` stores the returned bytes in the buffer without consuming them.
- `Peek(n)` returns `ErrPartialRead` when fewer than `n` bytes are available.
- `Reset` clears buffered data and read/write cursor state.

## Tests

From the repository root:

```bash
go test ./kgbuf/...
```

## Current limitations

- The buffer currently uses compaction; a ring buffer is noted as a future improvement in the source.
- `ReadBytes` and `ReadString` return an empty result instead of partial data when the delimiter is not found before EOF.
