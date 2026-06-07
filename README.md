# kgx

A personal Go monorepo for building a **minimal stack from scratch**—small, readable packages you can understand end to end, without leaning on large frameworks.

The goal is not to replace the standard library or popular tools like `net/http` and Chi. It is to learn how the pieces fit together by implementing them yourself, then composing them into something usable.

## Packages

| Package | Status | Description |
|---------|--------|-------------|
| [`kghttp`](./kghttp) | **Available** | HTTP/1.1 server on raw TCP — request parsing, response serialization, chunked bodies, trailers |
| `kgbuf` | **Coming Soon** | Minimal buffered I/O (like `bufio`, but smaller and tailored to this stack) |
| `kgroute` | **Coming Soon** | Minimal HTTP router (Chi-like API, fewer features) |
| `kgcache` | **Coming Soon** | Small in-memory cache |
| `kgdb` | **Coming Soon** | Minimal embedded database |

Package names may change as the stack grows. Each subdirectory is its own Go package under one module. See each package's `README.md` for API details and usage.

## Examples

Runnable examples live under [`examples/`](./examples/). Each example has its own README with setup and run instructions.

## Requirements

- Go **1.23.5** or newer

## Getting started

Clone the repo:

```bash
git clone https://github.com/Kaung-HtetKyaw/kgx.git
cd kgx
```

Import a package from the module root:

```go
import "github.com/Kaung-HtetKyaw/kgx/kghttp"
```

Run tests for everything:

```bash
go test ./...
```

Run tests for a single package:

```bash
go test ./kghttp/...
```

## Repository layout

```
.
├── go.mod
├── kghttp/              # HTTP/1.1 server (available)
│   └── README.md
└── examples/            # Runnable demos (each with its own README)
    └── httpserver/
```

Coming soon packages (`kgbuf`, `kgroute`, `kgcache`, `kgdb`, …) will appear as sibling directories under the module root.

## Philosophy

- **Small surface area** — only what is needed for the next layer
- **Readable over clever** — code you can step through in an afternoon
- **Composable** — each package should stand alone and plug into the others
- **Built to learn** — correctness and clarity before feature parity with production libraries

## License

No license file is included yet.
