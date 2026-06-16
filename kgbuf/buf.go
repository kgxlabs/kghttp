package kgbuf

import (
	"bytes"
	"errors"
	"io"
)

// if the r is significantly smaller than w, it is not worth to copy large chunks of bytes to free up small chunks of bytes
// For example, bufferSize=4029, r=2 and w=4029, we have to copy 4027 bytes just to free up 2 bytes. Since we compact the buffer after number of consumed bytes is more than half of the buffer size
// In addition to that, we no longer need to copy bytes since we are gonna overwrite consumed bytes
// TODO: Replace compact buffer with circular(ring) buffer
type Reader struct {
	buf         []byte
	reader      io.Reader
	r           int
	w           int
	defaultSize int
}

const bufferSize = 4096

var (
	ErrReaderFailedToRead   = errors.New("kgbuf: reader failed to read")
	ErrPartialRead          = errors.New("kgbuf: partial read")
	ErrByteReadLimitReached = errors.New("kgbuf: byte read limit reached")
	ErrBufferFull           = errors.New("kgbuf: buffer full")
)

func NewReader(reader io.Reader) *Reader {
	return &Reader{
		buf:         make([]byte, bufferSize),
		reader:      reader,
		defaultSize: bufferSize,
	}
}

func NewReaderSize(reader io.Reader, size int) *Reader {
	return &Reader{
		buf:         make([]byte, size),
		reader:      reader,
		defaultSize: size,
	}
}

func (b *Reader) Read(p []byte) (int, error) {
	if b.w-b.r > 0 {
		endR := min(b.w-b.r, len(p)) + b.r
		nc := copy(p, b.buf[b.r:endR])
		copy(b.buf, b.buf[endR:])
		b.w -= endR
		b.r = 0

		return nc, nil
	}

	// grow if threshold is reached
	if b.w >= len(b.buf)*8/10 {
		newBuf := make([]byte, 2*len(b.buf))
		copy(newBuf, b.buf)
		b.buf = newBuf
	}

	n, err := b.reader.Read(b.buf[b.w:])
	if n > 0 {
		b.w += n
	}

	if err != nil && !errors.Is(err, io.EOF) {
		return 0, err
	}

	endR := min(b.w-b.r, len(p)) + b.r
	nc := copy(p, b.buf[b.r:endR])
	copy(b.buf, b.buf[endR:])
	b.w -= endR
	b.r = 0

	return nc, err
}

func (b *Reader) ReadFull(p []byte) (int, error) {
	// number of unconsumed bytes are still below writable bytes of input
	for b.w-b.r < len(p) {
		// grow if capacity is reached
		if b.w >= cap(b.buf) {
			newBuf := make([]byte, cap(b.buf)*2)
			copy(newBuf, b.buf)
			b.buf = newBuf
		}

		n, err := b.reader.Read(b.buf[b.w:])

		if n > 0 {
			b.w += n
		}

		if err != nil {
			if errors.Is(err, io.EOF) && b.w-b.r < len(p) {
				endR := min(b.w-b.r, len(p)) + b.r
				n := copy(p, b.buf[b.r:endR])
				copy(b.buf, b.buf[endR:])
				b.w -= endR
				b.r = 0

				return n, io.ErrUnexpectedEOF
			}

			return 0, err
		}
	}

	// b.w-b.r => consumed bytes, if that is bigger than cap, only fill what cap can hold. If not fill everything
	// And then +b.r to move read cursor
	// Then, compact it
	endR := min(b.w-b.r, len(p)) + b.r
	n := copy(p, b.buf[b.r:endR])

	// TODO: Decide whether or not we should compact
	copy(b.buf, b.buf[endR:])
	b.w -= endR
	b.r = 0

	return n, nil
}

func (b *Reader) ReadBytes(delim []byte) ([]byte, error) {
	start := b.r
	for {
		if b.r > b.w/2 {
			copy(b.buf, b.buf[b.r:b.w])
			b.w -= b.r
			b.r = 0
			start = b.r
		}

		i := bytes.Index(b.buf[b.r:b.w], delim)
		if i != -1 {
			n := i + len(delim)
			b.r += n
			p := make([]byte, n)
			copy(p, b.buf[start:b.r])
			return p, nil
		}

		if b.w >= len(b.buf) {
			newBuf := make([]byte, len(b.buf)*2)
			copy(newBuf, b.buf)
			b.buf = newBuf
		}

		// read from underlying reader and write to internal if data we have is not enough
		n, err := b.reader.Read(b.buf[b.w:])
		if n > 0 {
			b.w += n
		}

		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return []byte{}, err
		}
	}

	return []byte{}, nil
}

func (b *Reader) ReadSlice(delim byte) ([]byte, error) {
	start := b.r
	for {
		if b.r > b.w/2 {
			copy(b.buf, b.buf[b.r:b.w])
			b.w -= b.r
			b.r = 0
			start = b.r
		}

		i := bytes.IndexByte(b.buf[b.r:b.w], delim)
		if i != -1 {
			n := i + 1
			b.r += n
			return b.buf[start:b.r], nil
		}

		if b.w == len(b.buf) {
			return b.buf, ErrBufferFull
		}

		// read from underlying reader and write to internal if data we have is not enough
		n, err := b.reader.Read(b.buf[b.w:])
		if n > 0 {
			b.w += min(len(b.buf), n)
		}

		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return []byte{}, err
		}
	}

	return []byte{}, nil

}

func (b *Reader) ReadBytesLimit(delim []byte, limit int) ([]byte, error) {
	budget := limit

	start := b.r
	for {

		if budget == 0 {
			return []byte{}, ErrByteReadLimitReached
		}

		if b.r > b.w/2 {
			copy(b.buf, b.buf[b.r:b.w])
			b.w -= b.r
			b.r = 0
			start = b.r
		}

		i := bytes.Index(b.buf[b.r:b.w], delim)
		if i != -1 {
			b.r += i + len(delim)
			return b.buf[start:b.r], nil
		}

		if b.w >= len(b.buf) {
			newBuf := make([]byte, len(b.buf)*2)
			copy(newBuf, b.buf)
			b.buf = newBuf
		}

		// read from underlying reader and write to internal if data we have is not enough
		n, err := b.reader.Read(b.buf[b.w:])
		if n > 0 {
			b.w += n
			budget -= min(n, budget)
		}

		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return []byte{}, err
		}
	}

	return []byte{}, nil

}

func (b *Reader) ReadString(delim string) (string, error) {
	line, err := b.ReadBytes([]byte(delim))
	return string(line), err
}

func (b *Reader) ReadStringLimit(delim string, limit int) (string, error) {
	line, err := b.ReadBytesLimit([]byte(delim), limit)
	return string(line), err
}

func (b *Reader) Peek(n int) ([]byte, error) {
	prevW := b.w
	if n == 0 {
		return []byte{}, nil
	}

	nw, err := b.reader.Read(b.buf[b.w : b.w+n])
	if nw > 0 {
		b.w += nw
	}

	if nw != n {
		return b.buf[prevW:b.w], ErrPartialRead
	}

	if err != nil {
		if errors.Is(err, io.EOF) && nw != n {
			return b.buf[prevW:b.w], ErrPartialRead
		}
		return b.buf[prevW:b.w], ErrReaderFailedToRead
	}

	return b.buf[prevW:b.w], nil
}

func (b *Reader) Buffered() int {
	return b.w - b.r
}

func (b *Reader) Size() int {
	return cap(b.buf)
}

func (b *Reader) Reset(r io.Reader) {
	b.reader = r
	b.r = 0
	b.w = 0
}
