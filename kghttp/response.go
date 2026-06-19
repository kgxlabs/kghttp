package kghttp

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/Kaung-HtetKyaw/kgx/kgbuf"
)

type StatusCode int

const (
	StatusOK                  StatusCode = 200
	StatusBadRequest          StatusCode = 400
	StatusInternalServerError StatusCode = 500
)

type ResponseWriter struct {
	writer      io.Writer
	headers     Headers
	trailers    Headers
	writerState writerState
}

type Response struct {
	StatusLine StatusLine
	Headers    Headers
	Body       io.ReadCloser
	Trailers   Headers
	state      ResponseState
}

type StatusLine struct {
	HttpVersion  string
	StatusCode   StatusCode
	ReasonPhrase string
}

type ResponseState string

const (
	ResponseStateInitialized    ResponseState = "initialized"
	ResponseStateDone           ResponseState = "done"
	ResponseStateParsingHeaders ResponseState = "parsingHeaders"
	ResponseStateParsingBody    ResponseState = "parsingBody"
)

type writerState string

const (
	writerStateWritingHeaders  writerState = "writingHeaders"
	writerStateWritingBody     writerState = "writingBody"
	writerStateWritingTrailers writerState = "writingTrailers"
)

func NewWriter(writer io.Writer) *ResponseWriter {
	return &ResponseWriter{
		writer:      writer,
		writerState: writerStateWritingHeaders,
	}
}

func ReadResponse(reader *kgbuf.Reader, req *Request) (*Response, error) {
	response := &Response{
		Headers: NewHeaders(),
		state:   ResponseStateInitialized,
	}
	_ = req

	line, err := reader.ReadBytes([]byte(CRLF))
	if err != nil {
		return nil, err
	}
	if len(line) == 0 {
		return nil, fmt.Errorf("incomplete http response at state: %s", response.state)
	}

	statusLine, _, err := parseStatusLine(line)
	if err != nil {
		return nil, err
	}
	response.StatusLine = *statusLine
	response.state = ResponseStateParsingHeaders

	for response.state == ResponseStateParsingHeaders {
		line, err = reader.ReadBytes([]byte(CRLF))
		if err != nil {
			return nil, err
		}
		if len(line) == 0 {
			return nil, fmt.Errorf("incomplete http response at state: %s", response.state)
		}

		_, done, err := response.Headers.Parse(line)
		if err != nil {
			return nil, err
		}
		if done {
			response.state = ResponseStateParsingBody
		}

	}

	if err := readTransfer(response, reader); err != nil {
		return nil, err
	}
	response.state = ResponseStateDone

	return response, nil
}

func parseStatusLine(data []byte) (*StatusLine, int, error) {
	i := bytes.Index(data, []byte(CRLF))
	if i == -1 {
		return nil, 0, nil
	}

	line := string(data[:i])

	statusLine, err := statusLineFromString(line)
	if err != nil {
		return nil, 0, err
	}

	return statusLine, i + 2, nil
}

func statusLineFromString(str string) (*StatusLine, error) {
	parts := strings.Fields(str)
	if len(parts) < 2 {
		return nil, errors.New("Invalid response")
	}

	version, err := getHTTPVersion(parts[0])
	if err != nil {
		return nil, err
	}

	code, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid status code: %s", err)
	}

	if code < 100 || code > 999 {
		return nil, fmt.Errorf("invalid status code: %d", code)
	}

	return &StatusLine{
		HttpVersion:  version,
		StatusCode:   GetStatusCode(code),
		ReasonPhrase: strings.Join(parts[2:], " "),
	}, nil
}

func (w *ResponseWriter) Headers() Headers {
	if w.headers == nil {
		w.headers = make(Headers)
	}
	return w.headers
}

func (w *ResponseWriter) WriteHeaders(statusCode StatusCode) error {
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

func (w *ResponseWriter) WriteBody(data []byte) (int, error) {
	if w.writerState != writerStateWritingBody {
		return 0, fmt.Errorf("cannot write body at state: %s", w.writerState)
	}

	return w.writer.Write(data)
}

func (w *ResponseWriter) WriteChunkedBody(data []byte) (int, error) {
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

func (w *ResponseWriter) WriteChunkedBodyDone() (int, error) {
	n, err := w.WriteChunkedBody([]byte{})
	if err != nil {
		return 0, err
	}

	if err = w.writeTrailers(); err != nil {
		return 0, err
	}

	return n, nil
}

func (w *ResponseWriter) Trailers() Headers {
	if w.trailers == nil {
		w.trailers = make(Headers)
	}

	return w.trailers
}

func (w *ResponseWriter) writeTrailers() error {
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

func getHTTPVersion(proto string) (string, error) {
	if !validateHTTPVersion(proto) {
		return "", errors.New("Invalid HTTP Version")
	}

	parts := strings.Split(proto, "/")

	if parts[0] != "HTTP" {
		return "", errors.New("Invalid HTTP Version")
	}

	if parts[1] != "1.1" {
		return "", errors.New("Invalid HTTP Version")
	}

	return parts[1], nil
}
