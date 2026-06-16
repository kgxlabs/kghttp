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

func (cr *chunkedReader) readChunkLine() ([]byte, error) {
	bs, err := cr.r.ReadBytes("\r\n")
	return []byte{}, nil
}

func (cr *chunkedReader) parseHexUnit(line []byte) (n uint64, err error) {
	if len(line) == 0 {
		return 0, errors.New("empty hex number for chunk length")
	}
	for i, b := range line {
		switch {
		case '0' <= b && b <= '9':
			b = b - '0'
		case 'a' <= b && b <= 'f':
			b = b - 'a' + 10
		case 'A' <= b && b <= 'F':
			b = b - 'A' + 10
		default:
			return 0, errors.New("invalid byte in chunk length")
		}
		if i == 16 {
			return 0, errors.New("http chunk length too large")
		}
		n <<= 4
		n |= uint64(b)
	}
	return
}

func (cr *chunkedReader) beginChunkProcess() {
	// read chunk line
	var line []byte
	line, cr.err = cr.readChunkLine()
	cr.n, cr.err = cr.parseHexUnit(line)

	if cr.err != nil {
		return
	}

	// TODO: handle chunk extension and handle very long chunk extension
	// limit the excess bytes
	if cr.n == 0 {
		cr.err = io.EOF
	}
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
