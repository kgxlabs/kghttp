package kghttp

import (
	"io"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/Kaung-HtetKyaw/kgx/kgurl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransportRoundTrip(t *testing.T) {
	// Valid: GET request
	ln, err := net.Listen("tcp", "127.0.0.1:80")
	require.NoError(t, err)

	server := &Server{
		Handler: func(w *ResponseWriter, req *Request) {
			body := "Hello World"
			data := []byte(body)
			w.Headers().Set("content-length", strconv.Itoa(len(data)))
			w.Headers().Set("content-type", "text/plain")
			w.WriteHeaders(200)
			w.WriteBody(data)

		},
		IdleConnTimeOut: 5 * time.Second,
	}
	err = server.Serve(ln)
	require.NoError(t, err)
	defer server.Close()

	url, err := kgurl.Parse("http://localhost")
	require.NoError(t, err)
	req := &Request{
		Method:     "GET",
		Proto:      "HTTP",
		ProtoMajor: 1,
		ProtoMinor: 1,
		URL:        url,
		Headers: Headers{
			"host": "localhost",
		},
	}

	transport := NewTransport()
	resp, err := transport.RoundTrip(req)
	require.NoError(t, err)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, "Hello World", string(body))

	// Valid: POST request declared content-length

	// Valid: POST request chunked encoding

	// Invalid: Body shorter than declared content-length

	// Invalid: Malformed chunked encoding
}
