package kghttp

import (
	"bytes"
	"testing"

	"github.com/Kaung-HtetKyaw/kgx/kgbuf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFixedWriterWrite(t *testing.T) {
	t.Run("write specified content length", func(t *testing.T) {
		ds := &bytes.Buffer{}
		bw := kgbuf.NewWriter(ds)
		fw := NewFixedWriter(bw, func() Headers {
			return Headers{
				"content-length": "11",
			}
		})
		_, err := fw.Write([]byte("hello world"))
		require.NoError(t, err)
		err = fw.Flush()
		require.NoError(t, err)
		assert.Contains(t, ds.String(), "hello world")
	})

	t.Run("write zero length content", func(t *testing.T) {
		ds := &bytes.Buffer{}
		bw := kgbuf.NewWriter(ds)
		fw := NewFixedWriter(bw, func() Headers {
			return Headers{
				"content-length": "0",
			}
		})
		n, err := fw.Write([]byte(""))
		err = fw.Flush()
		require.NoError(t, err)
		require.NoError(t, err)
		assert.Equal(t, 0, n)
	})

	t.Run("body exceeds declared content length", func(t *testing.T) {
		ds := &bytes.Buffer{}
		bw := kgbuf.NewWriter(ds)
		fw := NewFixedWriter(bw, func() Headers {
			return Headers{
				"content-length": "11",
			}
		})
		n, err := fw.Write([]byte("hello world!"))
		require.Error(t, err)
		assert.Equal(t, 0, n)
	})

	t.Run("partial body data", func(t *testing.T) {
		ds := &bytes.Buffer{}
		bw := kgbuf.NewWriter(ds)
		fw := NewFixedWriter(bw, func() Headers {
			return Headers{
				"content-length": "11",
			}
		})
		_, err := fw.Write([]byte("hello"))
		require.NoError(t, err)
		err = fw.Flush()
		require.NoError(t, err)
		err = fw.Close()
		require.Error(t, err)
	})

	t.Run("no content length header", func(t *testing.T) {
		ds := &bytes.Buffer{}
		bw := kgbuf.NewWriter(ds)
		fw := NewFixedWriter(bw, func() Headers {
			return Headers{}
		})
		_, err := fw.Write([]byte("hello"))
		require.Error(t, err)
	})

	t.Run("write after close", func(t *testing.T) {
		ds := &bytes.Buffer{}
		bw := kgbuf.NewWriter(ds)
		fw := NewFixedWriter(bw, func() Headers {
			return Headers{"content-length": "11"}
		})
		_, err := fw.Write([]byte("hello world"))
		require.NoError(t, err)
		err = fw.Flush()
		require.NoError(t, err)
		err = fw.Close()
		require.NoError(t, err)
		_, err = fw.Write([]byte("hi"))
		require.Error(t, err)
	})

	t.Run("flush after close", func(t *testing.T) {
		ds := &bytes.Buffer{}
		bw := kgbuf.NewWriter(ds)
		fw := NewFixedWriter(bw, func() Headers {
			return Headers{"content-length": "11"}
		})
		_, err := fw.Write([]byte("hello world"))
		require.NoError(t, err)
		err = fw.Flush()
		require.NoError(t, err)
		err = fw.Close()
		require.NoError(t, err)
		err = fw.Flush()
		require.Error(t, err)
	})

	t.Run("multiple close", func(t *testing.T) {
		ds := &bytes.Buffer{}
		bw := kgbuf.NewWriter(ds)
		fw := NewFixedWriter(bw, func() Headers {
			return Headers{"content-length": "11"}
		})
		_, err := fw.Write([]byte("hello world"))
		require.NoError(t, err)
		err = fw.Flush()
		require.NoError(t, err)
		err = fw.Close()
		require.NoError(t, err)
		err = fw.Close()
		require.Error(t, err)
	})
}
