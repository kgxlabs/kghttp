package kghttp

import (
	"errors"
	"fmt"
)

type Transport struct{}

type HttpMethod string

const (
	MethodGet     HttpMethod = "GET"
	MethodPost    HttpMethod = "POST"
	MethodPut     HttpMethod = "PUT"
	MethodPatch   HttpMethod = "PATCH"
	MethodDelete  HttpMethod = "DELETE"
	MethodOptions HttpMethod = "OPTIONS"
	MethodTrace   HttpMethod = "TRACE"
	MethodConnect HttpMethod = "CONNECT"
	MethodHead    HttpMethod = "HEAD"
)

var (
	ErrInvalidHttpMethod  = errors.New("kghttp: err invalid http method")
	ErrInvalidHttpRequest = errors.New("kghttp: err invalid http request")
)

func (t *Transport) RoundTrip(req *Request) (*Response, error) {
	return nil, nil
}

func serializeRequest(req *Request) (string, error) {
	message := ""
	// TODO: Figure out how to sniff out invalid Request
	if !isValidHttpMethod(req.RequestLine.Method) {
		return "", ErrInvalidHttpMethod
	}

	message = req.RequestLine.Method + " " + req.RequestLine.RequestTarget + " " + req.RequestLine.HttpVersion + "\r\n"

	// NOTE: there should always be headers when there is a body. Even for chunked data, there should be Transfer: chunked-encoding
	// if there is none, err
	if len(req.Headers) == 0 && len(req.Body) > 0 {
		return "", ErrInvalidHttpRequest
	}

	if req.Headers != nil {
		for name, value := range req.Headers {
			message = message + fmt.Sprintf("%s: %s\r\n", name, value)
		}
	}

	return "", nil
}

func isValidHttpMethod(method string) bool {
	switch HttpMethod(method) {
	case MethodGet, MethodPost, MethodPut, MethodPatch, MethodDelete, MethodOptions, MethodTrace, MethodConnect, MethodHead:
		return true
	default:
		return false
	}
}
