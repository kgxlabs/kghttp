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

var (
	ErrMalformedStatusLine = errors.New("kghttp: err malformed status line")
)

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
		return nil, ErrMalformedStatusLine
	}

	version, err := getHTTPVersion(parts[0])
	if err != nil {
		return nil, err
	}

	code, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, ErrMalformedStatusLine
	}

	if code < 100 || code > 999 {
		return nil, ErrMalformedStatusLine
	}

	return &StatusLine{
		HttpVersion:  version,
		StatusCode:   GetStatusCode(code),
		ReasonPhrase: strings.Join(parts[2:], " "),
	}, nil
}

type ResponseWriter struct {
	w           io.WriteCloser
	bw          io.WriteCloser
	headers     Headers
	trailers    Headers
	writerState writerState
}

func NewWriter(w io.WriteCloser) *ResponseWriter {
	return &ResponseWriter{
		w:           w,
		writerState: writerStateWritingHeaders,
	}
}

func (rw *ResponseWriter) Headers() Headers {
	if rw.headers == nil {
		rw.headers = make(Headers)
	}
	return rw.headers
}

func (rw *ResponseWriter) WriteHeaders(statusCode StatusCode) error {
	if rw.writerState != writerStateWritingHeaders {
		return fmt.Errorf("cannot write headers in state: %s", rw.writerState)
	}

	statusLine := serializeRespStatusLine(statusCode)

	if _, err := rw.w.Write(statusLine); err != nil {
		return err
	}

	hs, err := serializeHeaders(rw.headers)
	if err != nil {
		return err
	}

	if _, err := rw.w.Write(hs); err != nil {
		return err
	}

	rw.writerState = writerStateWritingBody

	cfg := writeTransferCfg{
		writer: kgbuf.NewWriter(rw.w),
		headers: func() Headers {
			return rw.Headers()
		},
		trailers: func() Headers {
			return rw.Trailers()
		},
	}
	tw, err := writeTransfer(cfg)
	if err != nil {
		return err
	}

	rw.bw = tw

	return nil
}

func (rw *ResponseWriter) Write(p []byte) (int, error) {
	if rw.writerState == writerStateWritingHeaders {
		if err := rw.WriteHeaders(StatusOK); err != nil {
			return 0, err
		}

		rw.writerState = writerStateWritingBody
	}

	return rw.bw.Write(p)
}

func (rw *ResponseWriter) Trailers() Headers {
	if rw.trailers == nil {
		rw.trailers = make(Headers)
	}

	return rw.trailers
}

func (rw *ResponseWriter) finish() error {
	if rw.bw == nil {
		return fmt.Errorf("cannot finish response in state: %s", rw.writerState)
	}
	return rw.bw.Close()
}

func serializeRespStatusLine(s StatusCode) []byte {
	return fmt.Appendf(nil, "HTTP/1.1 %d %s\r\n", s, getReasonPhrase(s))
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
		return "", ErrMalformedHttpVersion
	}

	parts := strings.Split(proto, "/")

	if parts[0] != "HTTP" {
		return "", ErrMalformedHttpVersion
	}

	if parts[1] != "1.1" {
		return "", ErrMalformedHttpVersion
	}

	return parts[1], nil
}
