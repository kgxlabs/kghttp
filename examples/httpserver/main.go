package main

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/Kaung-HtetKyaw/kgx/examples/assets"
	"github.com/Kaung-HtetKyaw/kgx/kghttp"
)

const port = 8000
const httpBinURL = "https://httpbin.org/"
const httpBinPath = "/httpbin/"

func main() {
	server := &kghttp.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: handler,
	}
	err := server.ListenAndServe()
	if err != nil {
		log.Printf("error starting server: %v", err)
		return
	}
	defer server.Close()
	log.Printf("server started on port %d", port)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Print("server gracefully stopped")
}

func handler(w *kghttp.ResponseWriter, req *kghttp.Request) {
	if strings.HasPrefix(req.URL.Path, httpBinPath) {
		httpBinProxyHandler(w, req)
		return
	}

	if req.URL.Path == "/yourproblem" {
		writeBadRequestError(w)
		return
	}

	if req.URL.Path == "/myproblem" {
		writeInternalServerError(w)
		return
	}

	if req.URL.Path == "/video" {
		writeVideoResponse(w)
		return
	}

	writeResponse(w)
}

func httpBinProxyHandler(w *kghttp.ResponseWriter, req *kghttp.Request) {
	path := strings.TrimPrefix(req.URL.Path, httpBinPath)
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
			log.Printf("error reading response body: %v", err)
			break
		}

		if _, err := w.WriteChunkedBody(buf[:n]); err != nil {
			log.Printf("error writing chunked body: %v", err)
			break
		}

		h.Write(buf[:n])
		totalN += n
	}

	w.Trailers().Set("X-Content-SHA256", fmt.Sprintf("%x", h.Sum(nil)))
	w.Trailers().Set("X-Content-Length", strconv.Itoa(totalN))

	if _, err := w.WriteChunkedBodyDone(); err != nil {
		log.Printf("error writing chunked body done: %v", err)
		return
	}
}

func writeVideoResponse(w *kghttp.ResponseWriter) {
	f, err := assets.FS.Open("pepe.mp4")
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
				log.Printf("error writing chunked body: %v", err)
				break

			}
			h.Write(buf[:n])
			totalN += n
		}

		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			log.Printf("error reading response body: %v", err)
			break
		}

	}

	w.Trailers().Set("X-Content-SHA256", fmt.Sprintf("%x", h.Sum(nil)))
	w.Trailers().Set("X-Content-Length", strconv.Itoa(totalN))

	if _, err := w.WriteChunkedBodyDone(); err != nil {
		log.Printf("error writing chunked body done: %v", err)
		return
	}
}

func writeResponse(w *kghttp.ResponseWriter) {
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

func writeBadRequestError(w *kghttp.ResponseWriter) {
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

func writeInternalServerError(w *kghttp.ResponseWriter) {
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
