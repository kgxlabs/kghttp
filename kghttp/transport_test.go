package kghttp

import (
	"bytes"
	"errors"
	"github.com/Kaung-HtetKyaw/kgx/kgurl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net"
	"strconv"
	"testing"
	"time"
)

func TestTransportRoundTrip(t *testing.T) {
	t.Run("valid GET request", func(t *testing.T) {
		server := newTestServer(t, func(w *ResponseWriter, req *Request) {
			writeStringResponse(t, w, "hello world")
		})

		req := newTransportRequest(t, "GET", server.listener.Addr().String(), nil, nil)
		resp, err := NewTransport().RoundTrip(req)
		require.NoError(t, err)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, "hello world", string(body))
	})

	t.Run("valid POST request declared content-length", func(t *testing.T) {
		server := newTestServer(t, func(w *ResponseWriter, req *Request) {
			body, err := io.ReadAll(req.Body)
			require.NoError(t, err)
			writeStringResponse(t, w, string(body))
		})

		req := newTransportRequest(t, "POST", server.listener.Addr().String(), Headers{
			"content-length": "11",
		}, newBody([]byte("hello world")))
		resp, err := NewTransport().RoundTrip(req)
		require.NoError(t, err)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, "hello world", string(body))
	})

	t.Run("valid POST request chunked encoding", func(t *testing.T) {
		addr, captured, closeServer := newRawRequestServer(t, "0\r\n\r\n", ""+
			"HTTP/1.1 200 OK\r\n"+
			"Content-Length: 11\r\n"+
			"\r\n"+
			"hello world")
		defer closeServer()

		req := newTransportRequest(t, "POST", addr, Headers{
			"transfer-encoding": "chunked",
		}, newBody([]byte("hello world")))
		resp, err := NewTransport().RoundTrip(req)
		require.NoError(t, err)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, "hello world", string(body))

		rawReq := <-captured
		assert.Contains(t, rawReq, "transfer-encoding: chunked\r\n")
		assert.Contains(t, rawReq, "b\r\nhello world\r\n0\r\n\r\n")
	})

	t.Run("invalid body shorter than declared content-length", func(t *testing.T) {
		server := newTestServer(t, func(w *ResponseWriter, req *Request) {
			t.Fatal("handler should not be called for a partial request body")
		})

		req := newTransportRequest(t, "POST", server.listener.Addr().String(), Headers{
			"content-length": "11",
		}, newBody([]byte("hello")))
		_, err := NewTransport().RoundTrip(req)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrPartialBody)
	})

	t.Run("invalid malformed chunked encoding", func(t *testing.T) {
		addr, closeServer := newRawResponseServer(t, ""+
			"HTTP/1.1 200 OK\r\n"+
			"Transfer-Encoding: chunked\r\n"+
			"\r\n"+
			"z\r\n"+
			"hello\r\n")
		defer closeServer()

		req := newTransportRequest(t, "GET", addr, nil, nil)
		resp, err := NewTransport().RoundTrip(req)
		require.NoError(t, err)

		_, err = io.ReadAll(resp.Body)
		require.Error(t, err)
	})
}

func newTestServer(t *testing.T, handler Handler) *Server {
	t.Helper()

	server := &Server{
		Addr:            "127.0.0.1:0",
		Handler:         handler,
		IdleConnTimeOut: 5 * time.Second,
	}
	err := server.ListenAndServe()
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, server.Close())
	})

	return server
}

func newTransportRequest(t *testing.T, method, addr string, headers Headers, reqBody io.ReadCloser) *Request {
	t.Helper()

	url, err := kgurl.Parse("http://" + addr)
	require.NoError(t, err)

	if headers == nil {
		headers = Headers{}
	}
	headers.Set("host", addr)

	return &Request{
		Method:     method,
		Proto:      "HTTP",
		ProtoMajor: 1,
		ProtoMinor: 1,
		URL:        url,
		Headers:    headers,
		Body:       reqBody,
	}
}

func writeStringResponse(t *testing.T, w *ResponseWriter, body string) {
	t.Helper()

	data := []byte(body)
	w.Headers().Set("content-length", strconv.Itoa(len(data)))
	w.Headers().Set("content-type", "text/plain")
	require.NoError(t, w.WriteHeaders(StatusOK))
	_, err := w.Write(data)
	require.NoError(t, err)
}

func newRawResponseServer(t *testing.T, response string) (string, func()) {
	t.Helper()

	addr, _, closeServer := newRawRequestServer(t, "\r\n\r\n", response)
	return addr, closeServer
}

func newRawRequestServer(t *testing.T, readUntil string, response string) (string, <-chan string, func()) {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	captured := make(chan string, 1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer close(captured)
		conn, err := ln.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			t.Errorf("accept raw response connection: %v", err)
			return
		}
		defer conn.Close()

		var req bytes.Buffer
		buf := make([]byte, 1024)
		for !bytes.Contains(req.Bytes(), []byte(readUntil)) {
			n, err := conn.Read(buf)
			if n > 0 {
				req.Write(buf[:n])
			}
			if err != nil {
				t.Errorf("read raw request: %v", err)
				return
			}
		}
		captured <- req.String()

		_, err = conn.Write([]byte(response))
		if err != nil {
			t.Errorf("write raw response: %v", err)
		}
	}()

	return ln.Addr().String(), captured, func() {
		require.NoError(t, ln.Close())
		<-done
	}
}

type body struct {
	r io.Reader
}

func newBody(p []byte) *body {
	return &body{
		r: bytes.NewReader(p),
	}
}

func (r *body) Read(p []byte) (int, error) {
	return r.r.Read(p)
}

func (r *body) Close() error {
	return nil
}
