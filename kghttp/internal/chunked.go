package kghttp

import (
	"errors"
	"io"

	"github.com/Kaung-HtetKyaw/kgx/kgbuf"
)

var (
	ErrMalformedChunkedEncoding = errors.New("kghttp: malformed chunked encoding")
)

type chunkedReader struct {
	r io.Reader
}

func NewChunkedReader(r io.Reader) io.Reader {
	br, ok := r.(*kgbuf.Reader)
	if !ok {
		br = kgbuf.NewReader(r)
	}

	return &chunkedReader{
		r: br,
	}
}

func (cr *chunkedReader) Read(p []byte) (int, error) {
	return 0, nil
}
