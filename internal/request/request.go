package request

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"unicode"
)

type Request struct {
	RequestLine RequestLine
}

type RequestLine struct {
	HttpVersion   string
	RequestTarget string
	Method        string
}

func RequestFromReader(reader io.Reader) (*Request, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	requestLine, err := parseRequestLine(data)
	if err != nil {
		return nil, err
	}

	return &Request{
		RequestLine: *requestLine,
	}, nil
}

func parseRequestLine(data []byte) (*RequestLine, error) {
	i := bytes.Index(data, []byte("\r\n"))
	if i == -1 {
		return nil, errors.New("Invalid input:")
	}

	line := string(data[:i])

	requestLine, err := requestLineFromString(line)
	if err != nil {
		return nil, err
	}

	return requestLine, nil
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
