package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
)

func getLinesChannel(f io.ReadCloser) <-chan string {
	out := make(chan string)

	go func() {
		defer close(out)
		defer f.Close()

		data := make([]byte, 8)
		str := ""

		for {
			n, err := f.Read(data)
			if err != nil {
				break
			}

			data := data[:n]
			if i := bytes.IndexByte(data, '\n'); i != -1 {
				str += string(data[:i])
				out <- str
				data = data[i+1:]
				str = ""
			}

			str += string(data)
		}

		if len(str) != 0 {
			out <- str
		}
	}()

	return out
}

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

		lines := getLinesChannel(conn)
		for line := range lines {
			fmt.Printf("read: %s\n", line)
		}
	}
}
