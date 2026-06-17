package kghttp

import (
	"bytes"
	"io"
	"strings"

	"github.com/Kaung-HtetKyaw/kgx/internal/testutil"
	"github.com/Kaung-HtetKyaw/kgx/kgbuf"
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

func TestReadResponseStatusLine(t *testing.T) {
	reader := kgbuf.NewReader(&testutil.ChunkedReader{
		Data:            "HTTP/1.1 200 OK\r\nContent-Length: 0\r\n\r\n",
		NumBytesPerRead: 3,
	})

	resp, err := ReadResponse(reader, nil)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "1.1", resp.StatusLine.HttpVersion)
	assert.Equal(t, StatusOK, resp.StatusLine.StatusCode)
	assert.Equal(t, "OK", resp.StatusLine.ReasonPhrase)

	reader = kgbuf.NewReader(&testutil.ChunkedReader{
		Data:            "HTTP/1.1 500 Internal Server Error\r\nContent-Length: 0\r\n\r\n",
		NumBytesPerRead: 50,
	})

	resp, err = ReadResponse(reader, nil)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, StatusInternalServerError, resp.StatusLine.StatusCode)
	assert.Equal(t, "Internal Server Error", resp.StatusLine.ReasonPhrase)

	reader = kgbuf.NewReader(&testutil.ChunkedReader{
		Data:            "TCP/1.1 200 OK\r\nContent-Length: 0\r\n\r\n",
		NumBytesPerRead: 3,
	})

	_, err = ReadResponse(reader, nil)
	require.Error(t, err)

	reader = kgbuf.NewReader(&testutil.ChunkedReader{
		Data:            "HTTP/1.1 OK\r\nContent-Length: 0\r\n\r\n",
		NumBytesPerRead: 3,
	})

	_, err = ReadResponse(reader, nil)
	require.Error(t, err)
}

func TestReadResponseHeaders(t *testing.T) {
	reader := kgbuf.NewReader(&testutil.ChunkedReader{
		Data: "HTTP/1.1 200 OK\r\n" +
			"Content-Type: text/plain\r\n" +
			"Connection: keep-alive\r\n" +
			"\r\n",
		NumBytesPerRead: 3,
	})

	resp, err := ReadResponse(reader, nil)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "text/plain", resp.Headers["content-type"])
	assert.Equal(t, "keep-alive", resp.Headers["connection"])

	reader = kgbuf.NewReader(&testutil.ChunkedReader{
		Data:            "HTTP/1.1 200 OK\r\n\r\n",
		NumBytesPerRead: 3,
	})

	resp, err = ReadResponse(reader, nil)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Empty(t, resp.Headers)

	reader = kgbuf.NewReader(&testutil.ChunkedReader{
		Data:            "HTTP/1.1 200 OK\r\nContent-Type: text/plain",
		NumBytesPerRead: 3,
	})

	_, err = ReadResponse(reader, nil)
	require.Error(t, err)
}

func TestReadResponseBody(t *testing.T) {
	reader := kgbuf.NewReader(&testutil.ChunkedReader{
		Data: "HTTP/1.1 200 OK\r\n" +
			"Content-Length: 12\r\n" +
			"\r\n" +
			"Hello World!",
		NumBytesPerRead: 3,
	})

	resp, err := ReadResponse(reader, nil)
	require.NoError(t, err)
	require.NotNil(t, resp)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, "Hello World!", string(body))

	reader = kgbuf.NewReader(&testutil.ChunkedReader{
		Data: "HTTP/1.1 200 OK\r\n" +
			"Content-Length: 0\r\n" +
			"\r\n",
		NumBytesPerRead: 3,
	})

	resp, err = ReadResponse(reader, nil)
	require.NoError(t, err)
	require.NotNil(t, resp)
	body, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, "", string(body))

	reader = kgbuf.NewReader(&testutil.ChunkedReader{
		Data: "HTTP/1.1 200 OK\r\n" +
			"Content-Length: 20\r\n" +
			"\r\n" +
			"partial content",
		NumBytesPerRead: 3,
	})

	resp, err = ReadResponse(reader, nil)
	require.NoError(t, err)
	require.NotNil(t, resp)
	_, err = io.ReadAll(resp.Body)
	require.Error(t, err)

	reader = kgbuf.NewReader(&testutil.ChunkedReader{
		Data: "HTTP/1.1 200 OK\r\n" +
			"\r\n" +
			"ignored body",
		NumBytesPerRead: 3,
	})

	resp, err = ReadResponse(reader, nil)
	require.NoError(t, err)
	require.NotNil(t, resp)
	body, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, "", string(body))
}

func TestReadResponseReadsMultipleResponses(t *testing.T) {
	reader := kgbuf.NewReader(strings.NewReader(
		"HTTP/1.1 200 OK\r\n" +
			"Content-Length: 5\r\n" +
			"\r\n" +
			"first" +
			"HTTP/1.1 500 Internal Server Error\r\n" +
			"Content-Length: 6\r\n" +
			"\r\n" +
			"second",
	))

	resp, err := ReadResponse(reader, nil)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, StatusOK, resp.StatusLine.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, "first", string(body))

	resp, err = ReadResponse(reader, nil)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, StatusInternalServerError, resp.StatusLine.StatusCode)
	body, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, "second", string(body))
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
