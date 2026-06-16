package kghttp

import (
	"bytes"
	"errors"
	"io"

	"github.com/Kaung-HtetKyaw/kgx/kgbuf"
)

var (
	ErrMalformedChunkedEncoding = errors.New("kghttp: malformed chunked encoding")
)

type chunkedReader struct {
	r          *kgbuf.Reader
	n          int
	chunkEnded bool
	err        error
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

func (cr *chunkedReader) beginChunkProcess() {
	// TODO: come back here after these issues https://github.com/Kaung-HtetKyaw/kgx/issues/21 , https://github.com/Kaung-HtetKyaw/kgx/issues/22
}

func (cr *chunkedReader) chunkHeaderAvailable() bool {
	n := cr.r.Buffered()

	if n > 0 {
		peek, _ := cr.r.Peek(n)
		return bytes.Index(peek, []byte("\r\n")) > 0
	}

	return false
}

func (cr *chunkedReader) Read(p []byte) (n int, err error) {

	for cr.err == nil {

		if cr.n == 0 {
			// do no wait for more data if next chunk header is not immediately available
			if n > 0 && !cr.chunkHeaderAvailable() {
				break
			}

			// begin processing chunks headers here
			cr.beginChunkProcess()

			// there could be malfomred chunked encoding so restart loop to recheck error
			continue
		}

		if len(p) == 0 {
			break
		}

		rbuf := p
		if len(p) > cr.n {
			rbuf = rbuf[:cr.n]
		}

		var bn int
		bn, cr.err = cr.r.Read(rbuf)
		n += bn
		cr.n -= bn
		p = p[bn:]

		if cr.n == 0 && cr.err == nil {
			cr.chunkEnded = true
		} else if err == io.EOF {
			cr.err = io.ErrUnexpectedEOF
		}
	}

	return n, cr.err
}
