package kghttp

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/Kaung-HtetKyaw/kgx/kgbuf"
)

type Server struct {
	Addr            string
	Handler         Handler
	IdleConnTimeOut time.Duration
	listener        net.Listener
	stopped         atomic.Bool
}

type Handler func(w *ResponseWriter, req *Request)

func (s *Server) ListenAndServe() error {
	listener, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return err
	}

	s.listener = listener

	return s.Serve(listener)
}

func (s *Server) Serve(ln net.Listener) error {
	s.listener = ln
	go s.listen()

	return nil
}

func (s *Server) Close() error {
	s.stopped.Store(true)
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

func (s *Server) isStopped() bool {
	return s.stopped.Load()
}

func (s *Server) listen() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			if s.isStopped() {
				return
			}

			log.Printf("error accepting connection: %s", err)
			continue
		}

		go s.handle(conn)
	}
}

func (s *Server) handle(conn net.Conn) {
	defer conn.Close()

	r := kgbuf.NewReader(conn)

	for {
		rw := NewWriter(conn)

		if s.IdleConnTimeOut > 0 {
			conn.SetReadDeadline(time.Now().Add(s.IdleConnTimeOut))
		} else {
			conn.SetReadDeadline(time.Time{})
		}

		req, err := ReadRequest(r)
		conn.SetReadDeadline(time.Time{})
		if err != nil {
			if errors.Is(err, io.EOF) {
				// Client closed connection
				return
			}

			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				return
			}

			body := []byte(fmt.Sprintf("failed to parse request: %v", err))
			rw.Headers().Set("content-length", strconv.Itoa(len(body)))
			rw.Headers().Set("content-type", "text/plain")
			rw.Headers().Set("connection", "close")
			rw.WriteHeaders(StatusInternalServerError)
			rw.WriteBody(body)
			return
		}

		s.Handler(rw, req)
		c, _ := rw.Headers().Get("connection")
		if c == "close" {
			return
		}
		c, _ = req.Headers.Get("connection")
		if c == "close" {
			return
		}
	}
}
