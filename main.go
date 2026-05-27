package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func read_file(reader *os.File) {
	bytes := make([]byte, 8)

	for {
		n, err := reader.Read(bytes)
		if err != nil {
			break
		}

		str := string(bytes[:n])

		fmt.Printf("read: %s\n", str)
	}
}

func main() {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	path := filepath.Join(dir, "messages.txt")
	reader, err := os.Open(path)
	defer reader.Close()
	if err != nil {
		panic(err)
	}

	read_file(reader)
}
