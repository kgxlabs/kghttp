package main

import (
	"fmt"
	"go-http-server/internal/request"
	"log"
	"net"
)

func main() {
	listener, err := net.Listen("tcp", ":42069")
	if err != nil {
		log.Fatal("Error listening: ", err)
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal("Connection failed, ", err)
		}

		fmt.Println("Connection is accepted")

		requestLine, err := request.RequestFromReader(conn)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("Request line: ")
		fmt.Printf("- Method: %s\n", requestLine.RequestLine.Method)
		fmt.Printf("- Target: %s\n", requestLine.RequestLine.RequestTarget)
		fmt.Printf("- Version: %s\n", requestLine.RequestLine.HttpVersion)
		fmt.Println("Headers: ")
		for name, value := range requestLine.Headers {
			fmt.Printf("- %s: %s\n", name, value)
		}
	}
}
