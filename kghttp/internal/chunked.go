package internal

import (
	"bytes"
	"errors"
	"io"

	"github.com/Kaung-HtetKyaw/kgx/kgbuf"
)

var (
	ErrMalformedChunkedEncoding = errors.New("kghttp: malformed chunked encoding")
	ErrLineTooLong              = errors.New("kghttp: chunk line too long")
)

const maxLineLength = 4096

type chunkedReader struct {
	r          *kgbuf.Reader
	n          int
	chunkEnded bool
	err        error
	buf        [2]byte
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

func readChunkLine(r *kgbuf.Reader) ([]byte, error) {
	bs, err := r.ReadSlice('\n')

	if err != nil {
		// NOTE: The idea is if a chunk is EOF, it should give 0\r\n not EOF
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}

		if err == kgbuf.ErrBufferFull {
			err = ErrLineTooLong
		}

		return nil, err
	}

	if idx := bytes.IndexByte(bs, '\r'); idx == -1 {
		return nil, errors.New("kghttp: error chunk line ends with bare RL")
	} else if idx != len(bs)-2 {
		return nil, errors.New("kghttp: error invalid CR")
	}

	// Trim CRLF
	bs = bs[:len(bs)-2]

	if len(bs) >= maxLineLength {
		return nil, ErrLineTooLong
	}

	return bs, nil
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
	line, cr.err = readChunkLine(cr.r)
	if cr.err != nil {
		return
	}

	var n uint64
	n, cr.err = cr.parseHexUnit(line)
	cr.n = int(n)

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
		return bytes.IndexByte(peek, '\n') >= 0
	}

	return false
}

func (cr *chunkedReader) Read(p []byte) (n int, err error) {
	for cr.err == nil {
		if cr.chunkEnded {
			// If there is no trailing CRLF available yet.
			// Return early instead of keep waiting and blocking next read
			if n > 0 && cr.r.Buffered() < 2 {
				break
			}

			// If there are more than 2 bytes available, validate CRLF
			if _, cr.err = io.ReadFull(cr.r, cr.buf[:2]); cr.err == nil {
				if string(cr.buf[:]) != "\r\n" {
					cr.err = ErrMalformedChunkedEncoding
					break
				}

			} else {
				if cr.err == io.EOF {
					cr.err = io.ErrUnexpectedEOF
				}
				break
			}

			// If there are more than 2 bytes available and CRLF is valid, treat it as the start of next chunk.
			cr.chunkEnded = false
		}
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
		if len(rbuf) > cr.n {
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

	if n > 0 && cr.err == io.EOF {
		return n, nil
	}

	return n, cr.err
}
