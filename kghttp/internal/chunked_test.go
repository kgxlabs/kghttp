package internal

import (
	"bytes"
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

func TestCunkedWriterWrite(t *testing.T) {
	// Valid: Write single complete chunk
	ds := &bytes.Buffer{}
	w := kgbuf.NewWriter(ds)
	cw := &chunkedWriter{
		w: w,
	}
	n, err := cw.Write([]byte("hello world"))
	require.NoError(t, err)
	assert.Equal(t, 11, n)
	assert.Equal(t, "11\r\nhello world\r\n", ds.String())
	err = cw.Close()
	require.NoError(t, err)
	assert.Equal(t, "11\r\nhello world\r\n0\r\n\r\n", ds.String())

	// Valid: Write multiple complete chunks
	ds = &bytes.Buffer{}
	w = kgbuf.NewWriter(ds)
	cw = &chunkedWriter{
		w: w,
	}
	n, err = cw.Write([]byte("hello "))
	require.NoError(t, err)
	assert.Equal(t, 6, n)
	assert.Equal(t, "6\r\nhello \r\n", ds.String())
	n, err = cw.Write([]byte("world"))
	require.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Equal(t, "6\r\nhello \r\n5\r\nworld\r\n", ds.String())
	err = cw.Close()
	require.NoError(t, err)
	assert.Equal(t, "6\r\nhello \r\n5\r\nworld\r\n0\r\n\r\n", ds.String())

	// Valid: Zero length data should do nothing
	ds = &bytes.Buffer{}
	w = kgbuf.NewWriter(ds)
	cw = &chunkedWriter{
		w: w,
	}
	n, err = cw.Write([]byte(""))
	require.NoError(t, err)
	assert.Equal(t, 0, n)
	assert.Equal(t, "", ds.String())
	err = cw.Close()
	require.NoError(t, err)
	assert.Equal(t, "0\r\n\r\n", ds.String())

	// Valid: Write chunk with trailers
	ds = &bytes.Buffer{}
	w = kgbuf.NewWriter(ds)
	cw = &chunkedWriter{
		w: w,
		writeTrailers: func(w io.Writer) error {
			w.Write([]byte("x-hello: true\r\n"))
			return nil
		},
	}
	n, err = cw.Write([]byte("hello"))
	require.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Equal(t, "5\r\nhello\r\n", ds.String())
	err = cw.Close()
	require.NoError(t, err)
	assert.Equal(t, "5\r\nhello\r\n0\r\nx-hello: true\r\n\r\n", ds.String())

	// Valid: Write chunk without trailers
	ds = &bytes.Buffer{}
	w = kgbuf.NewWriter(ds)
	cw = &chunkedWriter{
		w: w,
	}
	n, err = cw.Write([]byte("hello"))
	require.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Equal(t, "5\r\nhello\r\n", ds.String())
	err = cw.Close()
	require.NoError(t, err)
	assert.Equal(t, "5\r\nhello\r\n0\r\n\r\n", ds.String())

}
