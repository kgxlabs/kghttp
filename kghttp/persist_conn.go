package kghttp

import (
	"io"
	"net"
	"strconv"

	"github.com/Kaung-HtetKyaw/kgx/kgbuf"
)

type persistConn struct {
	bw   kgbuf.Writer
	br   kgbuf.Reader
	conn net.Conn
}

func (pc *persistConn) writeRequest(req *Request) error {
	prepareTransferRequestHeaders(req)

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

func prepareTransferRequestHeaders(req *Request) {
	if _, ok := req.Headers.Get("host"); !ok {
		req.Headers.Set("host", req.URL.Host)
	}

	if req.Body == nil {
		return
	}

	if _, ok := req.Headers.Get("transfer-encoding"); ok {
		req.Headers.Remove("content-length")
		return
	}

	if _, ok := req.Headers.Get("content-length"); ok {
		return
	}

	if req.ContentLength >= 0 {
		req.Headers.Set("content-length", strconv.FormatInt(int64(req.ContentLength), 10))
		return
	}

	req.Headers.Set("transfer-encoding", "chunked")
}
