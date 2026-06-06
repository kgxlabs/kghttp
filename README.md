# kghttp

An **HTTP/1.1** server library for Go, **built from scratch** on **raw TCP sockets**. You get a small stack you can read end to end: accept connections, parse requests, serialize responses, and wire your own handlers.

## Features

- **HTTP/1.1 request parsing** — request line, headers, and `Content-Length` bodies via a streaming parser
- **Response writer** — status line, headers, fixed-length bodies, **chunked** transfer encoding, and optional **trailers**
- **`Server` API** — configure `Addr` and `Handler`, then call `ListenAndServe()` or `Serve(net.Listener)`
- **Concurrent connections** — one goroutine per accepted connection
- **Listener shutdown** — `Close()` stops accepting new connections (does not wait for in-flight handlers to finish)
- **Reverse proxy (demo)** — [`examples/httpserver/main.go`](https://github.com/Kaung-HtetKyaw/kghttp/blob/main/examples/httpserver/main.go) forwards `/httpbin/*` to [httpbin.org](https://httpbin.org/) with chunked bodies and SHA-256 trailers
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
- **Chunked transfer encoding** — `WriteChunkedBody` / `WriteChunkedBodyDone`
- **Trailers** — trailer headers after the final chunked chunk
- **Reverse proxying** — demonstrated in the example server (stream upstream, re-encode as chunked + trailers)

## Requirements

- Go **1.23.5** or newer

## Installation

```bash
go get github.com/Kaung-HtetKyaw/kghttp
```

Import the package as `kghttp`:

```go
import kghttp "github.com/Kaung-HtetKyaw/kghttp"
```

Or clone the repo:

```bash
git clone https://github.com/Kaung-HtetKyaw/kghttp.git
cd kghttp
```

## Quick start

```go
package main

import (
	"fmt"
	"strconv"

	kghttp "github.com/Kaung-HtetKyaw/kghttp"
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
| `Server` | Holds `Addr`, `Handler`, and the TCP listener |
| `Server.ListenAndServe()` | Listen on `Addr` and start accepting connections |
| `Server.Serve(net.Listener)` | Serve on an existing listener |
| `Server.Close()` | Stop accepting new connections |
| `Handler` | `func(w *ResponseWriter, req *Request)` |
| `Request` | Parsed request line, headers, and body |
| `RequestFromReader(io.Reader)` | Parse a request from any reader (used internally by the server) |
| `ResponseWriter` | Build and send the HTTP response |
| `ResponseWriter.WriteHeaders(StatusCode)` | Send status line + headers |
| `ResponseWriter.WriteBody([]byte)` | Send a body after headers (fixed length) |
| `ResponseWriter.WriteChunkedBody` / `WriteChunkedBodyDone` | Chunked transfer encoding + trailers |
| `Headers` | Case-insensitive header map with `Get`, `Set`, `Remove`, `Parse` |

Supported status codes in the writer today: **200**, **400**, and **500** (see `response.go`).

## Example server

See [`examples/httpserver/main.go`](https://github.com/Kaung-HtetKyaw/kghttp/blob/main/examples/httpserver/main.go) for routing, HTML/error responses, chunked video, and the httpbin reverse proxy.

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
go test ./...
```

| Area | Covered? | Notes |
|------|----------|-------|
| Request line parsing | Yes | `request_test.go` — methods, targets, HTTP version, invalid lines |
| Header parsing (request) | Yes | Via `RequestFromReader` and `headers_test.go` field-line parser |
| `Content-Length` bodies | Yes | `TestBodyParse` — full body, empty body, short body, no length |
| Server (`ListenAndServe`) | Yes | `server_test.go` — end-to-end TCP request/response |
| Response writer | Yes | `response_test.go` — fixed-length body serialization |
| Chunked encoding | Yes | `TestWriteResponseChunkedWithTrailers` — chunk framing |
| Trailers | Yes | `TestWriteResponseChunkedWithTrailers` — trailer block after final chunk |

Tests use a `chunkReader` helper to simulate variable-size TCP reads.

## Project layout

```
.
├── server.go           # Server type, ListenAndServe, per-connection handler loop
├── server_test.go      # End-to-end ListenAndServe test
├── request.go          # HTTP/1.1 request parser (streaming)
├── request_test.go     # Request line, headers, body tests
├── response.go         # ResponseWriter, status codes, chunked + trailers
├── response_test.go    # Response writer, chunked, and trailer tests
├── headers.go          # Header map and field-line parsing
├── headers_test.go     # Standalone header field-line tests
└── examples/
    ├── httpserver/
    │   └── main.go     # Demo routes, proxy, chunked video
    └── assets/
        └── vim.mp4     # Sample video for /video
```

## Connection lifecycle

Each accepted TCP connection is handled like this:

1. The server reads **one** HTTP request from the connection.
2. Your handler runs and writes the response (you usually set `Connection: close`).
3. The connection is **closed** when the handler returns (`defer conn.Close()` in `server.go`).

There is **no keep-alive** support yet: the server does not read a second request on the same socket, even if the client sends `Connection: keep-alive`. Treat every connection as single-request, single-response today.

## Current limitations

- **HTTP/1.1 only** — request lines must use `HTTP/1.1`; methods must be uppercase letters
- **No TLS/HTTPS** — plain TCP only
- **No HTTP/2**
- **No HTTP client** — upstream calls in the demo use Go's `net/http` client
- **No request pipelining** — one request per connection, no queued responses on a live socket
- **No keep-alive** — connection closes after the handler returns
- **Limited status codes** — writer reason phrases for 200, 400, and 500 only
- **Handler-owned responses** — the library does not set `Content-Length` or pick a status for you; call `WriteHeaders` then write the body (or chunks) in order
- **No in-flight drain on shutdown** — `Close()` closes the listener; active handlers are not joined

## License

No license file is included yet. Add one before publishing if you plan to open-source or distribute the package.
