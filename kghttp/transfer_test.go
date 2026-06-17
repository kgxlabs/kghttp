package kghttp

import (
	"io"
	"testing"

	"github.com/Kaung-HtetKyaw/kgx/internal/testutil"
	"github.com/Kaung-HtetKyaw/kgx/kgbuf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBodyReaderRead(t *testing.T) {
	// Valid: Defined content length
	r := kgbuf.NewReader(&testutil.ChunkedReader{
		Data: "hello world" +
			"GET / HTTP/1.1\r\n" +
			"Content-Length: 0\r\n\r\n",
		NumBytesPerRead: 8,
	})

	req := &Request{
		RequestLine: RequestLine{
			Method:        "POST",
			RequestTarget: "/message",
			HttpVersion:   "HTTP/1.1",
		},
		Headers: Headers{
			"content-length": "11",
		},
	}
	err := readTransfer(req, r)
	require.NoError(t, err)
	p := make([]byte, 8)
	n, err := req.Body.Read(p)
	require.NoError(t, err)
	assert.Equal(t, 8, n)
	assert.Equal(t, "hello wo", string(p))
	n, err = req.Body.Read(p)
	require.NoError(t, err)
	assert.Equal(t, "rldlo wo", string(p))
	n, err = req.Body.Read(p)
	require.Error(t, err)
	require.ErrorIs(t, err, io.EOF)
	assert.Equal(t, 0, n)

	// Valid: Reading no body
	r = kgbuf.NewReader(&testutil.ChunkedReader{
		Data: "GET / HTTP/1.1\r\n" +
			"Content-Length: 0\r\n\r\n",
		NumBytesPerRead: 8,
	})

	req = &Request{
		RequestLine: RequestLine{
			Method:        "GET",
			RequestTarget: "/message",
			HttpVersion:   "HTTP/1.1",
		},
		Headers: Headers{
			"content-length": "0",
		},
	}
	err = readTransfer(req, r)
	p = make([]byte, 8)
	n, err = req.Body.Read(p)
	require.Error(t, err)
	require.ErrorIs(t, err, io.EOF)
	assert.Equal(t, 0, n)

	// Valid: Chunked body without trailers
	r = kgbuf.NewReader(&testutil.ChunkedReader{
		Data: "5\r\n" +
			"hello\r\n" +
			"6\r\n" +
			" world\r\n" +
			"0\r\n" + "\r\n" +
			"GET / HTTP/1.1\r\n" +
			"Content-Length: 0\r\n\r\n",
		NumBytesPerRead: 8,
	})

	req = &Request{
		RequestLine: RequestLine{
			Method:        "POST",
			RequestTarget: "/message",
			HttpVersion:   "HTTP/1.1",
		},
		Headers: Headers{
			"transfer-encoding": "chunked",
		},
	}
	err = readTransfer(req, r)
	require.NoError(t, err)
	p = make([]byte, 5)
	_, err = req.Body.Read(p)
	require.NoError(t, err)
	assert.Equal(t, "hello", string(p))
	_, err = req.Body.Read(p)
	require.NoError(t, err)
	assert.Equal(t, " worl", string(p))
	_, err = req.Body.Read(p)
	require.NoError(t, err)
	assert.Equal(t, "dworl", string(p))
	n, err = req.Body.Read(p)
	require.Error(t, err)
	require.ErrorIs(t, err, io.EOF)
	assert.Equal(t, 0, n)

	// Valid: Chunked body with trailers
	r = kgbuf.NewReader(&testutil.ChunkedReader{
		Data: "5\r\n" +
			"hello\r\n" +
			"6\r\n" +
			" world\r\n" +
			"0\r\n" +
			"X-Checksum: abcdefg\r\n" +
			"\r\n" +
			"GET / HTTP/1.1\r\n" +
			"Content-Length: 0\r\n\r\n",
		NumBytesPerRead: 8,
	})

	req = &Request{
		RequestLine: RequestLine{
			Method:        "POST",
			RequestTarget: "/message",
			HttpVersion:   "HTTP/1.1",
		},
		Headers: Headers{
			"transfer-encoding": "chunked",
		},
	}
	err = readTransfer(req, r)
	p = make([]byte, 5)
	_, err = req.Body.Read(p)
	require.NoError(t, err)
	assert.Equal(t, "hello", string(p))
	_, err = req.Body.Read(p)
	require.NoError(t, err)
	assert.Equal(t, " worl", string(p))
	_, err = req.Body.Read(p)
	require.NoError(t, err)
	assert.Equal(t, "dworl", string(p))
	n, err = req.Body.Read(p)
	require.Error(t, err)
	require.ErrorIs(t, err, io.EOF)
	assert.Equal(t, 0, n)
	trailer, ok := req.Trailers.Get("x-checksum")
	assert.True(t, ok)
	assert.Equal(t, "abcdefg", trailer)
}
