package server

import (
	"fmt"
	"go-http-server/internal/headers"
	"go-http-server/internal/request"
	"go-http-server/internal/response"
	"log"
	"net"
	"sync/atomic"
)

type Server struct {
	listener net.Listener
	handler  Handler
	stopped  atomic.Bool
}

type Handler func(w *response.Writer, req *request.Request)

func Serve(port int, handlerFunc Handler) (*Server, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, err
	}

	server := &Server{
		listener: listener,
		handler:  handlerFunc,
	}

	go server.listen()

	return server, nil
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
	req, err := request.RequestFromReader(conn)
	if err != nil {
		log.Printf("failed to read or parse request: %v", err)
		return
	}

	rw := &response.Writer{
		Headers: make(headers.Headers),
	}
	s.handler(rw, req)
	if _, err := conn.Write(rw.ResponseBytes()); err != nil {
		log.Printf("failed to write response: %v", err)
		return
	}
}
