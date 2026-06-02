package main

import (
	"errors"
	"fmt"
	"go-http-server/internal/request"
	"go-http-server/internal/response"
	"go-http-server/internal/server"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
)

const port = 42069
const httpBinURL = "https://httpbin.org/"
const httpBinPath = "/httpbin/"

func main() {
	server, err := server.Serve(port, handler)
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
	defer server.Close()
	log.Println("Server started on port", port)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Println("Server gracefully stopped")
}

func handler(w *response.Writer, req *request.Request) {

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

	writeResponse(w)
}

func httpBinProxyHandler(w *response.Writer, req *request.Request) {
	path := strings.TrimPrefix(req.RequestLine.RequestTarget, httpBinPath)
	url := httpBinURL + path
	resp, err := http.Get(url)
	if err != nil {
		writeInternalServerError(w)
		return
	}

	w.Headers().Set("connection", resp.Header.Get("connection"))
	w.Headers().Set("transfer-encoding", "chunked")
	w.Headers().Set("trailer", resp.Header.Get("trailer"))
	w.Headers().Set("content-type", resp.Header.Get("content-type"))
	w.WriteHeaders(response.StatusCode(resp.StatusCode))

	buf := make([]byte, 1024)
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
	}

	if _, err := w.WriteChunkedBodyDone(); err != nil {
		fmt.Println("error writing chunked body done, ", err)
	}

}

func writeResponse(w *response.Writer) {
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
	w.WriteHeaders(response.StatusOK)
	w.WriteBody([]byte(body))
}

func writeBadRequestError(w *response.Writer) {
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
	w.WriteHeaders(response.StatusBadRequest)
	w.WriteBody([]byte(body))
}

func writeInternalServerError(w *response.Writer) {
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
	w.WriteHeaders(response.StatusInternalServerError)
	w.WriteBody([]byte(body))
}
