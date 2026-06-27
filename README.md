# kghttp

A small Go HTTP layer built from scratch.

The goal is not to replace the standard library or popular tools like `net/http`. It is to learn how HTTP pieces fit together by implementing request parsing, response writing, transfer handling, and client/server behavior directly.

## Packages

| Package | Status | Description |
|---------|--------|-------------|
| [`kghttp`](./kghttp) | **Available** | HTTP/1.1 client and server on raw TCP — request/response parsing, transfer readers, chunked bodies, trailers |
| [`kgurl`](./kgurl) | **Available** | URL parsing wrapper around Go's `net/url`, intended to grow into a local implementation |
| [`kgbuf`](./kgbuf) | **Available** | Minimal buffered I/O utilities for this stack |
| `kgroute` | **Coming Soon** | Minimal HTTP router (Chi-like API, fewer features) |

See each package's `README.md` for API details and usage.

## Examples

Runnable examples live under [`examples/`](./examples/). The current example server is documented in [`kghttp/README.md`](./kghttp/README.md#example-server).

## Requirements

- Go **1.23.5** or newer

## Getting started

Clone the repo:

```bash
git clone https://github.com/kgxlabs/kghttp.git
cd kghttp
```

Import a package from the module root:

```go
import "github.com/kgxlabs/kghttp/kghttp"
import "github.com/kgxlabs/kghttp/kgurl"
import "github.com/kgxlabs/kghttp/kgbuf"
```

Run tests for everything:

```bash
go test ./...
```

Run tests for a single package:

```bash
go test ./kghttp/...
go test ./kgbuf/...
```

## Repository layout

```
.
├── go.mod
├── kgbuf/               # Minimal buffered I/O utilities (available)
│   └── README.md
├── kghttp/              # HTTP/1.1 client/server (available)
│   └── README.md
├── kgurl/               # URL parser wrapper (available)
│   └── README.md
└── examples/            # Runnable demos
    └── httpserver/
```

## Philosophy

- **Small surface area** — only what is needed for the next layer
- **Readable over clever** — code you can step through in an afternoon
- **Composable** — each package should stand alone and plug into the others
- **Built to learn** — correctness and clarity before feature parity with production libraries

## License

No license file is included yet.
