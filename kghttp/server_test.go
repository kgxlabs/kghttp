package kghttp

import (
	"github.com/kgxlabs/kghttp/kgbuf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net"
	"strconv"
	"testing"
	"time"
)

func TestListenAndServe(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	server := &Server{
		Handler: func(w *ResponseWriter, req *Request) {
			body := "Hello World"
			data := []byte(body)
			w.Headers().Set("content-length", strconv.Itoa(len(data)))
			w.Headers().Set("content-type", "text/plain")
			require.NoError(t, w.WriteHeaders(StatusOK))
			_, err := w.Write(data)
			require.NoError(t, err)
		},
		IdleConnTimeOut: 5 * time.Second,
	}
	err = server.Serve(ln)
	require.NoError(t, err)
	defer server.Close()

	resp, err := Get("http://" + ln.Addr().String() + "/ok")
	require.NoError(t, err)
	assert.Equal(t, StatusOK, resp.StatusLine.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, "Hello World", string(body))
}

func TestListenAndServeMissingHost(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	server := &Server{
		Handler: func(w *ResponseWriter, req *Request) {
			t.Fatal("handler should not be called")
		},
		IdleConnTimeOut: 5 * time.Second,
	}
	err = server.Serve(ln)
	require.NoError(t, err)
	defer server.Close()

	conn, err := net.Dial("tcp", ln.Addr().String())
	require.NoError(t, err)
	defer conn.Close()
	reader := kgbuf.NewReader(conn)

	_, err = conn.Write([]byte(
		"GET /ok HTTP/1.1\r\n" +
			"Content-Length: 0\r\n" +
			"\r\n",
	))
	require.NoError(t, err)

	resp, err := ReadResponse(reader, nil)
	require.NoError(t, err)
	assert.Equal(t, StatusBadRequest, resp.StatusLine.StatusCode)
}
