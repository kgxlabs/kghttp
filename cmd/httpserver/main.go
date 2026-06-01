package main

import (
	"go-http-server/internal/request"
	"go-http-server/internal/response"
	"go-http-server/internal/server"
	"log"
	"os"
	"os/signal"
	"syscall"
)

const port = 42069

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

// NOTE: Our user supplied handler func is a bit wonky.
// In the event of error, it returns HandlerError let the HandlerError.Write do the thing.
// But in the event of success, it handles the writing by itself by doing `w.Write`.
// TODO: Refactor that.
func handler(w *response.Writer, req *request.Request) {
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

	w.WriteStatusLine(response.StatusOK)
	w.Headers.Set("content-type", "text/html")
	w.Headers.Set("connection", "close")
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

	w.WriteStatusLine(response.StatusBadRequest)
	w.Headers.Set("content-type", "text/html")
	w.Headers.Set("connection", "close")
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

	w.WriteStatusLine(response.StatusInternalServerError)
	w.Headers.Set("content-type", "text/html")
	w.Headers.Set("connection", "close")
	w.WriteBody([]byte(body))
}
