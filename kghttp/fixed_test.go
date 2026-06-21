package kghttp

import (
	"bytes"
	"testing"

	"github.com/Kaung-HtetKyaw/kgx/kgbuf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFixedWriterWrite(t *testing.T) {
	// Valid: Write specified content length
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

	// Valid: Write zero length content
	ds = &bytes.Buffer{}
	bw = kgbuf.NewWriter(ds)
	fw = NewFixedWriter(bw, func() Headers {
		return Headers{
			"content-length": "0",
		}
	})
	n, err := fw.Write([]byte(""))
	err = fw.Flush()
	require.NoError(t, err)
	require.NoError(t, err)
	assert.Equal(t, 0, n)

	// Invalid: Body exceeds declared content length
	ds = &bytes.Buffer{}
	bw = kgbuf.NewWriter(ds)
	fw = NewFixedWriter(bw, func() Headers {
		return Headers{
			"content-length": "11",
		}
	})
	n, err = fw.Write([]byte("hello world!"))
	require.Error(t, err)
	assert.Equal(t, 0, n)

	// Invalid: Partial body data
	ds = &bytes.Buffer{}
	bw = kgbuf.NewWriter(ds)
	fw = NewFixedWriter(bw, func() Headers {
		return Headers{
			"content-length": "11",
		}
	})
	n, err = fw.Write([]byte("hello"))
	require.NoError(t, err)
	err = fw.Flush()
	require.NoError(t, err)
	err = fw.Close()
	require.Error(t, err)

	// Invalid: no content length header
	ds = &bytes.Buffer{}
	bw = kgbuf.NewWriter(ds)
	fw = NewFixedWriter(bw, func() Headers {
		return Headers{}
	})
	_, err = fw.Write([]byte("hello"))
	require.Error(t, err)

	// Invalid: Write after Close
	ds = &bytes.Buffer{}
	bw = kgbuf.NewWriter(ds)
	fw = NewFixedWriter(bw, func() Headers {
		return Headers{"content-length": "11"}
	})
	_, err = fw.Write([]byte("hello world"))
	require.NoError(t, err)
	err = fw.Flush()
	require.NoError(t, err)
	err = fw.Close()
	require.NoError(t, err)
	_, err = fw.Write([]byte("hi"))
	require.Error(t, err)

	// Invalid: Flush after Close
	ds = &bytes.Buffer{}
	bw = kgbuf.NewWriter(ds)
	fw = NewFixedWriter(bw, func() Headers {
		return Headers{"content-length": "11"}
	})
	_, err = fw.Write([]byte("hello world"))
	require.NoError(t, err)
	err = fw.Flush()
	require.NoError(t, err)
	err = fw.Close()
	require.NoError(t, err)
	err = fw.Flush()
	require.Error(t, err)

	// Invalid: Multiple Close
	ds = &bytes.Buffer{}
	bw = kgbuf.NewWriter(ds)
	fw = NewFixedWriter(bw, func() Headers {
		return Headers{"content-length": "11"}
	})
	_, err = fw.Write([]byte("hello world"))
	require.NoError(t, err)
	err = fw.Flush()
	require.NoError(t, err)
	err = fw.Close()
	require.NoError(t, err)
	err = fw.Close()
	require.Error(t, err)

}
