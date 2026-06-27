package kghttp

import (
	"bytes"
	"io"
	"strings"

	"github.com/kgxlabs/kghttp/internal/testutil"
	"github.com/kgxlabs/kghttp/kgbuf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestWriteResponseWholeBody(t *testing.T) {
	// Valid: Write response
	rw, ds := newTestWriter()
	body := `Hello World`
	rw.Headers().Set("content-length", "11")
	rw.Headers().Set("connection", "close")
	err := rw.WriteHeaders(200)
	require.NoError(t, err)
	_, err = rw.Write([]byte(body))
	require.NoError(t, err)
	err = rw.finish()
	require.NoError(t, err)
	resp := ds.Result()
	require.NotNil(t, resp)
	assert.Contains(t, resp, "HTTP/1.1 200 OK\r\n")
	assert.Contains(t, resp, "connection: close\r\n")
	assert.Contains(t, resp, "content-length: 11\r\n")
	assert.Contains(t, resp, "Hello World")
}

func TestWriteResponseChunkedWithTrailers(t *testing.T) {
	body := "Hello World"
	data := []byte(body)
	t.Run("write chunked response with trailer", func(t *testing.T) {
		resWriter, w := newTestWriter()
		resWriter.Headers().Set("connection", "close")
		resWriter.Headers().Set("transfer-encoding", "chunked")
		resWriter.Headers().Set("trailer", "x-content-length")
		err := resWriter.WriteHeaders(200)
		require.NoError(t, err)
		n, err := resWriter.Write(data[:6])
		require.NoError(t, err)
		assert.Equal(t, 6, n)
		n, err = resWriter.Write(data[6:])
		require.NoError(t, err)
		assert.Equal(t, 5, n)
		resWriter.Trailers().Set("x-content-length", "11")
		err = resWriter.finish()
		require.NoError(t, err)
		assert.Contains(t, w.Result(), "6\r\nHello \r\n5\r\nWorld\r\n0\r\nx-content-length: 11\r\n\r\n")
	})

	t.Run("write chunked response with trailer headers", func(t *testing.T) {
		resWriter, w := newTestWriter()
		resWriter.Headers().Set("connection", "close")
		resWriter.Headers().Set("transfer-encoding", "chunked")
		resWriter.Headers().Set("trailer", "x-content-length")
		err := resWriter.WriteHeaders(200)
		require.NoError(t, err)
		n, err := resWriter.Write(data[:3])
		require.NoError(t, err)
		assert.Equal(t, 3, n)
		resWriter.Trailers().Set("x-content-length", "8")
		err = resWriter.finish()
		resp := w.Result()
		require.NoError(t, err)
		assert.Contains(t, resp, "HTTP/1.1 200 OK\r\n")
		assert.Contains(t, resp, "connection: close\r\n")
		assert.Contains(t, resp, "transfer-encoding: chunked\r\n")
		assert.Contains(t, resp, "trailer: x-content-length\r\n")
		assert.Contains(t, resp, "3\r\nHel\r\n0\r\n")
		assert.Contains(t, resp, "x-content-length: 8\r\n\r\n")
	})

	t.Run("write chunked response without headers", func(t *testing.T) {
		resWriter, w := newTestWriter()
		resWriter.Headers().Set("transfer-encoding", "chunked")
		err := resWriter.WriteHeaders(200)
		require.NoError(t, err)
		n, err := resWriter.Write(data[:3])
		require.NoError(t, err)
		assert.Equal(t, 3, n)
		err = resWriter.finish()
		resp := w.Result()
		require.NoError(t, err)
		assert.Contains(t, resp, "HTTP/1.1 200 OK\r\n")
		assert.Contains(t, resp, "transfer-encoding: chunked\r\n")
		assert.Contains(t, resp, "3\r\nHel\r\n0\r\n\r\n")
	})

	t.Run("auto-sends headers before the body", func(t *testing.T) {
		resWriter, _ := newTestWriter()
		resWriter.Headers().Set("content-length", "3")
		n, err := resWriter.Write(data[:3])
		require.NoError(t, err)
		assert.Equal(t, 3, n)
	})

	t.Run("finish before headers", func(t *testing.T) {
		resWriter, _ := newTestWriter()
		err := resWriter.finish()
		require.Error(t, err)
	})
}

func TestReadResponseStatusLine(t *testing.T) {
	t.Run("valid OK status line", func(t *testing.T) {
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
	})

	t.Run("valid internal server error status line", func(t *testing.T) {
		reader := kgbuf.NewReader(&testutil.ChunkedReader{
			Data:            "HTTP/1.1 500 Internal Server Error\r\nContent-Length: 0\r\n\r\n",
			NumBytesPerRead: 50,
		})

		resp, err := ReadResponse(reader, nil)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, StatusInternalServerError, resp.StatusLine.StatusCode)
		assert.Equal(t, "Internal Server Error", resp.StatusLine.ReasonPhrase)
	})

	t.Run("invalid version", func(t *testing.T) {
		reader := kgbuf.NewReader(&testutil.ChunkedReader{
			Data:            "TCP/1.1 200 OK\r\nContent-Length: 0\r\n\r\n",
			NumBytesPerRead: 3,
		})

		_, err := ReadResponse(reader, nil)
		require.Error(t, err)
	})

	t.Run("invalid status code", func(t *testing.T) {
		reader := kgbuf.NewReader(&testutil.ChunkedReader{
			Data:            "HTTP/1.1 OK\r\nContent-Length: 0\r\n\r\n",
			NumBytesPerRead: 3,
		})

		_, err := ReadResponse(reader, nil)
		require.Error(t, err)
	})
}

func TestReadResponseHeaders(t *testing.T) {
	t.Run("standard headers", func(t *testing.T) {
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
	})

	t.Run("empty headers", func(t *testing.T) {
		reader := kgbuf.NewReader(&testutil.ChunkedReader{
			Data:            "HTTP/1.1 200 OK\r\n\r\n",
			NumBytesPerRead: 3,
		})

		resp, err := ReadResponse(reader, nil)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Empty(t, resp.Headers)
	})

	t.Run("missing end headers", func(t *testing.T) {
		reader := kgbuf.NewReader(&testutil.ChunkedReader{
			Data:            "HTTP/1.1 200 OK\r\nContent-Type: text/plain",
			NumBytesPerRead: 3,
		})

		_, err := ReadResponse(reader, nil)
		require.Error(t, err)
	})
}

func TestReadResponseBody(t *testing.T) {
	t.Run("standard body", func(t *testing.T) {
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
	})

	t.Run("empty body", func(t *testing.T) {
		reader := kgbuf.NewReader(&testutil.ChunkedReader{
			Data: "HTTP/1.1 200 OK\r\n" +
				"Content-Length: 0\r\n" +
				"\r\n",
			NumBytesPerRead: 3,
		})

		resp, err := ReadResponse(reader, nil)
		require.NoError(t, err)
		require.NotNil(t, resp)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, "", string(body))
	})

	t.Run("body shorter than content length", func(t *testing.T) {
		reader := kgbuf.NewReader(&testutil.ChunkedReader{
			Data: "HTTP/1.1 200 OK\r\n" +
				"Content-Length: 20\r\n" +
				"\r\n" +
				"partial content",
			NumBytesPerRead: 3,
		})

		resp, err := ReadResponse(reader, nil)
		require.NoError(t, err)
		require.NotNil(t, resp)
		_, err = io.ReadAll(resp.Body)
		require.Error(t, err)
	})

	t.Run("no content length ignores body", func(t *testing.T) {
		reader := kgbuf.NewReader(&testutil.ChunkedReader{
			Data: "HTTP/1.1 200 OK\r\n" +
				"\r\n" +
				"ignored body",
			NumBytesPerRead: 3,
		})

		resp, err := ReadResponse(reader, nil)
		require.NoError(t, err)
		require.NotNil(t, resp)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, "", string(body))
	})
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
	ds := &memWriter{}
	return NewWriter(ds), ds
}

type memWriter struct {
	buf  bytes.Buffer
	Body io.ReadCloser
}

func (w *memWriter) Write(p []byte) (int, error) {
	n, err := w.buf.Write(p)
	return n, err
}

func (w *memWriter) Close() error {
	return nil
}

func (w *memWriter) Result() string {
	return w.buf.String()
}
