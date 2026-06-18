package kgbuf

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriterWrite(t *testing.T) {
	// Valid: Underlying writer receives the data
	ds := &bytes.Buffer{}
	w := NewWriterSize(ds, 11)
	_, err := w.Write([]byte("hello world"))
	require.NoError(t, err)
	err = w.Flush()
	require.NoError(t, err)
	assert.Equal(t, "hello world", ds.String())

	// Valid: No Data if not Flush-ed
	ds = &bytes.Buffer{}
	w = NewWriter(ds)
	_, err = w.Write([]byte("hello world"))
	require.NoError(t, err)
	assert.Equal(t, "", ds.String())

	// Valid: Large data does not need Flush-ing
	ds = &bytes.Buffer{}
	w = NewWriterSize(ds, 10)
	_, err = w.Write([]byte("hello world"))
	require.NoError(t, err)
	assert.Equal(t, "hello world", ds.String())

	// Valid: Flushing empty buffer
	err = w.Flush()
	require.NoError(t, err)
}
