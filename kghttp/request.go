package kghttp

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/Kaung-HtetKyaw/kgx/kgbuf"
)

type Request struct {
	RequestLine    RequestLine
	Headers        Headers
	Body           []byte
	bodyLengthRead int
	state          RequestState
}

type RequestState string

const (
	RequestStateInitialized    RequestState = "initialized"
	RequestStateDone           RequestState = "done"
	RequestStateParsingHeaders RequestState = "parsingHeaders"
	RequestStateParsingBody    RequestState = "parsingBody"
)

type RequestLine struct {
	HttpVersion   string
	RequestTarget string
	Method        string
}

const (
	CRLF = "\r\n"
)

func ReadRequest(reader *kgbuf.Reader) (*Request, error) {
	request := &Request{
		Headers: NewHeaders(),
		state:   RequestStateInitialized,
	}

	line, err := reader.ReadBytes([]byte(CRLF))
	if err != nil {
		return nil, err
	}
	if len(line) == 0 {
		return nil, fmt.Errorf("incomplete http request at state: %s", request.state)
	}

	requestLine, _, err := parseRequestLine(line)
	if err != nil {
		return nil, err
	}
	request.RequestLine = *requestLine
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

	contentLengthStr, ok := request.Headers.Get("content-length")
	if !ok {
		request.state = RequestStateDone
		return request, nil
	}

	contentLen, err := strconv.Atoi(contentLengthStr)
	if err != nil {
		return nil, fmt.Errorf("malformed content length: %s", err)
	}

	request.Body = make([]byte, contentLen)
	// TODO: reaplce full body read into memory with ReadCloser
	n, err := reader.ReadFull(request.Body)
	if err != nil {
		return nil, err
	}
	request.bodyLengthRead = n
	request.state = RequestStateDone

	return request, nil
}

func parseRequestLine(data []byte) (*RequestLine, int, error) {
	i := bytes.Index(data, []byte(CRLF))
	if i == -1 {
		return nil, 0, nil
	}

	line := string(data[:i])

	requestLine, err := requestLineFromString(line)
	if err != nil {
		return nil, 0, err
	}

	// i+2 because after reading request line, there will be \r\n so we skip them and read next line
	return requestLine, i + 2, nil
}

func requestLineFromString(str string) (*RequestLine, error) {
	parts := strings.Fields(str)

	if len(parts) != 3 {
		return nil, errors.New("Invalid request")
	}

	target := parts[1]
	method := parts[0]
	if !validateRequestMethod(method) {
		return nil, errors.New("Invalid method")
	}

	version, err := getHTTPVersion(parts[2])
	if err != nil {
		return nil, err
	}

	return &RequestLine{
		RequestTarget: target,
		Method:        method,
		HttpVersion:   version,
	}, nil
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
