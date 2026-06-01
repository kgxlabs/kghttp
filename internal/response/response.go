package response

import (
	"bytes"
	"fmt"
	"go-http-server/internal/headers"
	"log"
	"strconv"
)

type StatusCode int

const (
	StatusOK                  StatusCode = 200
	StatusBadRequest          StatusCode = 400
	StatusInternalServerError StatusCode = 500
)

type Writer struct {
	Headers     headers.Headers
	StatusCode  StatusCode
	body        bytes.Buffer
	response    bytes.Buffer
	writerState writerState
}

type writerState string

const (
	writerStateWritingStatusLine writerState = "writingStatusLine"
	writerStateHeaders           writerState = "writingHeaders"
	writerStateBody              writerState = "writingBody"
)

func (w *Writer) ResponseBytes() []byte {
	w.Headers.Set("Content-Length", strconv.Itoa(w.body.Len()))

	if err := w.serializeStatusLine(); err != nil {
		log.Printf("failed to serialize status line: %v", err)
		return []byte{}
	}

	if err := w.serializeHeaders(); err != nil {
		log.Printf("failed to serialize headers: %v", err)
		return []byte{}
	}

	if _, err := w.response.Write(w.body.Bytes()); err != nil {
		log.Printf("failed to serialize body: %v", err)
		return []byte{}
	}

	return w.response.Bytes()
}

func (w *Writer) WriteStatusLine(statusCode StatusCode) {
	w.StatusCode = statusCode
}

func (w *Writer) serializeStatusLine() error {
	if _, err := w.response.Write(getStatusLine(w.StatusCode)); err != nil {
		return err
	}
	return nil

}

func (w *Writer) serializeHeaders() error {
	headersStr := ""
	for key, value := range w.Headers {
		headersStr = headersStr + fmt.Sprintf("%s: %s\r\n", key, value)

	}

	headersStr = headersStr + "\r\n"
	if _, err := w.response.Write([]byte(headersStr)); err != nil {
		return err
	}
	return nil
}

func (w *Writer) WriteBody(data []byte) (int, error) {
	return w.body.Write(data)
}

func getStatusLine(s StatusCode) []byte {
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
