package kgbuf

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriterWrite(t *testing.T) {
	t.Run("underlying writer receives the data", func(t *testing.T) {
		ds := &bytes.Buffer{}
		w := NewWriterSize(ds, 11)
		_, err := w.Write([]byte("hello world"))
		require.NoError(t, err)
		err = w.Flush()
		require.NoError(t, err)
		assert.Equal(t, "hello world", ds.String())
	})

	t.Run("no data if not flushed", func(t *testing.T) {
		ds := &bytes.Buffer{}
		w := NewWriter(ds)
		_, err := w.Write([]byte("hello world"))
		require.NoError(t, err)
		assert.Equal(t, "", ds.String())
	})

	t.Run("large data does not need flushing", func(t *testing.T) {
		ds := &bytes.Buffer{}
		w := NewWriterSize(ds, 10)
		_, err := w.Write([]byte("hello world"))
		require.NoError(t, err)
		assert.Equal(t, "hello world", ds.String())

		err = w.Flush()
		require.NoError(t, err)
	})
}

func TestWriterWriteFlushesBytesCopiedToFillBuffer(t *testing.T) {
	ds := &bytes.Buffer{}
	w := NewWriterSize(ds, 10)

	n, err := w.Write([]byte("1234567"))
	require.NoError(t, err)
	assert.Equal(t, 7, n)

	n, err = w.Write([]byte("abcdef"))
	require.NoError(t, err)
	assert.Equal(t, 6, n)

	err = w.Flush()
	require.NoError(t, err)
	assert.Equal(t, "1234567abcdef", ds.String())
}
