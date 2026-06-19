package kghttp

import (
	"github.com/Kaung-HtetKyaw/kgx/kgbuf"
	"net"
)

type persistConn struct {
	bw   kgbuf.Writer
	br   kgbuf.Reader
	conn net.Conn
}

func (pc *persistConn) writeRequest(req *Request) error {
	return nil
}

func (pc *persistConn) readResponse(req *Request) (*Response, error) {
	return ReadResponse(&pc.br, req)
}
