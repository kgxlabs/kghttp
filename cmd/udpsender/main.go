package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
)

func main() {
	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:42069")
	if err != nil {
		log.Fatal("Failed resolving udp addr", err)
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		log.Fatal("Connection failed", err)
	}
	defer conn.Close()

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Println("> ")
		message, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal("Read failed", err)
			os.Exit(1)
		}

		_, err = conn.Write([]byte(message))
		if err != nil {
			log.Fatal("Write failed", err)
			os.Exit(1)
		}

		fmt.Printf("Message sent: %s", message)
	}
}
