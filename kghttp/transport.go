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
	msg := ""
	// TODO: Figure out how to sniff out invalid Request
	if !isValidHttpMethod(req.RequestLine.Method) {
		return "", ErrInvalidHttpMethod
	}

	msg = req.RequestLine.Method + " " + req.RequestLine.RequestTarget + " " + req.RequestLine.HttpVersion + "\r\n"

	if req.Headers != nil {
		for name, value := range req.Headers {
			msg = msg + fmt.Sprintf("%s: %s\r\n", name, value)
		}
	}

	msg = msg + "\r\n"

	return msg, nil
}

func isValidHttpMethod(method string) bool {
	switch HttpMethod(method) {
	case MethodGet, MethodPost, MethodPut, MethodPatch, MethodDelete, MethodOptions, MethodTrace, MethodConnect, MethodHead:
		return true
	default:
		return false
	}
}
