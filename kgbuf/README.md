# kgbuf

`kgbuf` is a small buffered I/O package for the `kgx` stack. It is intentionally minimal: the goal is to provide just enough buffered reading behavior for the surrounding packages while keeping the implementation easy to read.

It is similar in spirit to parts of Go's `bufio`, but smaller and tailored to this project.

## Status

`kgbuf` is available, but still early.

| API | Status | Description |
|-----|--------|-------------|
| `Buffered()` | Implemented | Returns the number of bytes currently buffered and unread |
| `ReadString(delim string)` | Implemented | Reads through the next delimiter and returns the consumed string |
| `Read([]byte)` | Planned | Will read bytes into a caller-provided buffer |
| `ReadByte()` | Planned | Will read a single byte from the buffered reader |

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

## API overview

| Type / function | Role |
|-----------------|------|
| `Reader` | Wraps an `io.Reader` with an internal buffer |
| `NewReader(io.Reader)` | Creates a new buffered reader |
| `Reader.Buffered()` | Reports how many bytes are currently buffered |
| `Reader.ReadString(delim string)` | Reads until `delim` is found |
| `Reader.Read([]byte)` | Planned buffered read API |
| `Reader.ReadByte()` | Planned single-byte read API |

## Behavior today

- The reader keeps an internal buffer and grows it when needed.
- Consumed bytes are compacted when enough of the buffer has been read.
- `ReadString` returns the delimiter as part of the returned string.
- If the delimiter is not found before EOF, `ReadString` currently returns an empty string and no error.

## Tests

From the repository root:

```bash
go test ./kgbuf/...
```

## Current limitations

- `Read` is not implemented yet.
- `ReadByte` is not implemented yet.
- The buffer currently uses compaction; a ring buffer is noted as a future improvement in the source.
