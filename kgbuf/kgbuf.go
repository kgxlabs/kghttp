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
	buf    []byte
	reader io.Reader
	r      int
	w      int
}

const bufferSize = 4029

var (
	ErrReaderFailedToRead = errors.New("kgbuf: reader failed to read")
)

func NewReader(reader io.Reader) *Reader {
	return &Reader{
		buf:    make([]byte, bufferSize),
		reader: reader,
	}
}

func (b *Reader) Buffered() int {
	return b.w - b.r
}

func (b *Reader) ReadString(delim string) (string, error) {
	delimIndex := -1
	for {
		// compact if more than half the bytes is consumed , if not grow
		if b.r > b.w/2 {
			copy(b.buf, b.buf[b.r:])
			b.w -= b.r
			b.r = 0
		}

		if b.w >= len(b.buf) {
			newBuf := make([]byte, len(b.buf)*2)
			copy(newBuf, b.buf)
			b.buf = newBuf
		}

		numBytesWrite := 0
		// read from underlying reader and write to internal if data we have is not enough
		n, err := b.reader.Read(b.buf[b.w:])
		if n > 0 {
			numBytesWrite = n
		}

		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return "", ErrReaderFailedToRead
		}

		b.w += numBytesWrite
		i := bytes.Index(b.buf[b.r:], []byte(delim))

		if i == -1 {
			delimIndex = i
			continue
		}

		b.r += i + len([]byte(delim))
		delimIndex = i
		break
	}

	if delimIndex == -1 {
		return "", nil
	}

	line := b.buf[:b.r]

	return string(line), nil
}
