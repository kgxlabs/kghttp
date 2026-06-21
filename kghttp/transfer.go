package kghttp

import (
	"errors"
	"io"
	"strconv"

	"github.com/Kaung-HtetKyaw/kgx/kgbuf"
	"github.com/Kaung-HtetKyaw/kgx/kghttp/internal"
)

type bodyReader struct {
	// We are going to create new reader based on request headers
	src io.Reader
	r   *kgbuf.Reader
	// ref to a message either *Request or *Response
	msg         any
	sawEOF      bool
	closed      bool
	earlyClosed bool
	chunked     bool
	remaining   int
}

func readCloser(r *kgbuf.Reader, headers Headers, msg any) (io.ReadCloser, error) {
	if encoding, ok := headers.Get("transfer-encoding"); ok {
		if encoding == "chunked" {
			return &bodyReader{
				src:     NewChunkedReader(r),
				r:       r,
				msg:     msg,
				chunked: true,
			}, nil
		}
	}

	contentLenStr, ok := headers.Get("content-length")
	if !ok {
		return &bodyReader{
			src: &internal.NoBody,
			msg: msg,
		}, nil
	}

	contentLen, err := strconv.Atoi(contentLenStr)
	if err != nil {
		return nil, errors.New("invalid content length")
	}

	if contentLen == 0 {
		return &bodyReader{
			src: &internal.NoBody,
			msg: msg,
		}, nil
	}

	return &bodyReader{
		src:       io.LimitReader(r, int64(contentLen)),
		msg:       msg,
		remaining: contentLen,
	}, nil
}

func readTransfer(msg any, r *kgbuf.Reader) error {
	// TODO: Currently, we are treating both Request and Response the same
	// There will be a lot of scenarios in the future and we need to make changes here
	switch rr := msg.(type) {
	case *Request:
		rc, err := readCloser(r, rr.Headers, rr)
		if err != nil {
			return err
		}
		rr.Body = rc
	case *Response:
		rc, err := readCloser(r, rr.Headers, rr)
		if err != nil {
			return err
		}
		rr.Body = rc

	default:
		return errors.New("kghttp: invalid message type")
	}
	return nil
}

func transferFields(r *kgbuf.Reader, msg any) error {
	for {
		line, err := r.ReadBytes([]byte("\r\n"))
		if err != nil {
			return err
		}
		if len(line) == 0 {
			return errors.New("incomplete http request")
		}

		switch rr := msg.(type) {
		case *Request:
			if rr.Trailers == nil {
				rr.Trailers = Headers{}
			}
			_, done, err := rr.Trailers.Parse(line)
			if err != nil {
				return err
			}
			if done {
				return nil
			}
		case *Response:
			if rr.Trailers == nil {
				rr.Trailers = Headers{}
			}
			_, done, err := rr.Trailers.Parse(line)
			if err != nil {
				return err
			}
			if done {
				return nil
			}
		default:
			return errors.New("invalid message type")
		}
	}
}

func (br *bodyReader) Read(p []byte) (int, error) {
	if br.closed {
		if br.earlyClosed || !br.sawEOF {
			return 0, io.ErrUnexpectedEOF
		}
		return 0, io.EOF
	}

	n, err := br.src.Read(p)

	if n > 0 {
		br.remaining -= n
	}

	if err != nil && err == io.EOF {
		br.closed = true
		if br.remaining > 0 {
			br.earlyClosed = true
			return n, io.ErrUnexpectedEOF
		}

		br.sawEOF = true

		// after 0\r\n is processed, parse trailers
		if br.chunked {
			if trailerErr := transferFields(br.r, br.msg); trailerErr != nil {
				return n, trailerErr
			}
		}
	}

	return n, err
}

func (br *bodyReader) Close() error {
	return nil
}

type bodyWriter struct {
	src io.WriteCloser
	w   *kgbuf.Writer
}

func (bw *bodyWriter) Write(p []byte) (int, error) {
	return bw.src.Write(p)
}

func (bw *bodyWriter) Close() error {
	return bw.src.Close()
}

type writeTransferCfg struct {
	writer   *kgbuf.Writer
	headers  func() Headers
	trailers func() Headers
}

func writeTransfer(cfg writeTransferCfg) (io.WriteCloser, error) {
	hs := cfg.headers()
	encoding, ok := hs.Get("transfer-encoding")

	if ok {
		if encoding == "chunked" {
			return &bodyWriter{
				src: NewChunkedWriter(cfg.writer, cfg.trailers),
				w:   cfg.writer,
			}, nil
		}
	}

	contentLenStr, ok := hs.Get("content-length")
	if !ok {
		return &bodyWriter{
			src: &internal.NoBody,
		}, nil
	}

	contentLen, err := strconv.Atoi(contentLenStr)
	if err != nil {
		return nil, errors.New("malformed content len")
	}

	if contentLen == 0 {
		return &bodyWriter{
			src: &internal.NoBody,
		}, nil
	}

	return &bodyWriter{
		src: NewFixedWriter(cfg.writer, cfg.headers),
		w:   cfg.writer,
	}, nil
}
