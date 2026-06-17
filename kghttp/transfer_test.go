package kghttp

import (
	"io"
	"testing"

	"github.com/Kaung-HtetKyaw/kgx/internal/testutil"
	"github.com/Kaung-HtetKyaw/kgx/kghttp/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBodyReaderRead(t *testing.T) {
	// Valid: Defined content length
	r := &testutil.ChunkedReader{
		Data: "hello world" +
			"GET / HTTP/1.1\r\n" +
			"Content-Length: 0\r\n\r\n",
		NumBytesPerRead: 8,
	}

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
	req.Body = &bodyReader{
		src:     io.LimitReader(r, 11),
		msg:     req,
		chunked: true,
	}

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
	r = &testutil.ChunkedReader{
		Data: "GET / HTTP/1.1\r\n" +
			"Content-Length: 0\r\n\r\n",
		NumBytesPerRead: 8,
	}

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
	req.Body = &internal.NoBody
	p = make([]byte, 8)
	n, err = req.Body.Read(p)
	require.Error(t, err)
	require.ErrorIs(t, err, io.EOF)
	assert.Equal(t, 0, n)

	// Valid: Chunked body without trailers
	r = &testutil.ChunkedReader{
		Data: "5\r\n" +
			"hello\r\n" +
			"6\r\n" +
			" world\r\n" +
			"0\r\n" + "\r\n" +
			"GET / HTTP/1.1\r\n" +
			"Content-Length: 0\r\n\r\n",
		NumBytesPerRead: 8,
	}

	req = &Request{
		RequestLine: RequestLine{
			Method:        "POST",
			RequestTarget: "/message",
			HttpVersion:   "HTTP/1.1",
		},
		Headers: Headers{
			"content-length": "11",
		},
	}
	cr := internal.NewChunkedReader(r)
	req.Body = &bodyReader{
		src:     cr,
		msg:     req,
		chunked: true,
	}
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
	r = &testutil.ChunkedReader{
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
	}

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
	cr = internal.NewChunkedReader(r)
	req.Body = &bodyReader{
		src: cr,
		msg: req,
	}
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
	trailer, ok := req.Trailers.Get("x-checksum")
	assert.True(t, ok)
	assert.Equal(t, "abcdefg", trailer)
}
