package kgbuf

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Kaung-HtetKyaw/kgx/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReaderReadBytes(t *testing.T) {
	// Valid: Read String matches
	reader := newTestReader("hello world\nnice to meet you\n", 8)
	line, err := reader.ReadBytes([]byte("\n"))
	require.NoError(t, err)
	assert.Equal(t, "hello world\n", string(line))

	// Valid: Read string consumes advances to new line
	reader = newTestReader("hello world\nnice to meet you\nwelcome", 8)
	line, err = reader.ReadBytes([]byte("\n"))
	require.NoError(t, err)
	assert.Equal(t, "hello world\n", string(line))
	line, err = reader.ReadBytes([]byte("\n"))
	require.NoError(t, err)
	assert.Equal(t, "nice to meet you\n", string(line))
	line, err = reader.ReadBytes([]byte("\n"))
	require.NoError(t, err)
	assert.Equal(t, "", string(line))

	// Valid: No delim found
	reader = newTestReader("hello world. nice to meet you.", 8)
	line, err = reader.ReadBytes([]byte("\n"))
	require.NoError(t, err)
	assert.Equal(t, "", string(line))

	// Valid: Grow buffer if needed
	s := makeHugeString(1024, "")
	reader = newTestReader(fmt.Sprintf("%s\n", s), 1024)
	line, err = reader.ReadBytes([]byte("\n"))
	require.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("%s\n", s), string(line))
}

func TestReaderRead(t *testing.T) {
	// Valid: String matches
	reader := newTestReader("hello world", 8)
	p := make([]byte, 5)
	n, err := reader.Read(p)
	require.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Equal(t, "hello", string(p))

	// Valid: Read until buffer is empty
	reader = newTestReader("hello world", 8)
	p = make([]byte, 5)
	n, err = reader.Read(p)
	require.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Equal(t, "hello", string(p))
	n, err = reader.Read(p)
	require.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Equal(t, " worl", string(p))
	n, err = reader.Read(p)
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	assert.Equal(t, "dworl", string(p))
	n, err = reader.Read(p)
	require.NoError(t, err)
	assert.Equal(t, 0, n)
	assert.Equal(t, "dworl", string(p))

	// Valid: Read empty buffer
	reader = newTestReader("", 8)
	p = make([]byte, 0)
	n, err = reader.Read(p)
	require.NoError(t, err)
	assert.Equal(t, 0, n)
	assert.Equal(t, "", string(p))

	// Valid: Grow buffer if needed to fill the byte slice
	s := makeHugeString(32768, "")
	reader = newTestReader(s, 4096)
	p = make([]byte, 32768)
	n, err = reader.Read(p)
	require.NoError(t, err)
	assert.Equal(t, 32768, n)
	assert.Equal(t, s, string(p))
}

// ReadString is literally a wrapper around ReadBytes with no logic of it's own
// Add more tests if that is changed in the future
func TestReaderReadString(t *testing.T) {
	// Valid: String matches
	reader := newTestReader("hello world", 8)
	p := make([]byte, 5)
	n, err := reader.Read(p)
	require.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Equal(t, "hello", string(p))
}

func TestReaderSizeCap(t *testing.T) {
	// Valid: Empty buffer with capacity
	reader := newTestReader("", 8)
	assert.Equal(t, 0, reader.Buffered())
	assert.Equal(t, reader.defaultSize, reader.Size())

	// Valid: buffer with content
	reader = newTestReader("hello world", 11)
	_, err := reader.Peek(11)
	require.NoError(t, err)
	assert.Equal(t, 11, reader.Buffered())
	assert.Equal(t, reader.defaultSize, reader.Size())

}

func TestReaderPeek(t *testing.T) {
	// Valid: Specified n is available
	reader := newTestReader("hello world", 8)
	b, err := reader.Peek(5)
	require.NoError(t, err)
	assert.Equal(t, "hello", string(b))
	assert.Equal(t, 5, reader.Buffered())
	b, err = reader.Peek(6)
	require.NoError(t, err)
	assert.Equal(t, " world", string(b))
	assert.Equal(t, 11, reader.Buffered())
	b, err = reader.Peek(5)
	require.Error(t, err)
	assert.Equal(t, 0, len(b))

	// Valid: Peek zero byte
	reader = newTestReader("hello world", 8)
	b, err = reader.Peek(0)
	require.NoError(t, err)
	assert.Equal(t, 0, len(b))

	// Invalid: Only partial data is available
	reader = newTestReader("hello world", 5)
	b, err = reader.Peek(5)
	require.NoError(t, err)
	assert.Equal(t, "hello", string(b))
	assert.Equal(t, 5, reader.Buffered())
	b, err = reader.Peek(5)
	require.NoError(t, err)
	assert.Equal(t, " worl", string(b))
	assert.Equal(t, 10, reader.Buffered())
	b, err = reader.Peek(5)
	require.Error(t, err)
	assert.Equal(t, 1, len(b))

	// Invalid: No data is available
	reader = newTestReader("", 5)
	b, err = reader.Peek(5)
	require.Error(t, err)
	assert.Equal(t, 0, len(b))
}

func makeHugeString(repeat int, delim string) string {
	return strings.Repeat(fmt.Sprintf("a%s", delim), repeat)
}

func newTestReader(data string, n int) *Reader {
	cr := &testutil.ChunkedReader{
		Data:            data,
		NumBytesPerRead: n,
	}
	return NewReader(cr)
}
