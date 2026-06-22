# kgbuf

`kgbuf` is a small buffered I/O package for the `kgx` stack. It is intentionally minimal: the goal is to provide just enough buffered reading and writing behavior for the surrounding packages while keeping the implementation easy to read.

It is similar in spirit to parts of Go's `bufio`, but smaller and tailored to this project.

## Status

`kgbuf` is available, but still early.

| API | Status | Description |
|-----|--------|-------------|
| `Read([]byte)` | Implemented | Reads bytes into a caller-provided buffer |
| `ReadFull([]byte)` | Implemented | Reads until the caller-provided buffer is filled or EOF is reached |
| `ReadBytes(delim []byte)` | Implemented | Reads through the next delimiter and returns the consumed bytes |
| `ReadBytesLimit(delim []byte, limit int)` | Implemented | Reads through the next delimiter while enforcing a byte budget |
| `ReadSlice(delim byte)` | Implemented | Reads through the next delimiter byte and returns a slice backed by the internal buffer |
| `ReadString(delim string)` | Implemented | Reads through the next delimiter and returns the consumed string |
| `ReadStringLimit(delim string, limit int)` | Implemented | Reads through the next delimiter string while enforcing a byte budget |
| `Peek(n int)` | Implemented | Returns the next `n` unread bytes without consuming them |
| `Buffered()` | Implemented | Returns the number of bytes currently buffered and unread |
| `Size()` | Implemented | Returns the current internal buffer capacity |
| `Reset(io.Reader)` | Implemented | Reuses the reader with a new underlying `io.Reader` |
| `Write([]byte)` | Implemented | Writes bytes through the writer buffer |
| `WriteString(string)` | Implemented | Writes a string through the writer buffer |
| `Flush()` | Implemented | Writes buffered data to the underlying writer |
| `Available()` | Implemented | Returns the remaining writer buffer capacity |

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

Read until a caller-provided buffer is full:

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

	n, err := r.ReadFull(p)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%d %q\n", n, p)
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

Write bytes through a buffered writer:

```go
package main

import (
	"bytes"
	"fmt"

	"github.com/Kaung-HtetKyaw/kgx/kgbuf"
)

func main() {
	var out bytes.Buffer
	w := kgbuf.NewWriter(&out)

	if _, err := w.WriteString("hello world"); err != nil {
		panic(err)
	}
	if err := w.Flush(); err != nil {
		panic(err)
	}

	fmt.Println(out.String())
}
```

## API overview

| Type / function | Role |
|-----------------|------|
| `Reader` | Wraps an `io.Reader` with an internal buffer |
| `NewReader(io.Reader)` | Creates a new buffered reader |
| `NewReaderSize(io.Reader, int)` | Creates a new buffered reader with a custom internal buffer size |
| `Reader.Read([]byte)` | Reads up to the caller-provided buffer size |
| `Reader.ReadFull([]byte)` | Reads until the caller-provided buffer is filled, returning `io.ErrUnexpectedEOF` with partial data if EOF arrives first |
| `Reader.ReadBytes(delim []byte)` | Reads until `delim` is found and includes `delim` in the returned bytes |
| `Reader.ReadBytesLimit(delim []byte, limit int)` | Reads until `delim` is found or returns `ErrByteReadLimitReached` when the limit is exhausted |
| `Reader.ReadSlice(delim byte)` | Reads until `delim` is found and returns a slice backed by the internal buffer |
| `Reader.ReadString(delim string)` | Reads until `delim` is found and includes `delim` in the returned string |
| `Reader.ReadStringLimit(delim string, limit int)` | String wrapper around `ReadBytesLimit` |
| `Reader.Peek(n int)` | Returns the next `n` unread bytes without advancing the read cursor |
| `Reader.Buffered()` | Reports how many bytes are currently buffered |
| `Reader.Size()` | Reports the current buffer capacity |
| `Reader.Reset(io.Reader)` | Clears buffered state and switches to a new underlying reader |
| `Writer` | Wraps an `io.Writer` with an internal buffer |
| `NewWriter(io.Writer)` | Creates a new buffered writer |
| `NewWriterSize(io.Writer, int)` | Creates a new buffered writer with a custom internal buffer size |
| `Writer.Write([]byte)` | Buffers data or writes directly when the input is larger than the available buffer |
| `Writer.WriteString(string)` | String wrapper around `Write` |
| `Writer.Flush()` | Writes buffered data to the underlying writer |
| `Writer.Buffered()` | Reports how many bytes are currently buffered for writing |
| `Writer.Available()` | Reports remaining space in the writer buffer |

## Behavior today

- The reader keeps an internal buffer and grows it when needed.
- Consumed bytes are compacted when enough of the buffer has been read.
- `Read` fills the provided byte slice from buffered data and the underlying reader.
- `Read` returns the number of bytes copied into the provided slice.
- `Read` may return `io.EOF` when no more bytes are available.
- `ReadFull` fills the provided byte slice before returning successfully.
- `ReadFull` returns partial data and `io.ErrUnexpectedEOF` if EOF arrives before the slice is full.
- `ReadBytes` and `ReadString` return the delimiter as part of the returned value.
- If the delimiter is not found before EOF, `ReadBytes` returns the partial data with `io.EOF`.
- If the delimiter is not found before EOF, `ReadString` returns the partial string with `io.EOF`.
- `ReadSlice` returns a buffer-backed slice when it finds the delimiter.
- Callers should treat `ReadSlice` results as short-lived because later reads may overwrite the internal buffer.
- If `ReadSlice` cannot find the delimiter before the buffer fills, it returns `ErrBufferFull`.
- `ReadBytesLimit` and `ReadStringLimit` return `ErrByteReadLimitReached` when the delimiter is not found before the byte budget is exhausted.
- `Peek(0)` returns an empty slice and no error.
- `Peek(n)` returns already-buffered unread bytes first and reads only enough additional bytes to satisfy `n`.
- `Peek(n)` does not advance the read cursor.
- `Peek(n)` returns `ErrPartialRead` with the available unread bytes when fewer than `n` bytes are available.
- `Reset` clears buffered data and read/write cursor state.
- The writer keeps data in an internal buffer until the buffer fills, a large write bypasses the buffer, or `Flush` is called.
- `Write` returns the number of bytes accepted from the provided slice.
- `WriteString` writes string data through the same path as `Write`.
- `Flush` is a no-op when the writer buffer is empty.
- `Buffered` and `Available` report writer buffer usage.

## Tests

From the repository root:

```bash
go test ./kgbuf/...
```

## Current limitations

- The buffer currently uses compaction; a ring buffer is noted as a future improvement in the source.
