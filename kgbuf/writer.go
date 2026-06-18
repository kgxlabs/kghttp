package kgbuf

import "io"

type Writer struct {
	// Underlying lying writer that will directly write to data stream
	writer io.Writer
	size   int
	buf    []byte
	n      int
}

const writerDefaultBufferSize = 4096

func NewWriter(w io.Writer) *Writer {
	return &Writer{
		writer: w,
		size:   writerDefaultBufferSize,
		buf:    make([]byte, writerDefaultBufferSize),
	}
}

func NewWriterSize(w io.Writer, size int) *Writer {
	return &Writer{
		writer: w,
		size:   size,
		buf:    make([]byte, size),
	}
}

func (w *Writer) Write(p []byte) (n int, err error) {
	for len(p) > w.Available() && err == nil {
		var nn int
		if w.Buffered() == 0 {
			nn, err = w.writer.Write(p)
		} else {
			nn = copy(w.buf[w.n:], p[n:])
			err = w.Flush()
		}

		n += nn
		p = p[n:]
	}

	if err != nil {
		return n, err
	}

	nn := copy(w.buf[w.n:], p)
	n += nn
	w.n += nn
	return n, err
}

func (w *Writer) Flush() error {
	if w.n == 0 {
		return nil
	}

	n, err := w.writer.Write(w.buf)

	if n > 0 {
		w.n -= n
	}

	return err
}

func (w *Writer) Buffered() int {
	return w.n
}

func (w *Writer) Available() int {
	return w.size - w.n
}
