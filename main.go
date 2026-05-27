package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
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
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal("Dir get failed: ", err)
	}
	path := filepath.Join(dir, "messages.txt")
	reader, err := os.Open(path)
	defer reader.Close()
	if err != nil {
		log.Fatal("Read failed: ", err)
	}

	for line := range getLinesChannel(reader) {
		fmt.Printf("read: %s\n", line)
	}
}
