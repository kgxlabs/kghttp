package main

import (
	"go-http-server/internal/request"
	"go-http-server/internal/server"
	"io"
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
func handler(w io.Writer, req *request.Request) *server.HandlerError {
	if req.RequestLine.RequestTarget == "/yourproblem" {
		return &server.HandlerError{
			StatusCode: 400,
			Message:    "Your problem is not my problem\n",
		}
	}

	if req.RequestLine.RequestTarget == "/myproblem" {
		return &server.HandlerError{
			StatusCode: 500,
			Message:    "Woopsie, my bad\n",
		}
	}

	if _, err := w.Write([]byte("All good, frfr\n")); err != nil {
		return &server.HandlerError{
			StatusCode: 500,
			Message:    "Internal Server Error",
		}
	}

	return nil
}
