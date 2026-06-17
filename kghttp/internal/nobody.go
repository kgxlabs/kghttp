package internal

import "io"

var NoBody = nobody{}

type nobody struct{}

func (nb *nobody) Read(p []byte) (int, error) {
	return 0, io.EOF
}

func (nb *nobody) Close() error {
	return nil
}
