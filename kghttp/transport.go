package kghttp

import (
	"errors"
	"fmt"
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

	if ok && len(list) > 0 {
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

func (t *Transport) putBackToIdle() error {
	return nil
}

func (t *Transport) RoundTrip(req *Request) (*Response, error) {
	key := canonicalKey(req)
	pconn, err := t.getConn(key, req)
	if err != nil {
		return nil, err
	}

	if err := pconn.writeRequest(req); err != nil {
		pconn.conn.Close()
		return nil, err
	}

	_, err = pconn.readResponse(req)
	if err != nil {
		pconn.conn.Close()
		return nil, err
	}

	return nil, nil
}

func canonicalKey(req *Request) string {
	return fmt.Sprintf("%s://%s", req.URL.Scheme, req.URL.Host)
}
