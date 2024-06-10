package main

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func gyatInit() {
	// create .gyat directories
	for _, dir := range []string{".gyat", ".gyat/objects", ".gyat/refs"} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating directory: %s\n", err)
			return
		}
	}

	// create gyat HEAD file
	headFileContents := []byte("ref: refs/heads/main\n")
	if err := os.WriteFile(".gyat/HEAD", headFileContents, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing file: %s\n", err)
		return
	}

	fmt.Println("Initialized gyat directory")

}

func getBlobFile(hash string) (string, error) {
	blobDirectory := fmt.Sprintf(".gyat/objects/%v", hash[0:2])
	partialBlobName := hash[2:]
	partialBlobNameLength := len(hash) - 2
	hasFoundPattern := false
	var blobFileName string

	entries, err := os.ReadDir(blobDirectory)
	if err != nil {
		return "", err
	}

	for _, e := range entries {
		partialEntryName := e.Name()[0:partialBlobNameLength]

		if hasFoundPattern && partialBlobName == partialEntryName {
			return "", errors.New("multiple blob file has the same pattern provided")
		}

		if !hasFoundPattern && partialBlobName == partialEntryName {
			hasFoundPattern = true
			blobFileName = e.Name()
		}
	}

	return filepath.Join(blobDirectory, blobFileName), nil
}

func catFile() {
	// retrieve blob file name
	hash := os.Args[3]
	hashLength := len(hash)
	var blobPath string
	var err error

	if hashLength < 6 {
		fmt.Fprintf(os.Stderr, "Usage: provide a hash length of atleast 6 characters")
		return
	}

	if hashLength < 40 {
		blobPath, err = getBlobFile(hash)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error finding blob file: %s\n", err)
			return
		}

	} else {
		blobPath = fmt.Sprintf(".gyat/objects/%v/%v", hash[0:2], hash[2:])
	}

	file, err := os.Open(blobPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening file: %s\n", err)
		return
	}
	// instantiate file reader
	r, err := zlib.NewReader(io.Reader(file))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error decompressing blob file: %s\n", err)
		return
	}

	// read all bytes from reader
	blobBytes, err := io.ReadAll(r)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %s\n", err)
		return
	}

	// seperate the header and the content
	blob := strings.Split(string(blobBytes), "\x00")

	// print the content and close the reader
	fmt.Print(blob[1])
	err = r.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error closing file: %s\n", err)
		return
	}
}

func hashObject() {
	// retrieve file path
	filePath := os.Args[3]

	// retrieve content from file
	content, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading from file: %s\n", err)
	}

	// convert content into blob format
	blob := fmt.Sprintf("blob %d\x00%s", len(content), content)
	shaHash := sha1.Sum([]byte(blob))
	shaHashHex := fmt.Sprintf("%x", shaHash)
	blobName := shaHashHex[2:]
	blobDir := filepath.Join(".gyat", "objects", shaHashHex[:2])

	// write compress blob into buffer
	var buffer bytes.Buffer
	w := zlib.NewWriter(&buffer)
	_, err = w.Write([]byte(blob))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error compressing blob: %s\n", err)
	}
	err = w.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error closing buffer writer: %s\n", err)
	}

	// create blob directory
	err = os.Mkdir(blobDir, 0755)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating directory: %s\n", err)
	}

	// write compressed blob to blob file
	err = os.WriteFile(filepath.Join(blobDir, blobName), buffer.Bytes(), 0755)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing to file: %s\n", err)
	}

	fmt.Println(shaHashHex)
}
