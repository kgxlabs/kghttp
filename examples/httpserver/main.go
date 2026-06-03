package main

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	kghttp "kg-http"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

const port = 42069
const httpBinURL = "https://httpbin.org/"
const httpBinPath = "/httpbin/"

func main() {
	server, err := kghttp.Serve(port, handler)
	if err != nil {
		fmt.Printf("Error starting server: %v\n", err)
		return
	}
	defer server.Close()
	fmt.Println("Server started on port", port)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	fmt.Println("Server gracefully stopped")
}

func handler(w *kghttp.Writer, req *kghttp.Request) {
	if strings.HasPrefix(req.RequestLine.RequestTarget, httpBinPath) {
		httpBinProxyHandler(w, req)
		return
	}

	if req.RequestLine.RequestTarget == "/yourproblem" {
		writeBadRequestError(w)
		return
	}

	if req.RequestLine.RequestTarget == "/myproblem" {
		writeInternalServerError(w)
		return
	}

	if req.RequestLine.RequestTarget == "/video" {
		writeVideoResponse(w)
		return
	}

	writeResponse(w)
}

func httpBinProxyHandler(w *kghttp.Writer, req *kghttp.Request) {
	path := strings.TrimPrefix(req.RequestLine.RequestTarget, httpBinPath)
	url := httpBinURL + path
	resp, err := http.Get(url)
	if err != nil {
		writeInternalServerError(w)
		return
	}

	w.Headers().Set("transfer-encoding", "chunked")
	w.Headers().Set("content-type", resp.Header.Get("content-type"))
	w.Headers().Set("trailer", "X-Content-SHA256, X-Content-Length")
	w.WriteHeaders(kghttp.StatusCode(resp.StatusCode))

	buf := make([]byte, 1024)
	h := sha256.New()
	totalN := 0

	for {
		n, err := resp.Body.Read(buf)

		if n == 0 {
			break
		}

		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			fmt.Printf("error reading response body: %v", err)
			break
		}

		if _, err := w.WriteChunkedBody(buf[:n]); err != nil {
			fmt.Println("error writing chunked body ", err)
			break
		}

		h.Write(buf[:n])
		totalN += n
	}

	w.Trailers().Set("X-Content-SHA256", fmt.Sprintf("%x", h.Sum(nil)))
	w.Trailers().Set("X-Content-Length", strconv.Itoa(totalN))

	if _, err := w.WriteChunkedBodyDone(); err != nil {
		fmt.Println("error writing chunked body done, ", err)
		return
	}
}

func writeVideoResponse(w *kghttp.Writer) {
	dir, err := os.Getwd()
	if err != nil {
		writeInternalServerError(w)
		return
	}

	path := filepath.Join(dir, "assets", "vim.mp4")
	f, err := os.Open(path)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	defer f.Close()

	w.Headers().Set("transfer-encoding", "chunked")
	w.Headers().Set("content-type", "video/mp4")
	w.Headers().Set("trailer", "X-Content-SHA256, X-Content-Length")
	w.WriteHeaders(kghttp.StatusCode(kghttp.StatusOK))

	buf := make([]byte, 1024)
	h := sha256.New()
	totalN := 0

	for {
		n, err := f.Read(buf)

		if n > 0 {
			if _, err := w.WriteChunkedBody(buf[:n]); err != nil {
				fmt.Println("error writing chunked body ", err)
				break

			}
			h.Write(buf[:n])
			totalN += n
		}

		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			fmt.Printf("error reading response body: %v", err)
			break
		}

	}

	w.Trailers().Set("X-Content-SHA256", fmt.Sprintf("%x", h.Sum(nil)))
	w.Trailers().Set("X-Content-Length", strconv.Itoa(totalN))

	if _, err := w.WriteChunkedBodyDone(); err != nil {
		fmt.Println("error writing chunked body done, ", err)
		return
	}
}

func writeResponse(w *kghttp.Writer) {
	body := `
	<html>
  <head>
    <title>200 OK</title>
  </head>
  <body>
    <h1>Success!</h1>
    <p>Your request was an absolute banger.</p>
  </body>
</html>
	`

	w.Headers().Set("content-length", strconv.Itoa(len(body)))
	w.Headers().Set("content-type", "text/html")
	w.Headers().Set("connection", "close")
	w.WriteHeaders(kghttp.StatusOK)
	w.WriteBody([]byte(body))
}

func writeBadRequestError(w *kghttp.Writer) {
	body := `
		<html>
  <head>
    <title>400 Bad Request</title>
  </head>
  <body>
    <h1>Bad Request</h1>
    <p>Your request honestly kinda sucked.</p>
  </body>
</html>
	`

	w.Headers().Set("content-length", strconv.Itoa(len(body)))
	w.Headers().Set("content-type", "text/html")
	w.Headers().Set("connection", "close")
	w.WriteHeaders(kghttp.StatusBadRequest)
	w.WriteBody([]byte(body))
}

func writeInternalServerError(w *kghttp.Writer) {
	body := `
		<html>
  <head>
    <title>500 Internal Server Error</title>
  </head>
  <body>
    <h1>Internal Server Error</h1>
    <p>Okay, you know what? This one is on me.</p>
  </body>
</html>
	`

	w.Headers().Set("content-length", strconv.Itoa(len(body)))
	w.Headers().Set("content-type", "text/html")
	w.Headers().Set("connection", "close")
	w.WriteHeaders(kghttp.StatusInternalServerError)
	w.WriteBody([]byte(body))
}
