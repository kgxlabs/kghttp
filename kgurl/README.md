# kgurl

`kgurl` is the URL package for `kgx`.

For now, it is a thin wrapper around Go's standard [`net/url`](https://pkg.go.dev/net/url) package. The goal is to let the rest of the stack depend on `kgurl.URL` and `kgurl.Parse` now, while leaving room to replace the internals with a local implementation later.

## API

```go
u, err := kgurl.Parse("/coffee")
if err != nil {
	panic(err)
}

fmt.Println(u.Path)
```

## Current status

- `URL` embeds `*url.URL` from `net/url`
- `Parse` delegates directly to `url.Parse`
- No additional URL parsing logic is implemented yet
