package internal

import (
	"errors"
	"io"
)

var NoBody = nobody{}

type nobody struct {
	ended bool
}

var (
	ErrNoBodyClosed = errors.New("kghttp: body already closed")
)

func (nb *nobody) Read(p []byte) (int, error) {
	return 0, io.EOF
}

func (nb *nobody) Close() error {
	if nb.ended {
		return ErrNoBodyClosed
	}

	if err := nb.Flush(); err != nil {
		return err
	}

	nb.ended = true
	return nil
}

func (nb *nobody) Write(p []byte) (int, error) {
	return 0, nil
}

func (nb *nobody) Flush() error {
	if nb.ended {
		return ErrNoBodyClosed
	}
	return nil
}
