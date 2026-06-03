package kghttp

import (
	"bytes"
	"fmt"
	"io"
)

type StatusCode int

const (
	StatusOK                  StatusCode = 200
	StatusBadRequest          StatusCode = 400
	StatusInternalServerError StatusCode = 500
)

type Writer struct {
	writer      io.Writer
	headers     Headers
	trailers    Headers
	writerState writerState
}

type writerState string

const (
	writerStateWritingHeaders  writerState = "writingHeaders"
	writerStateWritingBody     writerState = "writingBody"
	writerStateWritingTrailers writerState = "writingTrailers"
)

func NewWriter(writer io.Writer) *Writer {
	return &Writer{
		writer:      writer,
		writerState: writerStateWritingHeaders,
	}
}

func (w *Writer) Headers() Headers {
	if w.headers == nil {
		w.headers = make(Headers)
	}
	return w.headers
}

func (w *Writer) WriteHeaders(statusCode StatusCode) error {
	if w.writerState != writerStateWritingHeaders {
		return fmt.Errorf("cannot write headers in state: %s", w.writerState)
	}

	statusLine := serializeStatusLine(statusCode)

	if _, err := w.writer.Write(statusLine); err != nil {
		return err
	}

	hs, err := serializeHeaders(w.headers)
	if err != nil {
		return err
	}

	if _, err := w.writer.Write(hs); err != nil {
		return err
	}

	w.writerState = writerStateWritingBody

	return nil
}

func (w *Writer) WriteBody(data []byte) (int, error) {
	if w.writerState != writerStateWritingBody {
		return 0, fmt.Errorf("cannot write body at state: %s", w.writerState)
	}

	return w.writer.Write(data)
}

func (w *Writer) WriteChunkedBody(data []byte) (int, error) {
	if w.writerState != writerStateWritingBody {
		return 0, fmt.Errorf("cannot write body in state: %s", w.writerState)
	}

	dataLen := len(data)
	hexLen := []byte(fmt.Sprintf("%x\r\n", dataLen))
	totalSize := len(hexLen) + dataLen + 2

	chunkedData := make([]byte, 0, totalSize)
	chunkedData = append(chunkedData, hexLen...)
	if dataLen > 0 {
		chunkedData = append(chunkedData, data...)
		chunkedData = append(chunkedData, []byte("\r\n")...)
	}

	n, err := w.writer.Write(chunkedData)
	return n, err
}

func (w *Writer) WriteChunkedBodyDone() (int, error) {
	n, err := w.WriteChunkedBody([]byte{})
	if err != nil {
		return 0, err
	}

	if err = w.WriteTrailers(); err != nil {
		return 0, err
	}

	return n, nil
}

func (w *Writer) Trailers() Headers {
	if w.trailers == nil {
		w.trailers = make(Headers)
	}

	return w.trailers
}

func (w *Writer) WriteTrailers() error {
	ts, err := serializeHeaders(w.trailers)
	if err != nil {
		return err
	}

	if _, err := w.writer.Write(ts); err != nil {
		return err
	}

	return nil
}

func serializeHeaders(headers Headers) ([]byte, error) {
	var buf bytes.Buffer
	for key, value := range headers {
		if _, err := buf.Write([]byte(fmt.Sprintf("%s: %s\r\n", key, value))); err != nil {
			return []byte{}, err
		}
	}

	if _, err := buf.Write([]byte("\r\n")); err != nil {
		return []byte{}, err
	}

	return buf.Bytes(), nil
}

func serializeStatusLine(s StatusCode) []byte {
	return []byte(fmt.Sprintf("HTTP/1.1 %d %s\r\n", s, getReasonPhrase(s)))
}

func GetStatusCode(s int) StatusCode {
	switch s {
	case 200:
		return StatusOK
	case 400:
		return StatusBadRequest
	case 500:
		return StatusInternalServerError
	default:
		return StatusOK
	}
}

func getReasonPhrase(s StatusCode) string {
	switch s {
	case 200:
		return "OK"
	case 400:
		return "Bad Request"
	case 500:
		return "Internal Server Error"
	default:
		return ""
	}
}
