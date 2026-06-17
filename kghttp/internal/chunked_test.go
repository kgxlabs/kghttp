package internal

import (
	"io"
	"strings"
	"testing"

	"github.com/Kaung-HtetKyaw/kgx/kgbuf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChunkedBodyReaderRead(t *testing.T) {
	// Valid: Read one chunk
	r := kgbuf.NewReader(
		strings.NewReader("5\r\n" +
			"hello\r\n" +
			"5\r\n" +
			"world\r\n" +
			"0\r\n" + "\r\n" +
			"GET / HTTP/1.1\r\n" +
			"Content-Length: 0\r\n\r\n"),
	)
	cr := NewChunkedReader(r)
	p := make([]byte, 5)
	_, err := cr.Read(p)
	require.NoError(t, err)
	assert.Equal(t, "hello", string(p))

	// Valid: Unfinished content from a chunk
	r = kgbuf.NewReader(
		strings.NewReader("6\r\n" +
			"hello,\r\n" +
			"5\r\n" +
			"world\r\n" +
			"0\r\n" + "\r\n" +
			"GET / HTTP/1.1\r\n" +
			"Content-Length: 0\r\n\r\n"),
	)
	cr = NewChunkedReader(r)
	p = make([]byte, 5)
	_, err = cr.Read(p)
	require.NoError(t, err)
	assert.Equal(t, "hello", string(p))
	_, err = cr.Read(p)
	require.NoError(t, err)
	assert.Equal(t, ",ello", string(p))
	n, err := cr.Read(p)
	require.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Equal(t, "world", string(p))
	n, err = cr.Read(p)
	require.Error(t, err)
	require.ErrorIs(t, err, io.EOF)
	assert.Equal(t, 0, n)

	// Invalid: Invalid byte in chunked length
	r = kgbuf.NewReader(
		strings.NewReader(
			"GET / HTTP/1.1\r\n" +
				"Content-Length: 0\r\n\r\n"),
	)
	cr = NewChunkedReader(r)
	p = make([]byte, 5)
	n, err = cr.Read(p)
	require.Error(t, err)
	assert.Equal(t, 0, n)
}
