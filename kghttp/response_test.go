package kghttp

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestWriteResponseWholeBody(t *testing.T) {
	// Test Valid Write response
	resWriter, w := newTestWriter()
	body := `Hello World`
	resWriter.Headers().Set("connection", "close")
	resWriter.WriteHeaders(200)
	_, err := resWriter.WriteBody([]byte(body))
	resp := w.Result()
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Contains(t, resp, "HTTP/1.1 200 OK\r\n")
	assert.Contains(t, resp, "connection: close\r\n\r\n")
	assert.Contains(t, resp, "Hello World")
}

func TestWriteResponseChunkedWithTrailers(t *testing.T) {
	body := "Hello World"
	data := []byte(body)
	// Test Valid Write chunked response with trailer
	resWriter, w := newTestWriter()
	resWriter.Headers().Set("connection", "close")
	resWriter.Headers().Set("trailer", "x-content-length")
	resWriter.WriteHeaders(200)
	n, err := resWriter.WriteChunkedBody(data[:3])
	require.NoError(t, err)
	assert.Equal(t, 8, n)

	resWriter.Trailers().Set("x-content-length", "8")
	n, err = resWriter.WriteChunkedBodyDone()
	resp := w.Result()
	require.NoError(t, err)
	assert.Equal(t, 3, n)
	assert.Contains(t, resp, "HTTP/1.1 200 OK\r\n")
	assert.Contains(t, resp, "connection: close\r\n")
	assert.Contains(t, resp, "trailer: x-content-length\r\n")
	assert.Contains(t, resp, "3\r\nHel\r\n0\r\n")
	assert.Contains(t, resp, "x-content-length: 8\r\n\r\n")

	// Valid Write chunked response without headers
	resWriter, w = newTestWriter()
	resWriter.WriteHeaders(200)
	n, err = resWriter.WriteChunkedBody(data[:3])
	require.NoError(t, err)
	assert.Equal(t, 8, n)
	n, err = resWriter.WriteChunkedBodyDone()
	resp = w.Result()
	require.NoError(t, err)
	assert.Equal(t, 3, n)
	assert.Contains(t, resp, "HTTP/1.1 200 OK\r\n\r\n")

	// Invalid Write chunked body before headers
	resWriter, _ = newTestWriter()
	n, err = resWriter.WriteChunkedBody(data[:3])
	require.Error(t, err)
	assert.Equal(t, 0, n)

	// Invalid Write done before headers
	resWriter, _ = newTestWriter()
	n, err = resWriter.WriteChunkedBodyDone()
	require.Error(t, err)
	assert.Equal(t, 0, n)
}

func newTestWriter() (*ResponseWriter, *memWriter) {
	w := &memWriter{}
	return NewWriter(w), w
}

type memWriter struct {
	buf bytes.Buffer
}

func (w *memWriter) Write(p []byte) (int, error) {
	n, err := w.buf.Write(p)
	return n, err
}

func (w *memWriter) Result() string {
	return w.buf.String()
}
