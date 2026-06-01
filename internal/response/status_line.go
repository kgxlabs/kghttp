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

func GetStatusCodeMessage(s StatusCode) string {
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

func getStatusLine(s StatusCode) []byte {
	return []byte(fmt.Sprintf("HTTP/1.1 %d %s\r\n", s, GetStatusCodeMessage(s)))
}

func WriteStatusLine(w io.Writer, statusCode StatusCode) error {
	_, err := w.Write(getStatusLine(statusCode))
	if err != nil {
		return err
	}

	return nil
}
