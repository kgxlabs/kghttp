package kghttp

import (
	"errors"
	"strconv"

	"github.com/Kaung-HtetKyaw/kgx/kgbuf"
)

type fixedWriter struct {
	w       *kgbuf.Writer
	headers func() Headers
	size    int
	n       int
	started bool
	ended   bool
}

var (
	ErrContentLengthMismatch = errors.New("kghttp: err content length mismatched")
	ErrContentLengthExceeded = errors.New("kghttp: err content length exceeded")
	ErrPartialBody           = errors.New("kghttp: err partial body")
	ErrFixedWriterClosed     = errors.New("kghttp: err fixed writer closed")
)

func NewFixedWriter(w *kgbuf.Writer, headers func() Headers) *fixedWriter {
	return &fixedWriter{
		w:       w,
		headers: headers,
	}
}

func (fw *fixedWriter) Write(p []byte) (int, error) {
	if !fw.started {
		fw.started = true
		contentLen, err := fw.contentLen()

		if err != nil {
			return 0, err
		}
		fw.size = contentLen
	}

	if fw.ended {
		return 0, ErrFixedWriterClosed
	}

	if fw.available() == 0 || len(p) > fw.available() {
		return 0, ErrContentLengthExceeded
	}

	n, err := fw.w.Write(p)

	if n > 0 {
		fw.n += n
	}

	if err != nil {
		return n, err
	}

	return n, nil
}

func (fw *fixedWriter) Close() error {
	if fw.ended {
		return ErrFixedWriterClosed
	}

	if fw.available() > 0 {
		return ErrPartialBody
	}

	if err := fw.Flush(); err != nil {
		return err
	}

	fw.ended = true
	return nil
}

func (fw *fixedWriter) Flush() error {
	if fw.ended {
		return ErrFixedWriterClosed
	}

	return fw.w.Flush()
}

func (fw *fixedWriter) available() int {
	return fw.size - fw.n
}

func (fw *fixedWriter) contentLen() (int, error) {
	if fw.headers == nil {
		return 0, ErrContentLengthMismatch
	}

	hs := fw.headers()

	contentLenStr, ok := hs.Get("content-length")
	if !ok {
		return 0, errors.New("malformed content len value")
	}

	return strconv.Atoi(contentLenStr)
}
