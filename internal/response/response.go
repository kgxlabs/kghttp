package response

import (
	"bytes"
	"fmt"
	"go-http-server/internal/headers"
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
	headers     headers.Headers
	writerState writerState
}

type writerState string

const (
	writerStateWritingHeaders writerState = "writingHeaders"
	writerStateWritingBody    writerState = "writingBody"
)

func NewWriter(writer io.Writer) *Writer {
	return &Writer{
		writer:      writer,
		writerState: writerStateWritingHeaders,
	}
}

func (w *Writer) Headers() headers.Headers {
	if w.headers == nil {
		w.headers = make(headers.Headers)
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

func serializeHeaders(headers headers.Headers) ([]byte, error) {
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
