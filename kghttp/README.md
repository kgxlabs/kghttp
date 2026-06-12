# kghttp

An **HTTP/1.1** server library for Go, **built from scratch** on **raw TCP sockets**. You get a small stack you can read end to end: accept connections, parse requests, serialize responses, and wire your own handlers.

## Features

- **HTTP/1.1 request parsing** — request line, headers, and `Content-Length` bodies via a streaming parser
- **Response writer** — status line, headers, fixed-length bodies, **chunked** transfer encoding, and optional **trailers**
- **`Server` API** — configure `Addr`, `Handler`, and optional `IdleConnTimeOut`, then call `ListenAndServe()` or `Serve(net.Listener)`
- **Concurrent connections** — one goroutine per accepted connection
- **Persistent connections** — a connection can serve multiple requests until either side sends `Connection: close`, the client disconnects, or the idle timeout expires
- **Listener shutdown** — `Close()` stops accepting new connections (does not wait for in-flight handlers to finish)
- **Reverse proxy (demo)** — [`examples/httpserver/main.go`](../examples/httpserver/main.go) forwards `/httpbin/*` to [httpbin.org](https://httpbin.org/) with chunked bodies and SHA-256 trailers
- **Zero runtime dependencies** — library code uses only the Go standard library (tests use `testify`)

## Architecture

```text
TCP Connection
      │
      ▼
Request Parser
      │
      ▼
   Handler
      │
      ▼
Response Writer
      ├── Content-Length
      └── Chunked + Trailers
```

## Why?

This project exists to understand HTTP on top of TCP—not to replace `net/http`. Along the way it implements:

- **Request parsing** — incremental reads, request line, headers, bodies
- **Response serialization** — status line, header blocks, body writes
- **Persistent server connections** — repeated request parsing on the same accepted socket
- **Chunked transfer encoding** — `WriteChunkedBody` / `WriteChunkedBodyDone`
- **Trailers** — trailer headers after the final chunked chunk
- **Reverse proxying** — demonstrated in the example server (stream upstream, re-encode as chunked + trailers)

## Requirements

- Go **1.23.5** or newer

## Installation

This package lives in the [`kgx`](https://github.com/Kaung-HtetKyaw/kgx) monorepo.

```bash
go get github.com/Kaung-HtetKyaw/kgx/kghttp
```

Import:

```go
import "github.com/Kaung-HtetKyaw/kgx/kghttp"
```

## Quick start

```go
package main

import (
	"fmt"
	"strconv"

	"github.com/Kaung-HtetKyaw/kgx/kghttp"
)

func main() {
	server := &kghttp.Server{
		Addr: ":8080",
		Handler: func(w *kghttp.ResponseWriter, req *kghttp.Request) {
			body := []byte(fmt.Sprintf("Hello from %s %s\n", req.RequestLine.Method, req.RequestLine.RequestTarget))

			w.Headers().Set("content-type", "text/plain")
			w.Headers().Set("content-length", strconv.Itoa(len(body)))
			w.Headers().Set("connection", "close")
			w.WriteHeaders(kghttp.StatusOK)
			w.WriteBody(body)
		},
	}

	if err := server.ListenAndServe(); err != nil {
		panic(err)
	}
	defer server.Close()

	fmt.Println("listening on :8080")
	select {} // block until you call server.Close()
}
```

Run it:

```bash
go run .
```

Then:

```bash
curl -v http://localhost:8080/
```

## API overview

| Type / function | Role |
|-----------------|------|
| `Server` | Holds `Addr`, `Handler`, optional `IdleConnTimeOut`, and the TCP listener |
| `Server.ListenAndServe()` | Listen on `Addr` and start accepting connections |
| `Server.Serve(net.Listener)` | Serve on an existing listener |
| `Server.Close()` | Stop accepting new connections |
| `Handler` | `func(w *ResponseWriter, req *Request)` |
| `Request` | Parsed request line, headers, and body |
| `ReadRequest(*kgbuf.Reader)` | Parse a request from a buffered reader (used internally by the server) |
| `ReadResponse(*kgbuf.Reader, *Request)` | Parse a fixed-length HTTP/1.1 response from a buffered reader |
| `ResponseWriter` | Build and send the HTTP response |
| `ResponseWriter.WriteHeaders(StatusCode)` | Send status line + headers |
| `ResponseWriter.WriteBody([]byte)` | Send a body after headers (fixed length) |
| `ResponseWriter.WriteChunkedBody` / `WriteChunkedBodyDone` | Chunked transfer encoding + trailers |
| `Headers` | Case-insensitive header map with `Get`, `Set`, `Remove`, `Parse` |

Supported status codes in the writer today: **200**, **400**, and **500** (see `response.go`).

## Example server

See [`examples/httpserver/main.go`](../examples/httpserver/main.go) for routing, HTML/error responses, chunked video, and the httpbin reverse proxy.

From the repo root:

```bash
go run ./examples/httpserver
```

Default port: **8000**. Demo routes:

| Path | Behavior |
|------|----------|
| `/` | 200 HTML success page |
| `/yourproblem` | 400 Bad Request |
| `/myproblem` | 500 Internal Server Error |
| `/video` | Chunked `video/mp4` with SHA-256 trailer |
| `/httpbin/*` | Proxies to `https://httpbin.org/` (chunked, with trailers) |

Stop the server with `Ctrl+C` (SIGINT / SIGTERM).

## Tests

```bash
go test ./kghttp/...
```

| Area | Covered? | Notes |
|------|----------|-------|
| Request line parsing | Yes | `request_test.go` — methods, targets, HTTP version, invalid lines |
| Header parsing (request) | Yes | Via `ReadRequest` and `headers_test.go` field-line parser |
| `Content-Length` bodies | Yes | `TestBodyParse` — full body, empty body, short body, no length |
| Server (`Serve`) | Yes | `server_test.go` — end-to-end TCP request/response over a kept-alive connection |
| Response writer | Yes | `response_test.go` — fixed-length body serialization |
| Chunked encoding | Yes | `TestWriteResponseChunkedWithTrailers` — chunk framing |
| Trailers | Yes | `TestWriteResponseChunkedWithTrailers` — trailer block after final chunk |

Tests use a `chunkReader` helper to simulate variable-size TCP reads.

## Project layout

```
kghttp/
├── server.go           # Server type, ListenAndServe, per-connection handler loop
├── server_test.go      # End-to-end ListenAndServe test
├── request.go          # HTTP/1.1 request parser (streaming)
├── request_test.go     # Request line, headers, body tests
├── response.go         # ResponseWriter, status codes, chunked + trailers
├── response_test.go    # Response writer, chunked, and trailer tests
├── headers.go          # Header map and field-line parsing
└── headers_test.go     # Standalone header field-line tests
```

## Connection lifecycle

Each accepted TCP connection is handled like this:

1. The server creates a buffered reader for the accepted connection.
2. It sets a read deadline when `IdleConnTimeOut` is greater than zero.
3. It reads one HTTP request, runs your handler, and writes the response.
4. It repeats for the next request on the same connection unless either the request or response has `Connection: close`.
5. The connection is closed when the loop exits because of `Connection: close`, client disconnect, read timeout, or a request parse error.

Handlers own the response framing. For fixed-length responses, set `Content-Length` before `WriteHeaders`; for streamed responses, set `Transfer-Encoding: chunked`, write chunks, then call `WriteChunkedBodyDone`.

## Current limitations

- **HTTP/1.1 only** — request lines must use `HTTP/1.1`; methods must be uppercase letters
- **No TLS/HTTPS** — plain TCP only
- **No HTTP/2**
- **No HTTP client** — upstream calls in the demo use Go's `net/http` client
- **Limited response parser** — `ReadResponse` supports `Content-Length` bodies; chunked, close-delimited, and status-specific no-body rules are not implemented yet
- **Limited status codes** — writer reason phrases for 200, 400, and 500 only
- **Handler-owned responses** — the library does not set `Content-Length` or pick a status for you; call `WriteHeaders` then write the body (or chunks) in order
- **No in-flight drain on shutdown** — `Close()` closes the listener; active handlers are not joined

## License

No license file is included yet. Add one before publishing if you plan to open-source or distribute the package.
