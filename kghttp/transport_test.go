package kghttp

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestTransportRequestSerialization(t *testing.T) {
	// Valid: request with no body
	requestLine := &RequestLine{
		HttpVersion:   "HTTP/1.1",
		Method:        "GET",
		RequestTarget: "/",
	}
	headers := &Headers{
		"host": "example.com",
	}
	req := &Request{
		RequestLine: *requestLine,
		Headers:     *headers,
	}
	m, err := serializeRequest(req)
	require.NoError(t, err)
	assert.Contains(t, m, "GET / HTTP/1.1\r\n")
	assert.Contains(t, m, "host: example\r\n\r\n")

	// Valid: request with body
	requestLine = &RequestLine{
		HttpVersion:   "HTTP/1.1",
		Method:        "POST",
		RequestTarget: "/message",
	}
	headers = &Headers{
		"content-type":   "text/plain",
		"content-length": "12",
	}
	req = &Request{
		RequestLine: *requestLine,
		Headers:     *headers,
	}
	m, err = serializeRequest(req)
	require.NoError(t, err)
	assert.Contains(t, m, "POST /message HTTP/1.1\r\n")
	assert.Contains(t, m, "content-type: text/plain\r\n")
	assert.Contains(t, m, "content-length: 12\r\n")
	assert.Contains(t, m, "message body")

	// Invalid: Incorrect HTTP request line
	requestLine = &RequestLine{
		HttpVersion:   "HTTP /1.1",
		Method:        "TEST",
		RequestTarget: "/",
	}
	req = &Request{
		RequestLine: *requestLine,
	}
	m, err = serializeRequest(req)
	require.Error(t, err)

	// Invalid: Having body without headers
	requestLine = &RequestLine{
		HttpVersion:   "HTTP/1.1",
		Method:        "POST",
		RequestTarget: "/",
	}
	req = &Request{
		RequestLine: *requestLine,
		Body:        []byte("message body"),
	}
	m, err = serializeRequest(req)
	require.Error(t, err)

}
