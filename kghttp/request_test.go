package kghttp

import (
	"fmt"
	"github.com/kgxlabs/kghttp/internal/testutil"
	"github.com/kgxlabs/kghttp/kgbuf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"strings"
	"testing"
)

func TestRequestLineParse(t *testing.T) {
	t.Run("good GET request line", func(t *testing.T) {
		reader := kgbuf.NewReader(&testutil.ChunkedReader{
			Data:            "GET / HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n",
			NumBytesPerRead: 3,
		})
		r, err := ReadRequest(reader)
		require.NoError(t, err)
		require.NotNil(t, r)
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/", r.URL.Path)
		assert.Equal(t, "HTTP/1.1", r.Proto)
		assert.Equal(t, 1, r.ProtoMajor)
		assert.Equal(t, 1, r.ProtoMinor)
	})

	t.Run("good GET request line with path", func(t *testing.T) {
		reader := kgbuf.NewReader(&testutil.ChunkedReader{
			Data:            "GET /coffee HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n",
			NumBytesPerRead: 1,
		})
		r, err := ReadRequest(reader)
		require.NoError(t, err)
		require.NotNil(t, r)
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/coffee", r.URL.Path)
		assert.Equal(t, "HTTP/1.1", r.Proto)
		assert.Equal(t, 1, r.ProtoMajor)
		assert.Equal(t, 1, r.ProtoMinor)
	})

	t.Run("good POST request with path", func(t *testing.T) {
		reader := kgbuf.NewReader(&testutil.ChunkedReader{
			Data:            "POST /coffee HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n",
			NumBytesPerRead: 50,
		})
		r, err := ReadRequest(reader)
		require.NoError(t, err)
		require.NotNil(t, r)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/coffee", r.URL.Path)
		assert.Equal(t, "HTTP/1.1", r.Proto)
		assert.Equal(t, 1, r.ProtoMajor)
		assert.Equal(t, 1, r.ProtoMinor)
	})

	t.Run("invalid number of parts in request line", func(t *testing.T) {
		reader := kgbuf.NewReader(&testutil.ChunkedReader{
			Data:            "/coffee HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n",
			NumBytesPerRead: 3,
		})
		_, err := ReadRequest(reader)
		require.Error(t, err)
	})

	t.Run("invalid method out of order request line", func(t *testing.T) {
		reader := kgbuf.NewReader(&testutil.ChunkedReader{
			Data:            "/coffee POST HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n",
			NumBytesPerRead: 3,
		})
		_, err := ReadRequest(reader)
		require.Error(t, err)
	})

	t.Run("invalid version in request line", func(t *testing.T) {
		reader := kgbuf.NewReader(&testutil.ChunkedReader{
			Data:            "OPTIONS /prime/rib TCP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n",
			NumBytesPerRead: 50,
		})
		_, err := ReadRequest(reader)
		require.Error(t, err)
	})
}

func TestHeadersParse(t *testing.T) {
	t.Run("standard headers", func(t *testing.T) {
		reader := kgbuf.NewReader(&testutil.ChunkedReader{
			Data:            "GET / HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n",
			NumBytesPerRead: 3,
		})
		r, err := ReadRequest(reader)
		require.NoError(t, err)
		require.NotNil(t, r)
		assert.Equal(t, "localhost:42069", r.Headers["host"])
		assert.Equal(t, "curl/7.81.0", r.Headers["user-agent"])
		assert.Equal(t, "*/*", r.Headers["accept"])
	})

	t.Run("malformed header", func(t *testing.T) {
		reader := kgbuf.NewReader(&testutil.ChunkedReader{
			Data:            "GET / HTTP/1.1\r\nHost localhost:42069\r\n\r\n",
			NumBytesPerRead: 3,
		})
		_, err := ReadRequest(reader)
		require.Error(t, err)
	})

	t.Run("empty header", func(t *testing.T) {
		reader := kgbuf.NewReader(&testutil.ChunkedReader{
			Data:            "GET / HTTP/1.1\r\n\r\n",
			NumBytesPerRead: 3,
		})
		_, err := ReadRequest(reader)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrBadRequest)
	})

	t.Run("duplicate headers", func(t *testing.T) {
		reader := kgbuf.NewReader(&testutil.ChunkedReader{
			Data:            "GET / HTTP/1.1\r\nhost: localhost:42069\r\nhost: duplicate:420420\r\n\r\n",
			NumBytesPerRead: 3,
		})
		r, err := ReadRequest(reader)
		require.NoError(t, err)
		require.NotNil(t, r)
		assert.Equal(t, "localhost:42069, duplicate:420420", r.Headers["host"])
	})

	t.Run("case insensitive headers", func(t *testing.T) {
		reader := kgbuf.NewReader(&testutil.ChunkedReader{
			Data:            "GET / HTTP/1.1\r\nHost: localhost:42069\r\n\r\n",
			NumBytesPerRead: 3,
		})
		r, err := ReadRequest(reader)
		require.NoError(t, err)
		require.NotNil(t, r)
		assert.Equal(t, "localhost:42069", r.Headers["host"])
	})

	t.Run("missing end headers", func(t *testing.T) {
		reader := kgbuf.NewReader(&testutil.ChunkedReader{
			Data:            "GET / HTTP/1.1\r\nHost: localhost:42069",
			NumBytesPerRead: 3,
		})
		_, err := ReadRequest(reader)
		require.Error(t, err)
	})
}

func TestBodyParse(t *testing.T) {
	t.Run("standard body", func(t *testing.T) {
		reader := kgbuf.NewReader(&testutil.ChunkedReader{
			Data: "POST /submit HTTP/1.1\r\n" +
				"Host: localhost:42069\r\n" +
				"Content-Length: 13\r\n" +
				"\r\n" +
				"hello world!\n",
			NumBytesPerRead: 1024,
		})
		r, err := ReadRequest(reader)
		require.NoError(t, err)
		require.NotNil(t, r)
		p := make([]byte, 13)
		_, err = r.Body.Read(p)
		require.NoError(t, err)
		assert.Equal(t, "hello world!\n", string(p))
		n, err := r.Body.Read(p)
		require.Error(t, err)
		require.ErrorIs(t, err, io.EOF)
		assert.Equal(t, 0, n)
	})

	t.Run("empty body with 0 reported content length", func(t *testing.T) {
		reader := kgbuf.NewReader(&testutil.ChunkedReader{
			Data: "POST /submit HTTP/1.1\r\n" +
				"Host: localhost:42069\r\n" +
				"Content-Length: 0\r\n" +
				"\r\n",
			NumBytesPerRead: 3,
		})
		r, err := ReadRequest(reader)
		require.NoError(t, err)
		require.NotNil(t, r)
		p := make([]byte, 4)
		_, err = r.Body.Read(p)
		require.Error(t, err)
		require.ErrorIs(t, err, io.EOF)
	})

	t.Run("body shorter than reported content length", func(t *testing.T) {
		reader := kgbuf.NewReader(&testutil.ChunkedReader{
			Data: "POST /submit HTTP/1.1\r\n" +
				"Host: localhost:42069\r\n" +
				"Content-Length: 30\r\n" +
				"\r\n" +
				"partial content",
			NumBytesPerRead: 1024,
		})
		r, err := ReadRequest(reader)
		require.NoError(t, err)
		p := make([]byte, 15)
		_, err = r.Body.Read(p)
		require.NoError(t, err)
		assert.Equal(t, "partial content", string(p))
		n, err := r.Body.Read(p)
		require.Error(t, err)
		require.ErrorIs(t, err, io.ErrUnexpectedEOF)
		assert.Equal(t, 0, n)
	})

	t.Run("no content length but body exists", func(t *testing.T) {
		reader := kgbuf.NewReader(&testutil.ChunkedReader{
			Data: "POST /submit HTTP/1.1\r\n" +
				"Host: localhost:42069\r\n" +
				"\r\n" +
				"hello world!\n",
			NumBytesPerRead: 3,
		})
		r, err := ReadRequest(reader)
		require.NoError(t, err)
		require.NotNil(t, r)
		p := make([]byte, 4)
		n, err := r.Body.Read(p)
		require.Error(t, err)
		require.ErrorIs(t, err, io.EOF)
		assert.Equal(t, 0, n)
	})
}

func TestReadRequestReadsMultipleRequests(t *testing.T) {
	// Valid: multiple requests
	reader := kgbuf.NewReader(strings.NewReader(
		"POST /one HTTP/1.1\r\n" +
			"Host: localhost:42069\r\n" +
			"Content-Length: 5\r\n" +
			"\r\n" +
			"first" +
			"GET /two HTTP/1.1\r\n" +
			"Host: localhost:42069\r\n" +
			"Content-Length: 0\r\n" +
			"\r\n",
	))

	r, err := ReadRequest(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, "POST", r.Method)
	assert.Equal(t, "/one", r.URL.Path)
	p := make([]byte, 5)
	_, err = r.Body.Read(p)
	require.NoError(t, err)
	assert.Equal(t, "first", string(p))
	n, err := r.Body.Read(p)
	require.Error(t, err)
	require.ErrorIs(t, err, io.EOF)
	assert.Equal(t, 0, n)

	r, err = ReadRequest(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, "GET", r.Method)
	assert.Equal(t, "/two", r.URL.Path)
	p = make([]byte, 4)
	require.NoError(t, err)
	n, err = r.Body.Read(p)
	require.Error(t, err)
	require.ErrorIs(t, err, io.EOF)
	assert.Equal(t, 0, n)
}

func TestReadRequestLimitRequestLine(t *testing.T) {
	// Invalid: Garbage bytes before request line
	reader := kgbuf.NewReader(strings.NewReader(
		makeHugeString(RequestLineLimit, "") +
			"POST /one HTTP/1.1\r\n" +
			"Host: localhost:42069\r\n" +
			"Content-Length: 5\r\n" +
			"\r\n" +
			"first",
	))
	_, err := ReadRequest(reader)
	require.Error(t, err)
}

func makeHugeString(repeat int, delim string) string {
	return strings.Repeat(fmt.Sprintf("a%s", delim), repeat)
}
