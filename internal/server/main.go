package server

import (
	"bytes"
	"fmt"
	"go-http-server/internal/request"
	"go-http-server/internal/response"
	"io"
	"log"
	"net"
	"sync/atomic"
)

type Server struct {
	listener net.Listener
	handler  Handler
	stopped  atomic.Bool
}

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
	request, err := request.RequestFromReader(conn)
	if err != nil {
		fmt.Printf("error %v\n", err)
	}

	var buf bytes.Buffer
	handlerError := s.handler(&buf, request)
	if handlerError != nil {
		writeError(conn, *handlerError)
		return
	}

	if err := response.WriteStatusLine(conn, response.StatusOK); err != nil {
		log.Printf("failed to write response status line: %v", err)
		return
	}

	headers := response.GetDefaultHeaders(len(buf.Bytes()))
	if err := response.WriteHeaders(conn, headers); err != nil {
		log.Printf("failed to write response headers: %v", err)
		return
	}

	if _, err := conn.Write(buf.Bytes()); err != nil {
		log.Printf("failed to write response: %v", err)
		return
	}

}

func writeError(w io.Writer, handlerError HandlerError) {

	if err := response.WriteStatusLine(w, response.StatusCode(handlerError.StatusCode)); err != nil {
		log.Printf("failed to write error status line: %v", err)
		return
	}

	headers := response.GetDefaultHeaders(len(handlerError.Message))
	if err := response.WriteHeaders(w, headers); err != nil {
		log.Printf("failed to write error headers: %v", err)
		return
	}

	if _, err := w.Write([]byte(handlerError.Message)); err != nil {
		log.Printf("failed to write error body: %v", err)
		return
	}
}
