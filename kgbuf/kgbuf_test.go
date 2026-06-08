package kgbuf

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Kaung-HtetKyaw/kgx/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReaderReadString(t *testing.T) {
	// Valid: Read String matches
	reader := newTestReader("hello world\nnice to meet you\n", 8)
	line, err := reader.ReadString("\n")
	require.NoError(t, err)
	assert.Equal(t, "hello world\n", line)

	// Valid: Read string consumes advances to new line
	reader = newTestReader("hello world\nnice to meet you\nwelcome", 8)
	line, err = reader.ReadString("\n")
	require.NoError(t, err)
	assert.Equal(t, "hello world\n", line)
	line, err = reader.ReadString("\n")
	require.NoError(t, err)
	assert.Equal(t, "nice to meet you\n", line)
	line, err = reader.ReadString("\n")
	require.NoError(t, err)
	assert.Equal(t, "", line)

	// Valid: No delim found
	reader = newTestReader("hello world. nice to meet you.", 8)
	line, err = reader.ReadString("\n")
	require.NoError(t, err)
	assert.Equal(t, "", line)

	// Valid: Grow buffer if needed
	s := makeHugeString(1000, "")
	reader = newTestReader(fmt.Sprintf("%s\n", s), 1024)
	line, err = reader.ReadString("\n")
	require.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("%s\n", s), line)

}

func makeHugeString(repeat int, delim string) string {
	return strings.Repeat(fmt.Sprintf("abcdefghijklmnopqrstuvwxyz%s", delim), repeat)
}

func newTestReader(data string, n int) *Reader {
	cr := &testutil.ChunkedReader{
		Data:            data,
		NumBytesPerRead: n,
	}
	return NewReader(cr)
}
