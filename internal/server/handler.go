package server

import (
	"go-http-server/internal/request"
	"go-http-server/internal/response"
	"io"
	"log"
)

type Handler func(w io.Writer, req *request.Request) *HandlerError

type HandlerError struct {
	StatusCode response.StatusCode
	Message    string
}

func (hE HandlerError) Write(w io.Writer) {
	if err := response.WriteStatusLine(w, hE.StatusCode); err != nil {
		log.Printf("failed to write error status line: %v", err)
		return
	}

	headers := response.GetDefaultHeaders(len(hE.Message))
	if err := response.WriteHeaders(w, headers); err != nil {
		log.Printf("failed to write error headers: %v", err)
		return
	}

	if _, err := w.Write([]byte(hE.Message)); err != nil {
		log.Printf("failed to write error body: %v", err)
		return
	}
}
