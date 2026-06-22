package kghttp

import (
	"errors"
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/Kaung-HtetKyaw/kgx/kgbuf"
)

type Transport struct {
	mu   sync.Mutex
	idle map[string][]*persistConn
}

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

func NewTransport() *Transport {
	return &Transport{
		idle: make(map[string][]*persistConn),
	}
}

func (t *Transport) getConn(key string, req *Request) (*persistConn, error) {
	t.mu.Lock()
	if t.idle == nil {
		t.idle = make(map[string][]*persistConn)
	}
	list, ok := t.idle[key]
	if !ok {
		t.idle[key] = []*persistConn{}
	}

	if len(list) > 0 {
		pconn := list[len(list)-1]
		t.idle[key] = list[:len(list)-1]

		t.mu.Unlock()
		return pconn, nil
	}
	t.mu.Unlock()

	port := req.URL.Port()
	if port == "" {
		switch req.URL.Scheme {
		case "https":
			port = "443"
		case "http":
			port = "80"
		}
	}

	conn, err := net.Dial("tcp", net.JoinHostPort(req.URL.Hostname(), port))
	if err != nil {
		return nil, err
	}

	pconn := &persistConn{
		bw:   *kgbuf.NewWriter(conn),
		br:   *kgbuf.NewReader(conn),
		conn: conn,
	}

	return pconn, nil
}

func (t *Transport) putConnBackToIdle(key string, pc *persistConn) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.idle == nil {
		return errors.New("no connection pool")
	}

	if _, ok := t.idle[key]; !ok {
		return errors.New("connection pool not initialized")
	}

	t.idle[key] = append(t.idle[key], pc)

	return nil
}

func (t *Transport) RoundTrip(req *Request) (*Response, error) {
	key := canonicalKey(req)
	pc, err := t.getConn(key, req)
	if err != nil {
		return nil, err
	}

	if err := pc.writeRequest(req); err != nil {
		pc.conn.Close()
		return nil, err
	}

	resp, err := pc.readResponse(req)
	if err != nil {
		pc.conn.Close()
		return nil, err
	}

	resp.Body = &bodyReader{
		src:    resp.Body,
		onDone: t.onBodyDone(req, resp, key, pc),
	}

	return resp, nil
}

func (t *Transport) onBodyDone(req *Request, resp *Response, key string, pc *persistConn) func(error) {
	return func(err error) {
		if err != nil && err != io.EOF {
			pc.conn.Close()
			return
		}

		ch := connectionHeader(req, resp)
		if ch == "close" {
			pc.conn.Close()
			return
		}

		t.putConnBackToIdle(key, pc)
	}
}

func connectionHeader(req *Request, resp *Response) string {
	reqCh := ""
	respCh := ""
	if req.Headers != nil {
		reqCh, _ = req.Headers.Get("connection")
	}

	if resp.Headers != nil {
		respCh, _ = resp.Headers.Get("connection")
	}

	if reqCh == "close" || respCh == "close" {
		return "close"
	}

	if reqCh == "" && respCh == "" {
		return "keep-alive"
	}

	return "keep-alive"
}

func canonicalKey(req *Request) string {
	return fmt.Sprintf("%s://%s", req.URL.Scheme, req.URL.Host)
}
