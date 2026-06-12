package kghttp

import (
	"testing"

	"strings"

	"github.com/Kaung-HtetKyaw/kgx/internal/testutil"
	"github.com/Kaung-HtetKyaw/kgx/kgbuf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestLineParse(t *testing.T) {
	// Test: Good GET Request line
	reader := kgbuf.NewReader(&testutil.ChunkedReader{
		Data:            "GET / HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n",
		NumBytesPerRead: 3,
	})
	r, err := ReadRequest(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, "GET", r.RequestLine.Method)
	assert.Equal(t, "/", r.RequestLine.RequestTarget)
	assert.Equal(t, "1.1", r.RequestLine.HttpVersion)

	// Test: Good GET Request line with path
	reader = kgbuf.NewReader(&testutil.ChunkedReader{
		Data:            "GET /coffee HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n",
		NumBytesPerRead: 1,
	})
	r, err = ReadRequest(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, "GET", r.RequestLine.Method)
	assert.Equal(t, "/coffee", r.RequestLine.RequestTarget)
	assert.Equal(t, "1.1", r.RequestLine.HttpVersion)

	// Test: Good POST Request with path
	reader = kgbuf.NewReader(&testutil.ChunkedReader{
		Data:            "POST /coffee HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n",
		NumBytesPerRead: 50,
	})
	r, err = ReadRequest(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, "POST", r.RequestLine.Method)
	assert.Equal(t, "/coffee", r.RequestLine.RequestTarget)
	assert.Equal(t, "1.1", r.RequestLine.HttpVersion)

	// Test: Invalid number of parts in request line
	reader = kgbuf.NewReader(&testutil.ChunkedReader{
		Data:            "/coffee HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n",
		NumBytesPerRead: 3,
	})
	_, err = ReadRequest(reader)
	require.Error(t, err)

	// Test: Invalid method (out of order) Request line
	reader = kgbuf.NewReader(&testutil.ChunkedReader{
		Data:            "/coffee POST HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n",
		NumBytesPerRead: 3,
	})
	_, err = ReadRequest(reader)
	require.Error(t, err)

	// Test: Invalid version in Request line
	reader = kgbuf.NewReader(&testutil.ChunkedReader{
		Data:            "OPTIONS /prime/rib TCP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n",
		NumBytesPerRead: 50,
	})
	_, err = ReadRequest(reader)
	require.Error(t, err)

}

func TestHeadersParse(t *testing.T) {
	// Test: Standard Headers
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

	// Test: Malformed Header
	reader = kgbuf.NewReader(&testutil.ChunkedReader{
		Data:            "GET / HTTP/1.1\r\nHost localhost:42069\r\n\r\n",
		NumBytesPerRead: 3,
	})
	r, err = ReadRequest(reader)
	require.Error(t, err)

	// Test: Empty Header
	reader = kgbuf.NewReader(&testutil.ChunkedReader{
		Data:            "GET / HTTP/1.1\r\n\r\n",
		NumBytesPerRead: 3,
	})
	r, err = ReadRequest(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Empty(t, r.Headers)

	// Test: Duplicate Headers
	reader = kgbuf.NewReader(&testutil.ChunkedReader{
		Data:            "GET / HTTP/1.1\r\nhost: localhost:42069\r\nhost: duplicate:420420\r\n\r\n",
		NumBytesPerRead: 3,
	})
	r, err = ReadRequest(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, "localhost:42069, duplicate:420420", r.Headers["host"])

	// Test: Case Insentitive Headers
	reader = kgbuf.NewReader(&testutil.ChunkedReader{
		Data:            "GET / HTTP/1.1\r\nHost: localhost:42069\r\n\r\n",
		NumBytesPerRead: 3,
	})
	r, err = ReadRequest(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, "localhost:42069", r.Headers["host"])

	// Test: Missing end Headers
	reader = kgbuf.NewReader(&testutil.ChunkedReader{
		Data:            "GET / HTTP/1.1\r\nHost: localhost:42069",
		NumBytesPerRead: 3,
	})
	r, err = ReadRequest(reader)
	require.Error(t, err)
}

func TestBodyParse(t *testing.T) {
	// Test: Standard Body
	reader := kgbuf.NewReader(&testutil.ChunkedReader{
		Data: "POST /submit HTTP/1.1\r\n" +
			"Host: localhost:42069\r\n" +
			"Content-Length: 13\r\n" +
			"\r\n" +
			"hello world!\n",
		NumBytesPerRead: 3,
	})
	r, err := ReadRequest(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, "hello world!\n", string(r.Body))

	// Test: Empty Body, 0 reported content length
	reader = kgbuf.NewReader(&testutil.ChunkedReader{
		Data: "POST /submit HTTP/1.1\r\n" +
			"Host: localhost:42069\r\n" +
			"Content-Length: 0\r\n" +
			"\r\n",
		NumBytesPerRead: 3,
	})
	r, err = ReadRequest(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, "", string(r.Body))

	// Test: Body shorter than reported content length
	reader = kgbuf.NewReader(&testutil.ChunkedReader{
		Data: "POST /submit HTTP/1.1\r\n" +
			"Host: localhost:42069\r\n" +
			"Content-Length: 20\r\n" +
			"\r\n" +
			"partial content",
		NumBytesPerRead: 3,
	})
	r, err = ReadRequest(reader)
	require.Error(t, err)

	// Test: No Content-Length but Body Exists
	reader = kgbuf.NewReader(&testutil.ChunkedReader{
		Data: "POST /submit HTTP/1.1\r\n" +
			"Host: localhost:42069\r\n" +
			"\r\n" +
			"hello world!\n",
		NumBytesPerRead: 3,
	})
	r, err = ReadRequest(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, "", string(r.Body))
}

func TestReadRequestReadsMultipleRequests(t *testing.T) {
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
	assert.Equal(t, "POST", r.RequestLine.Method)
	assert.Equal(t, "/one", r.RequestLine.RequestTarget)
	assert.Equal(t, "first", string(r.Body))

	r, err = ReadRequest(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, "GET", r.RequestLine.Method)
	assert.Equal(t, "/two", r.RequestLine.RequestTarget)
	assert.Equal(t, "", string(r.Body))
}
