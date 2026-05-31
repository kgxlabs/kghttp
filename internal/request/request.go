package request

import (
	"bytes"
	"errors"
	"fmt"
	"go-http-server/internal/headers"
	"io"
	"strconv"
	"strings"
	"unicode"
)

type Request struct {
	RequestLine    RequestLine
	Headers        headers.Headers
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
	crlf       = "\r\n"
	bufferSize = 8
)

func RequestFromReader(reader io.Reader) (*Request, error) {
	request := &Request{
		Headers: headers.NewHeaders(),
		state:   RequestStateInitialized,
	}
	buf := make([]byte, bufferSize, bufferSize)
	readToIndex := 0

	for request.state != RequestStateDone {

		// increase the buffer if cursor is beyond the current buffer size
		if readToIndex >= len(buf) {
			newBuf := make([]byte, len(buf)*2)
			copy(newBuf, buf)
			buf = newBuf
		}

		numBytesRead, err := reader.Read(buf[readToIndex:])
		if err != nil {
			// Is it because we reach to an end
			if errors.Is(err, io.EOF) {
				if request.state != RequestStateDone {
					return nil, fmt.Errorf("incomplete http request at state: %s, bytes read: %d", request.state, numBytesRead)
				}

				// we reached EOF and also is in done state, so get out of execution
				break
			}
			return nil, err
		}

		readToIndex += numBytesRead

		numBytesParsed, err := request.parse(buf[:readToIndex])
		if err != nil {
			return nil, err
		}

		// Exclude already parsed bytes
		copy(buf, buf[numBytesParsed:])
		readToIndex -= numBytesParsed
	}

	return request, nil
}

func (r *Request) parse(data []byte) (int, error) {
	totalBytesParsed := 0

	for r.state != RequestStateDone {
		n, err := r.parseSingle(data[totalBytesParsed:])
		if err != nil {
			return 0, err
		}

		totalBytesParsed += n

		if n == 0 {
			break
		}

	}

	return totalBytesParsed, nil
}

func (r *Request) parseSingle(data []byte) (int, error) {
	switch r.state {
	case RequestStateInitialized:
		requestLine, n, err := parseRequestLine(data)
		if err != nil {
			return 0, err
		}

		if n == 0 {
			return 0, nil
		}

		r.state = RequestStateParsingHeaders
		r.RequestLine = *requestLine

		return n, nil
	case RequestStateParsingHeaders:
		n, done, err := r.Headers.Parse(data)
		if err != nil {
			return 0, err
		}

		if done {
			r.state = RequestStateParsingBody
		}

		return n, nil
	case RequestStateParsingBody:
		contentLengthStr, ok := r.Headers.Get("content-length")
		if !ok {
			r.state = RequestStateDone
			return len(data), nil
		}

		contentLen, err := strconv.Atoi(contentLengthStr)
		if err != nil {
			return 0, fmt.Errorf("malformed content length: %s", err)
		}

		// This is different from the assignment solution (They append the bytes (both invalid and valid) and then return with err)
		// This is better solution.This is how parsers for keep alive connection usually works
		remaining := contentLen - r.bodyLengthRead
		consume := min(len(data), remaining)

		r.Body = append(r.Body, data[:consume]...)
		r.bodyLengthRead += consume

		if r.bodyLengthRead == contentLen {
			r.state = RequestStateDone
		}
		return len(data), nil
	case RequestStateDone:
		return 0, fmt.Errorf("trying to read in done state: %s", r.state)
	default:
		return 0, fmt.Errorf("invalid request state: %s", r.state)
	}
}

func parseRequestLine(data []byte) (*RequestLine, int, error) {
	i := bytes.Index(data, []byte(crlf))
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
