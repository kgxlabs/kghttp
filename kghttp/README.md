# kghttp

An **HTTP/1.1** client/server library for Go, **built from scratch** on **raw TCP sockets**. You get a small stack you can read end to end: accept connections, parse requests, serialize responses, issue client requests, and wire your own handlers.

## Features

- **HTTP/1.1 request parsing** — request line, headers, and body readers for `Content-Length`, `Transfer-Encoding: chunked`, and empty bodies
- **HTTP/1.1 response parsing** — status line, headers, and the same transfer body reader used for requests
- **Response writer** — status line, headers, fixed-length bodies, **chunked** transfer encoding, and optional **trailers**
- **`Server` API** — configure `Addr`, `Handler`, and optional `IdleConnTimeOut`, then call `ListenAndServe()` or `Serve(net.Listener)`
- **`Client` / `Transport` API** — create requests with `NewRequest`, call `Client.Do`, or use `Get`, `Head`, and `Post`
- **Concurrent connections** — one goroutine per accepted connection
- **Persistent connections** — a connection can serve multiple requests until either side sends `Connection: close`, the client disconnects, or the idle timeout expires
- **Client connection reuse** — the transport keeps idle connections by scheme/host and reuses them after response bodies are fully read
- **Listener shutdown** — `Close()` stops accepting new connections (does not wait for in-flight handlers to finish)
- **Reverse proxy (demo)** — [`examples/httpserver/main.go`](../examples/httpserver/main.go) forwards `/httpbin/*` to [httpbin.org](http://httpbin.org/) with the local client, chunked bodies, and SHA-256 trailers
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

Client Request
      │
      ▼
  Transport
      │
      ▼
TCP Connection
      │
      ▼
Response Parser
```

## Why?

This project exists to understand HTTP on top of TCP—not to replace `net/http`. Along the way it implements:

- **Message parsing** — incremental reads, request/status lines, headers, and transfer-aware body readers
- **Response serialization** — status line, header blocks, body writes
- **Persistent server connections** — repeated request parsing on the same accepted socket
- **Client round trips** — request serialization, response parsing, and simple idle connection pooling
- **Chunked transfer encoding** — chunked body reads plus response writes that frame chunks and finish trailers when the handler returns
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
			body := []byte(fmt.Sprintf("Hello from %s %s\n", req.Method, req.URL.Path))

			w.Headers().Set("content-type", "text/plain")
			w.Headers().Set("content-length", strconv.Itoa(len(body)))
			w.Headers().Set("connection", "close")
			w.WriteHeaders(kghttp.StatusOK)
			w.Write(body)
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
| `Request` | Parsed method, URL, protocol version, headers, `io.ReadCloser` body, and trailers |
| `NewRequest(method, url string, body io.Reader)` | Build a client request |
| `ReadRequest(*kgbuf.Reader)` | Parse a request from a buffered reader (used internally by the server) |
| `Response` | Parsed status line, headers, `io.ReadCloser` body, and trailers |
| `ReadResponse(*kgbuf.Reader, *Request)` | Parse an HTTP/1.1 response from a buffered reader |
| `Client` | Holds a `RoundTripper` and sends requests with `Do` |
| `DefaultClient` | Package-level client using `DefaultTransport` |
| `Client.Do(*Request)` | Write one request and parse one response |
| `Get(url string)` | Convenience helper for `GET` |
| `Head(url string)` | Convenience helper for `HEAD` |
| `Post(url, contentType string, body io.Reader)` | Convenience helper for `POST` |
| `Transport` | RoundTripper implementation with TCP dialing and idle connection reuse |
| `NewTransport()` | Create an isolated transport with its own idle connection pool |
| `ResponseWriter` | Build and send the HTTP response |
| `ResponseWriter.WriteHeaders(StatusCode)` | Send status line + headers |
| `ResponseWriter.Write([]byte)` | Send body bytes after headers; auto-sends `200 OK` headers if needed |
| `ResponseWriter.Trailers()` | Mutable trailer header map for chunked responses |
| `Headers` | Case-insensitive header map with `Get`, `Set`, `Remove`, `Parse` |

Supported status codes in the writer today: **200**, **400**, and **500** (see `response.go`).

## Example server

See [`examples/httpserver/main.go`](../examples/httpserver/main.go) for routing, HTML/error responses, chunked video, and the httpbin reverse proxy. Handlers write body bytes with `ResponseWriter.Write`; the server finalizes the selected transfer writer after the handler returns.

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
| `/httpbin/*` | Proxies to `http://httpbin.org/` through `kghttp.Get` (chunked, with trailers) |

Stop the server with `Ctrl+C` (SIGINT / SIGTERM).

## Tests

```bash
go test ./kghttp/...
```

| Area | Covered? | Notes |
|------|----------|-------|
| Request line parsing | Yes | `request_test.go` — method, URL, HTTP version fields, invalid lines |
| Header parsing (request) | Yes | Via `ReadRequest` and `headers_test.go` field-line parser |
| `Content-Length` bodies | Yes | `TestBodyParse` — full body, empty body, short body, no length |
| Transfer body reader | Yes | `transfer_test.go` — `Content-Length`, empty, chunked, and chunked trailers |
| Server (`Serve`) | Yes | `server_test.go` — end-to-end request/response through the local client, plus malformed raw requests |
| Client helpers | Yes | `client_test.go` — `Get`, `Post`, fixed-length and chunked request bodies |
| Transport | Yes | `transport_test.go` — round trips, request serialization, chunked bodies, idle connections, malformed responses |
| Response writer | Yes | `response_test.go` — fixed-length body serialization |
| Response parser | Yes | `response_test.go` — status line, headers, fixed-length body, multiple responses |
| Chunked encoding | Yes | `TestWriteResponseChunkedWithTrailers` — chunk framing |
| Trailers | Yes | `TestWriteResponseChunkedWithTrailers` and `transfer_test.go` — trailer block after final chunk |

Tests use a `chunkReader` helper to simulate variable-size TCP reads.

## Project layout

```
kghttp/
├── client.go           # Client type plus Get, Head, and Post helpers
├── client_test.go      # Client helper tests
├── transport.go        # RoundTripper implementation and idle connection pool
├── transport_test.go   # Transport round-trip tests
├── persist_conn.go     # Per-connection request write / response read helpers
├── roundtrip.go        # RoundTripper interface
├── server.go           # Server type, ListenAndServe, per-connection handler loop
├── server_test.go      # End-to-end server tests
├── request.go          # HTTP/1.1 request parser (streaming)
├── request_test.go     # Request line, headers, body tests
├── response.go         # Response parser, ResponseWriter, status codes
├── response_test.go    # Response writer, chunked, and trailer tests
├── transfer.go         # Shared request/response body reader selection
├── transfer_test.go    # Content-Length, chunked, and trailer body reader tests
├── fixed.go            # Fixed-length response body writer
├── chunked.go          # Chunked transfer body reader/writer
├── fixed_test.go       # Fixed-length writer tests
├── chunked_test.go     # Chunked reader/writer tests
├── internal/
│   └── nobody.go       # Empty body reader/writer
├── headers.go          # Header map and field-line parsing
└── headers_test.go     # Standalone header field-line tests
```

## Connection lifecycle

```text
accept TCP connection
        |
        v
create buffered reader
        |
        v
read one HTTP request
        |
        v
run handler
        |
        v
finalize response body writer
        |
        v
keep connection alive?
   +----+----+
  yes        no
   |         |
   +-> next request
             |
             v
          close TCP connection
```

The server repeats this loop for persistent HTTP/1.1 connections. If
`IdleConnTimeOut` is greater than zero, each read waits only until that
deadline.

| Event | Result |
|-------|--------|
| Request has `Connection: close` | Close after the response |
| Response has `Connection: close` | Close after the response |
| Client disconnects | Exit the loop and close |
| Read timeout | Exit the loop and close |
| Request parse error | Exit the loop and close |
| No close signal and no error | Read the next request on the same connection |

### Handler responsibilities

Handlers choose the response framing before the body is written.

| Response style | Required headers | Body behavior |
|----------------|------------------|---------------|
| Fixed length | `Content-Length` | Write exactly that many bytes |
| Streamed | `Transfer-Encoding: chunked` | Write one or more chunks; optional trailers are sent during finalization |
| Empty | Neither header | Body writes are ignored |

Fixed-length response:

```go
body := []byte("hello\n")

w.Headers().Set("content-length", strconv.Itoa(len(body)))
w.WriteHeaders(kghttp.StatusOK)
w.Write(body)
```

Chunked response with a trailer:

```go
w.Headers().Set("transfer-encoding", "chunked")
w.WriteHeaders(kghttp.StatusOK)
w.Write([]byte("first chunk\n"))
w.Trailers().Set("x-finished", "true")
```

If `Write` is called before `WriteHeaders`, the writer sends `200 OK` first.

### Body readers

Parsed request and response bodies are exposed as `io.ReadCloser`.

| Headers | Reader behavior |
|---------|-----------------|
| `Transfer-Encoding: chunked` | Decode chunks and parse trailers after the terminating `0` chunk |
| `Content-Length` | Read exactly that many bytes; return `io.ErrUnexpectedEOF` if the stream ends early |
| Neither header | Empty body |

## Client lifecycle

```text
NewRequest
    |
    v
prepare headers and body
    |
    v
Client.Do
    |
    v
Transport gets idle connection or dials TCP
    |
    v
write request
    |
    v
read response
    |
    v
response body read to EOF?
   +----+----+
  yes        no/error
   |         |
   v         v
reuse if     close connection
allowed
```

`NewRequest` parses the URL, initializes headers, and wraps non-closer bodies
with `io.NopCloser`.

| Request body | Transfer headers |
|--------------|------------------|
| `nil` | No body headers added by `NewRequest` |
| `*bytes.Reader`, `*bytes.Buffer`, `*strings.Reader` | `Content-Length` can be inferred |
| Other non-nil `io.Reader` | Sent with `Transfer-Encoding: chunked` |

`Client.Do` delegates to its transport. The default transport uses plain TCP,
serializes one request, parses one response, and then waits for the response
body to finish before deciding what to do with the connection.

| Event | Connection behavior |
|-------|---------------------|
| Response body is read to EOF | Return to the idle pool |
| Request has `Connection: close` | Close instead of reusing |
| Response has `Connection: close` | Close instead of reusing |
| Body reading ends with an error | Close instead of reusing |

## Current limitations

- **HTTP/1.1 only** — request lines must use `HTTP/1.1`; methods must be uppercase letters
- **No TLS/HTTPS** — plain TCP only
- **No HTTP/2**
- **Plain HTTP client only** — no TLS, redirects, proxies, cookies, compression, or context cancellation yet
- **Limited transfer handling** — close-delimited bodies, status/method-specific no-body rules, and chunk extensions are not implemented yet
- **Limited status codes** — writer reason phrases for 200, 400, and 500 only
- **Handler-owned responses** — the library does not infer `Content-Length` or `Transfer-Encoding`; set response framing headers before `WriteHeaders`, then write the body in order
- **No in-flight drain on shutdown** — `Close()` closes the listener; active handlers are not joined

## License

No license file is included yet. Add one before publishing if you plan to open-source or distribute the package.
