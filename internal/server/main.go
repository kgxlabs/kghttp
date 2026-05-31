package server

import (
	"fmt"
	"go-http-server/internal/response"
	"log"
	"net"
	"sync/atomic"
)

type Server struct {
	listener net.Listener
	stopped  atomic.Bool
}

func Serve(port int) (*Server, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, err
	}

	server := &Server{
		listener: listener,
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
	defaultHeaders := response.GetDefaultHeaders(0)

	response.WriteStatusLine(conn, response.StatusOK)
	err := response.WriteHeaders(conn, defaultHeaders)
	if err != nil {
		fmt.Printf("error %v\n", err)
	}
}
