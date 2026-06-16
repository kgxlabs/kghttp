package kghttp

import (
	"io"

	"github.com/Kaung-HtetKyaw/kgx/kgbuf"
)

type bodyReader struct {
	// We are going to create new reader based on request headers
	src io.Reader
	r   *kgbuf.Reader
	// ref to a message either *Request or *Response
	msg     any
	sawEOF  bool
	closing bool
}

func readTransfer(msg any, r *kgbuf.Reader) error {
	return nil
}

func (br *bodyReader) Read(p []byte) (int, error) {
	n, err := br.src.Read(p)
	return n, err
}

func (br *bodyReader) Close() error {
	return nil
}

func (br *bodyReader) readTrailers() error {
	return nil
}
