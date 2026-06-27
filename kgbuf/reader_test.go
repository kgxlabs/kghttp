package kgbuf

import (
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/kgxlabs/kghttp/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReaderReadBytes(t *testing.T) {
	t.Run("read string matches", func(t *testing.T) {
		reader := newTestReader("hello world\nnice to meet you\n", 8)
		line, err := reader.ReadBytes([]byte("\n"))
		require.NoError(t, err)
		assert.Equal(t, "hello world\n", string(line))
	})

	t.Run("read string consumes and advances to new line", func(t *testing.T) {
		reader := newTestReader("hello world\nnice to meet you\nwelcome", 8)
		line, err := reader.ReadBytes([]byte("\n"))
		require.NoError(t, err)
		assert.Equal(t, "hello world\n", string(line))
		line, err = reader.ReadBytes([]byte("\n"))
		require.NoError(t, err)
		assert.Equal(t, "nice to meet you\n", string(line))
		line, err = reader.ReadBytes([]byte("\n"))
		require.Error(t, err)
		require.ErrorIs(t, err, io.EOF)
		assert.Equal(t, "welcome", string(line))
	})

	t.Run("no delim found", func(t *testing.T) {
		reader := newTestReader("hello world. nice to meet you.", 8)
		line, err := reader.ReadBytes([]byte("\n"))
		require.Error(t, err)
		require.ErrorIs(t, err, io.EOF)
		assert.Equal(t, "hello world. nice to meet you.", string(line))
	})

	t.Run("grow buffer if needed", func(t *testing.T) {
		s := makeHugeString(1024, "")
		reader := newTestReader(fmt.Sprintf("%s\n", s), 1024)
		line, err := reader.ReadBytes([]byte("\n"))
		require.NoError(t, err)
		assert.Equal(t, fmt.Sprintf("%s\n", s), string(line))
	})

	// Valid: Overwrite buffer later
	// TODO: Figure out how to add test case for overwritting already read byte slices
}

func TestReaderReadSlice(t *testing.T) {
	t.Run("read string matches", func(t *testing.T) {
		reader := newTestReader("hello world\nnice to meet you\n", 8)
		line, err := reader.ReadSlice('\n')
		require.NoError(t, err)
		assert.Equal(t, "hello world\n", string(line))
	})

	t.Run("read string consumes and advances to new line", func(t *testing.T) {
		reader := newTestReader("hello world\nnice to meet you\nwelcome", 8)
		line, err := reader.ReadSlice('\n')
		require.NoError(t, err)
		assert.Equal(t, "hello world\n", string(line))
		line, err = reader.ReadSlice('\n')
		require.NoError(t, err)
		assert.Equal(t, "nice to meet you\n", string(line))
		line, err = reader.ReadSlice('\n')
		require.NoError(t, err)
		assert.Equal(t, "", string(line))
	})

	t.Run("no delim found but buffer still has space", func(t *testing.T) {
		reader := newTestReader("hello world. nice to meet you.", 8)
		line, err := reader.ReadSlice('\n')
		require.NoError(t, err)
		assert.Equal(t, "", string(line))
	})

	t.Run("no delim found and buffer fills", func(t *testing.T) {
		s := makeHugeString(readerDefaultBufferSize+16, "")
		reader := newTestReader(fmt.Sprintf("%s\n", s), 1024)
		line, err := reader.ReadSlice('\n')
		require.Error(t, err)
		require.ErrorIs(t, err, ErrBufferFull)
		assert.Equal(t, reader.size, len(line))
	})
}

func TestReaderRead(t *testing.T) {
	// Valid: Read gives exactly how many underlying reader can give
	reader := newTestReader("hello world", 8)
	p := make([]byte, 11)
	n, err := reader.Read(p)
	require.NoError(t, err)
	assert.Equal(t, 8, n)
	assert.Equal(t, "hello wo\x00\x00\x00", string(p))
	n, err = reader.Read(p[8:])
	require.NoError(t, err)
	assert.Equal(t, 3, n)
	assert.Equal(t, "hello world", string(p))
	n, err = reader.Read(p)
	require.Error(t, err)
	assert.Equal(t, 0, n)
	assert.Equal(t, "hello world", string(p))
}

func TestReaderReadFull(t *testing.T) {
	t.Run("string matches", func(t *testing.T) {
		reader := newTestReader("hello world", 8)
		p := make([]byte, 5)
		n, err := reader.ReadFull(p)
		require.NoError(t, err)
		assert.Equal(t, 5, n)
		assert.Equal(t, "hello", string(p))
	})

	t.Run("read until byte slice is full", func(t *testing.T) {
		reader := newTestReader("hello world", 8)
		p := make([]byte, 11)
		n, err := reader.ReadFull(p)
		require.NoError(t, err)
		assert.Equal(t, 11, n)
		assert.Equal(t, "hello world", string(p))
	})

	t.Run("read empty buffer", func(t *testing.T) {
		reader := newTestReader("", 8)
		p := make([]byte, 0)
		n, err := reader.ReadFull(p)
		require.NoError(t, err)
		assert.Equal(t, 0, n)
		assert.Equal(t, "", string(p))
	})

	t.Run("grow buffer if needed to fill the byte slice", func(t *testing.T) {
		s := makeHugeString(32768, "")
		reader := newTestReader(s, 4096)
		p := make([]byte, 32768)
		n, err := reader.ReadFull(p)
		require.NoError(t, err)
		assert.Equal(t, 32768, n)
		assert.Equal(t, s, string(p))
	})

	t.Run("reach end before input is filled", func(t *testing.T) {
		reader := newTestReader("hello world", 8)
		p := make([]byte, 12)
		n, err := reader.ReadFull(p)
		require.Error(t, err)
		assert.Equal(t, 11, n)
		assert.Equal(t, "hello world\x00", string(p))
	})
}

// ReadString is literally a wrapper around ReadBytes with no logic of it's own
// Add more tests if that is changed in the future
func TestReaderReadString(t *testing.T) {
	// Valid: String matches
	reader := newTestReader("hello world", 8)
	p := make([]byte, 5)
	n, err := reader.ReadFull(p)
	require.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Equal(t, "hello", string(p))
}

func TestReaderSizeCap(t *testing.T) {
	t.Run("empty buffer with capacity", func(t *testing.T) {
		reader := newTestReader("", 8)
		assert.Equal(t, 0, reader.Buffered())
		assert.Equal(t, reader.size, reader.Size())
	})

	t.Run("buffer with content", func(t *testing.T) {
		reader := newTestReader("hello world", 11)
		_, err := reader.Peek(11)
		require.NoError(t, err)
		assert.Equal(t, 11, reader.Buffered())
		assert.Equal(t, reader.size, reader.Size())
	})
}

func TestReaderPeek(t *testing.T) {
	t.Run("specified n is available", func(t *testing.T) {
		reader := newTestReader("hello world", 8)
		b, err := reader.Peek(5)
		require.NoError(t, err)
		assert.Equal(t, "hello", string(b))
		assert.Equal(t, 5, reader.Buffered())
		b, err = reader.Peek(6)
		require.NoError(t, err)
		assert.Equal(t, "hello ", string(b))
		assert.Equal(t, 6, reader.Buffered())
		b, err = reader.Peek(12)
		require.Error(t, err)
		assert.Equal(t, 11, len(b))
	})

	t.Run("peek zero byte", func(t *testing.T) {
		reader := newTestReader("hello world", 8)
		b, err := reader.Peek(0)
		require.NoError(t, err)
		assert.Equal(t, 0, len(b))
	})

	t.Run("only partial data is available", func(t *testing.T) {
		reader := newTestReader("hello world", 5)
		b, err := reader.Peek(5)
		require.NoError(t, err)
		assert.Equal(t, "hello", string(b))
		assert.Equal(t, 5, reader.Buffered())
		b, err = reader.Peek(5)
		require.NoError(t, err)
		assert.Equal(t, "hello", string(b))
		assert.Equal(t, 5, reader.Buffered())
		b, err = reader.Peek(12)
		require.Error(t, err)
		assert.Equal(t, 11, len(b))
	})

	t.Run("no data is available", func(t *testing.T) {
		reader := newTestReader("", 5)
		b, err := reader.Peek(5)
		require.Error(t, err)
		assert.Equal(t, 0, len(b))
	})
}

func TestReaderReadBytesLimit(t *testing.T) {
	t.Run("only read specified limit", func(t *testing.T) {
		reader := newTestReader("partial read! Ignore the rest", 8)
		b, err := reader.ReadBytesLimit([]byte("!"), 20)
		require.NoError(t, err)
		assert.Equal(t, "partial read!", string(b))
		assert.LessOrEqual(t, len(b), 20)
	})

	t.Run("exceed limit", func(t *testing.T) {
		reader := newTestReader("there is no delimiter for this sentence", 8)
		_, err := reader.ReadBytesLimit([]byte("\n"), 20)
		require.Error(t, err)
	})
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
