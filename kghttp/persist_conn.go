package kghttp

import (
	"io"
	"net"

	"github.com/Kaung-HtetKyaw/kgx/kgbuf"
)

type persistConn struct {
	bw   kgbuf.Writer
	br   kgbuf.Reader
	conn net.Conn
}

func (pc *persistConn) writeRequest(req *Request) error {
	bs := serizlizeReqStatusLine(req)

	if _, err := pc.bw.Write(bs); err != nil {
		return err
	}

	bh, err := serializeHeaders(req.Headers)
	if err != nil {
		return err
	}

	if _, err := pc.bw.Write(bh); err != nil {
		return err
	}

	cfg := writeTransferCfg{
		writer: &pc.bw,
		headers: func() Headers {
			return req.Headers
		},
		trailers: func() Headers {
			return req.Trailers
		},
	}
	tw, err := writeTransfer(cfg)
	if err != nil {
		return err
	}

	if tw != nil && req.Body != nil {
		if _, err = io.Copy(tw, req.Body); err != nil {
			return err
		}

		if err := tw.Close(); err != nil {
			return err
		}
	}

	return pc.bw.Flush()
}

func (pc *persistConn) readResponse(req *Request) (*Response, error) {
	return ReadResponse(&pc.br, req)
}

func (pc *persistConn) close() error {
	return pc.conn.Close()
}
