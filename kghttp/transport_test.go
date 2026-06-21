package kghttp

import (
	"io"
	"strconv"
	"testing"
	"time"

	"github.com/Kaung-HtetKyaw/kgx/kgurl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransportRoundTrip(t *testing.T) {
	// Valid: GET request
	server := &Server{
		Addr: "127.0.0.1:0",
		Handler: func(w *ResponseWriter, req *Request) {
			body := "Hello World"
			data := []byte(body)
			w.Headers().Set("content-length", strconv.Itoa(len(data)))
			w.Headers().Set("content-type", "text/plain")
			w.WriteHeaders(200)
			w.Write(data)
		},
		IdleConnTimeOut: 5 * time.Second,
	}
	err := server.ListenAndServe()
	require.NoError(t, err)
	defer server.Close()

	addr := server.listener.Addr().String()
	url, err := kgurl.Parse("http://" + addr)
	require.NoError(t, err)
	req := &Request{
		Method:     "GET",
		Proto:      "HTTP",
		ProtoMajor: 1,
		ProtoMinor: 1,
		URL:        url,
		Headers: Headers{
			"host": addr,
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
