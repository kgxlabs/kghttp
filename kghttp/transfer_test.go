package kghttp

import (
	"io"
	"testing"

	"github.com/Kaung-HtetKyaw/kgx/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBodyReaderRead(t *testing.T) {
	// Valid: Defined content length
	cr := &testutil.ChunkedReader{
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
		src: io.LimitReader(cr, 11),
		msg: req,
	}

	p := make([]byte, 12)
	n, err := req.Body.Read(p)
	require.NoError(t, err)
	assert.Equal(t, 11, n)
	assert.Equal(t, "hello world\x00", string(p))
	n, err = req.Body.Read(p)
	require.Error(t, err)

	// Valid: Reading no body
	cr = &testutil.ChunkedReader{
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
	req.Body = &NoBody
	p = make([]byte, 8)
	n, err = req.Body.Read(p)
	require.Error(t, err)
	assert.Equal(t, 0, n)

	// Valid: Chunked body without trailers

	// Valid: Chunked body with trailers
}
