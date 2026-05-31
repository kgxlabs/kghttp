package response

import (
	"fmt"
	"go-http-server/internal/headers"
	"io"
	"strconv"
)

func WriteHeaders(w io.Writer, headers headers.Headers) error {
	headersStr := ""
	for key, value := range headers {
		headersStr = headersStr + fmt.Sprintf("%s: %s\r\n", key, value)

	}

	headersStr = headersStr + "\r\n"
	_, err := w.Write([]byte(headersStr))
	if err != nil {
		return fmt.Errorf("failed to write headers: %s", err)
	}

	return nil
}

func GetDefaultHeaders(contentLen int) headers.Headers {
	return headers.Headers{
		"Content-Length": strconv.Itoa(contentLen),
		"Connection":     "close",
		"Content-Type":   "text/plain",
	}
}
