package kghttp

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"

	"github.com/Kaung-HtetKyaw/kgx/kgbuf"
	"github.com/Kaung-HtetKyaw/kgx/kgurl"
)

type Request struct {
	Method         string
	URL            *kgurl.URL
	Proto          string
	ProtoMajor     int
	ProtoMinor     int
	Headers        Headers
	Body           io.ReadCloser
	Trailers       Headers
	ContentLength  int
	bodyLengthRead int
	state          RequestState
}

var (
	ErrBadRequest           = errors.New("bad request")
	ErrInvalidHttpMethod    = errors.New("kghttp:err invalid http method")
	ErrMalformedHttpVersion = errors.New("kghttp:err malformed http version")
	ErrMalformedHeaders     = errors.New("kghttp:err malformed http headers")
)

type RequestState string

const (
	RequestStateInitialized    RequestState = "initialized"
	RequestStateDone           RequestState = "done"
	RequestStateParsingHeaders RequestState = "parsingHeaders"
	RequestStateParsingBody    RequestState = "parsingBody"
)

const (
	CRLF             = "\r\n"
	RequestLineLimit = 8192
)

func NewRequest(method string, url string, body io.Reader) (*Request, error) {
	if method == "" {
		method = "GET"
	}

	if !validateRequestMethod(method) {
		return nil, ErrInvalidHttpMethod
	}

	u, err := kgurl.Parse(url)
	if err != nil {
		return nil, err
	}

	rc, ok := body.(io.ReadCloser)
	if !ok && body != nil {
		rc = io.NopCloser(body)
	}

	// NOTE: Sometime url parser can produce "example.com:"
	u.Host = strings.TrimSuffix(u.Host, ":")

	req := &Request{
		Method:     method,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Headers:    make(Headers),
		Body:       rc,
		URL:        u,
	}

	if body != nil {
		switch v := body.(type) {
		case *bytes.Reader:
			req.ContentLength = v.Len()
		case *bytes.Buffer:
			req.ContentLength = v.Len()
		case *strings.Reader:
			req.ContentLength = v.Len()
		default:
			// NOTE: This is for content length unknown
			req.ContentLength = -1
		}
	}

	req.Headers.Set("host", u.Host)

	return req, nil
}

func ReadRequest(reader *kgbuf.Reader) (*Request, error) {
	request := &Request{
		Headers: NewHeaders(),
		state:   RequestStateInitialized,
	}

	line, err := reader.ReadBytesLimit([]byte(CRLF), RequestLineLimit)
	if err != nil {
		return nil, err
	}
	if len(line) == 0 {
		return nil, fmt.Errorf("incomplete http request at state: %s", request.state)
	}

	if _, err := request.parseRequestLine(line); err != nil {
		return nil, err
	}
	request.state = RequestStateParsingHeaders

	for request.state == RequestStateParsingHeaders {
		line, err = reader.ReadBytes([]byte(CRLF))
		if err != nil {
			return nil, err
		}
		if len(line) == 0 {
			return nil, fmt.Errorf("incomplete http request at state: %s", request.state)
		}

		_, done, err := request.Headers.Parse(line)
		if err != nil {
			return nil, err
		}
		if done {
			request.state = RequestStateParsingBody
		}
	}

	if err := readTransfer(request, reader); err != nil {
		return nil, err
	}
	if _, ok := request.Headers.Get("host"); !ok {
		return nil, fmt.Errorf("%w: missing host header", ErrBadRequest)
	}
	request.state = RequestStateDone

	return request, nil
}

func (r *Request) parseRequestLine(data []byte) (int, error) {
	i := bytes.Index(data, []byte(CRLF))
	if i == -1 {
		return 0, nil
	}

	line := string(data[:i])

	parts := strings.Fields(line)

	if len(parts) != 3 {
		return 0, ErrMalformedHeaders
	}

	target := parts[1]
	method := parts[0]
	if !validateRequestMethod(method) {
		return 0, ErrInvalidHttpMethod
	}

	u, err := kgurl.Parse(target)
	if err != nil {
		return 0, err
	}

	proto, protoMajor, protoMinor, err := parseHTTPVersion(parts[2])
	if err != nil {
		return 0, err
	}

	r.Method = method
	r.URL = u
	r.Proto = proto
	r.ProtoMajor = protoMajor
	r.ProtoMinor = protoMinor

	// i+2 because after reading request line, there will be \r\n so we skip them and read next line
	return i + 2, nil
}

func (r *Request) HTTPVersion() string {
	if r.Proto != "" {
		return r.Proto
	}
	return fmt.Sprintf("HTTP/%d.%d", r.ProtoMajor, r.ProtoMinor)
}

func validateRequestMethod(method string) bool {
	if method == "" {
		return false
	}

	for _, str := range method {
		if !unicode.IsUpper(str) {
			return false
		}
	}

	return true
}

func validateHTTPVersion(proto string) bool {
	parts := strings.Split(proto, "/")
	if len(parts) != 2 {
		return false
	}

	if parts[0] != "HTTP" {
		return false
	}

	if parts[1] != "1.1" {
		return false
	}

	return true
}

func parseHTTPVersion(proto string) (string, int, int, error) {
	if !validateHTTPVersion(proto) {
		return "", 0, 0, ErrMalformedHttpVersion
	}

	parts := strings.Split(proto, "/")
	version := strings.Split(parts[1], ".")
	if len(version) != 2 {
		return "", 0, 0, ErrMalformedHttpVersion
	}

	major, err := strconv.Atoi(version[0])
	if err != nil {
		return "", 0, 0, ErrMalformedHttpVersion
	}
	minor, err := strconv.Atoi(version[1])
	if err != nil {
		return "", 0, 0, ErrMalformedHttpVersion
	}

	return proto, major, minor, nil
}

func serizlizeReqStatusLine(req *Request) []byte {
	path := req.URL.Path
	if path == "" {
		path = "/"
	}
	return fmt.Appendf(nil, "%s %s %s\r\n", req.Method, path, req.HTTPVersion())
}
