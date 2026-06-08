package testutil

import "io"

type ChunkedReader struct {
	Data            string
	NumBytesPerRead int
	pos             int
}

// Read reads up to len(p) or numBytesPerRead bytes from the string per call
// its useful for simulating reading a variable number of bytes per chunk from a network connection
func (cr *ChunkedReader) Read(p []byte) (n int, err error) {
	if cr.pos >= len(cr.Data) {
		return 0, io.EOF
	}

	endIndex := min(cr.pos+cr.NumBytesPerRead, len(cr.Data))
	n = copy(p, cr.Data[cr.pos:endIndex])
	cr.pos += n

	return n, nil
}
