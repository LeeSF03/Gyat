package main

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func main() {

	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: gyat <commands> [<args>...]\n")
		os.Exit(1)
	}

	switch command := os.Args[1]; command {
	case "init":
		for _, dir := range []string{".gyat", ".gyat/objects", ".gyat/refs"} {
			if err := os.MkdirAll(dir, 0755); err != nil {
				fmt.Fprintf(os.Stderr, "Error creating directory: %s\n", err)
			}
		}

		headFileContents := []byte("ref: refs/heads/main\n")
		if err := os.WriteFile(".gyat/HEAD", headFileContents, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing file: %s\n", err)
		}

		fmt.Println("Initialized gyat directory")

	case "cat-file":
		sha := os.Args[3]

		path := fmt.Sprintf(".gyat/objects/%v/%v", sha[0:2], sha[2:])
		file, err := os.Open(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening file: %s\n", err)
			return
		}

		r, err := zlib.NewReader(io.Reader(file))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error decompressing blob file: %s\n", err)
			return
		}

		blobBytes, err := io.ReadAll(r)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file: %s\n", err)
			return
		}

		blob := strings.Split(string(blobBytes), "\x00")

		fmt.Print(blob[1])
		r.Close()

	case "hash-object":
		filePath := os.Args[3]

		rawContent, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading from file: %s\n", err)
		}

		blob := fmt.Sprintf("blob %d\x00%s", len(rawContent), rawContent)
		sha := sha1.Sum([]byte(blob))
		hash := fmt.Sprintf("%x", sha)
		blobName := hash[2:]
		blobDir := filepath.Join(".gyat", "objects", hash[:2])

		var buffer bytes.Buffer
		w := zlib.NewWriter(&buffer)
		_, err = w.Write([]byte(blob))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error compressing blob: %s\n", err)
		}
		w.Close()

		err = os.Mkdir(blobDir, 0755)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating directory: %s\n", err)
		}

		err = os.WriteFile(filepath.Join(blobDir, blobName), buffer.Bytes(), 0755)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing to file: %s\n", err)
		}

		fmt.Println(hash)

	default:
		fmt.Println("Initialized gyat directory")
		os.Exit(1)
	}
}
