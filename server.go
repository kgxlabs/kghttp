package kghttp

import (
	"fmt"
	"log"
	"net"
	"strconv"
	"sync/atomic"
)

type Server struct {
	Addr     string
	listener net.Listener
	Handler  Handler
	stopped  atomic.Bool
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
	rw := NewWriter(conn)

	req, err := RequestFromReader(conn)
	if err != nil {
		body := []byte(fmt.Sprintf("failed to parse request: %v", err))
		rw.Headers().Set("content-length", strconv.Itoa(len(body)))
		rw.Headers().Set("content-type", "text/plain")
		rw.Headers().Set("connection", "close")
		rw.WriteHeaders(StatusInternalServerError)
		rw.WriteBody(body)
		return
	}

	s.Handler(rw, req)
}
