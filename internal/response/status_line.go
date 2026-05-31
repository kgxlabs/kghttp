package response

import (
	"fmt"
	"io"
)

type StatusCode int

const (
	StatusOK                  StatusCode = 200
	StatusBadRequest          StatusCode = 400
	StatusInternalServerError StatusCode = 500
)

func (s StatusCode) String() string {
	switch s {
	case 200:
		return "OK"
	case 400:
		return "Bad Request"
	case 500:
		return "Internal Server Error"
	default:
		return ""
	}
}

func WriteStatusLine(w io.Writer, statusCode StatusCode) error {
	statusLine := fmt.Sprintf("HTTP/1.1 %d %s\r\n", statusCode, StatusCode(statusCode))
	_, err := w.Write([]byte(statusLine))
	if err != nil {
		return err
	}

	return nil
}
