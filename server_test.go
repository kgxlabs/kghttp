package kghttp

import (
	"fmt"
	"io"
	"net"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListenAndServe(t *testing.T) {
	port := 9999
	addr := fmt.Sprintf(":%d", port)
	server := &Server{
		Addr: addr,
		Handler: func(w *ResponseWriter, req *Request) {
			body := "Hello World"
			data := []byte(body)
			w.Headers().Set("connection", "close")
			w.Headers().Set("content-type", strconv.Itoa(len(data)))
			w.WriteHeaders(200)
			w.WriteBody(data)

		},
	}
	err := server.ListenAndServe()
	require.NoError(t, err)
	defer server.Close()

	conn, err := net.Dial("tcp", addr)
	require.NoError(t, err)
	defer conn.Close()

	_, err = conn.Write([]byte(
		"GET / HTTP/1.1\r\n" +
			"Host: localhost\r\n" +
			"\r\n",
	))
	require.NoError(t, err)

	resp, err := io.ReadAll(conn)
	require.NoError(t, err)

	assert.Contains(t, string(resp), "HTTP/1.1 200 OK")
	assert.Contains(t, string(resp), "Hello World")
}
