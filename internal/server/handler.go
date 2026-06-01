package server

import (
	"go-http-server/internal/request"
	"go-http-server/internal/response"
	"io"
)

type Handler func(w io.Writer, req *request.Request) *HandlerError

type HandlerError struct {
	StatusCode int
	Message    string
}

func GetHandlerError(statusCode response.StatusCode) HandlerError {
	return HandlerError{
		StatusCode: int(statusCode),
		Message:    response.GetStatusCodeMessage(statusCode),
	}
}
