package kghttp

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"testing"
)

func TestClientMethods(t *testing.T) {
	t.Run("GET request", func(t *testing.T) {
		server := newTestServer(t, func(w *ResponseWriter, req *Request) {
			writeStringResponse(t, w, "hello world")
		})

		resp, err := Get("http://" + server.listener.Addr().String())
		require.NoError(t, err)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, "hello world", string(body))
	})

	t.Run("POST request known content length", func(t *testing.T) {
		server := newTestServer(t, func(w *ResponseWriter, req *Request) {
			body, err := io.ReadAll(req.Body)
			require.NoError(t, err)
			writeStringResponse(t, w, string(body))
		})

		req, err := NewRequest("POST", "http://"+server.listener.Addr().String(), newBody([]byte("hello world")))
		require.NoError(t, err)

		resp, err := DefaultClient.Do(req)
		require.NoError(t, err)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, "hello world", string(body))
	})

	t.Run("POST request unknown content length", func(t *testing.T) {
		addr, captured, closeServer := newRawRequestServer(t, "0\r\n\r\n", ""+
			"HTTP/1.1 200 OK\r\n"+
			"Content-Length: 11\r\n"+
			"\r\n"+
			"hello world")
		defer closeServer()

		resp, err := Post("http://"+addr, "text/plain", newBody([]byte("hello world")))
		require.NoError(t, err)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, "hello world", string(body))

		rawReq := <-captured
		assert.Contains(t, rawReq, "transfer-encoding: chunked\r\n")
		assert.Contains(t, rawReq, "b\r\nhello world\r\n0\r\n\r\n")
	})
}
