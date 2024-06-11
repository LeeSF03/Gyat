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

func gyatInit() {
	// create .gyat directories
	for _, dir := range []string{".gyat", ".gyat/objects", ".gyat/refs"} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating directory: %s\n", err)
			os.Exit(1)
		}
	}

	// create gyat HEAD file
	headFileContents := []byte("ref: refs/heads/main\n")
	if err := os.WriteFile(".gyat/HEAD", headFileContents, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing file: %s\n", err)
		os.Exit(1)
	}

	fmt.Println("Initialized gyat directory")

}

func catFile() {
	arg := os.Args[2]
	if arg != "-p" {
		fmt.Fprintf(os.Stderr, "Fatal: missng -p argument")
		os.Exit(1)
	}

	isGit := isGyatFolderExist()
	if isGit {
		fmt.Fprintf(os.Stderr, "Fatal: .git folder not found in working directory")
		os.Exit(1)
	}

	// retrieve blob file name
	hash := os.Args[3]
	hashLength := len(hash)
	var blobPath string
	var err error

	if hashLength < 6 {
		fmt.Fprintf(os.Stderr, "Usage: provide a hash length of atleast 6 characters")
		os.Exit(1)
	}

	if hashLength < 40 {
		blobPath, err = getObjectFile(hash)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error finding blob file: %s\n", err)
			os.Exit(1)
		}

	} else {
		blobPath = fmt.Sprintf(".gyat/objects/%v/%v", hash[0:2], hash[2:])
	}

	file, err := os.Open(blobPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening file: %s\n", err)
		os.Exit(1)
	}
	defer file.Close()

	// instantiate file reader
	r, err := zlib.NewReader(io.Reader(file))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error decompressing blob file: %s\n", err)
		os.Exit(1)
	}
	defer r.Close()

	// read all bytes from reader
	blobBytes, err := io.ReadAll(r)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %s\n", err)
		os.Exit(1)
	}

	// seperate the header and the content
	blob := strings.Split(string(blobBytes), "\x00")

	// print the content and close the reader
	fmt.Print(blob[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error closing file: %s\n", err)
		os.Exit(1)
	}
}

func hashObject() {
	arg := os.Args[2]
	if arg != "-w" {
		fmt.Fprintf(os.Stderr, "Fatal: missng -w argument")
		os.Exit(1)
	}

	isGit := isGyatFolderExist()
	if isGit {
		fmt.Fprintf(os.Stderr, "Fatal: .git folder not found in working directory")
		os.Exit(1)
	}

	// retrieve file path
	filePath := os.Args[3]

	// retrieve content from file
	content, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading from file: %s\n", err)
		os.Exit(1)
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
		os.Exit(1)
	}
	err = w.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error closing buffer writer: %s\n", err)
		os.Exit(1)
	}

	// create blob directory
	err = os.Mkdir(blobDir, 0755)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating directory: %s\n", err)
		os.Exit(1)
	}

	// write compressed blob to blob file
	err = os.WriteFile(filepath.Join(blobDir, blobName), buffer.Bytes(), 0755)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing to file: %s\n", err)
		os.Exit(1)
	}

	fmt.Println(shaHashHex)
}

func lsTree() {
	// arg := os.Args[2]
	// if arg != "--name-only" {
	// 	fmt.Fprintf(os.Stderr, "Fatal: missng --name-only argument")
	// 	return
	// }

	treeHash := os.Args[2]

	isGit := isGyatFolderExist()
	if isGit {
		fmt.Fprintf(os.Stderr, "Fatal: .git folder not found in working directory")
		return
	}

	treeFilePath, err := getObjectFile(treeHash)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting file: %s\n", err)
		os.Exit(1)
	}

	treeFile, err := os.Open(treeFilePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening file: %s\n", err)
		os.Exit(1)
	}
	defer treeFile.Close()

	r, err := zlib.NewReader(treeFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error decompressing file: %s\n", err)
		os.Exit(1)
	}

	treeFileBytes, err := io.ReadAll(r)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %s\n", err)
		os.Exit(1)
	}

	if string(treeFileBytes[:4]) != "tree" {
		fmt.Fprintf(os.Stderr, "Fatal: not a tree file")
		os.Exit(1)
	}

	var hIdx int
	for i, v := range treeFileBytes {
		if string(v) == "\x00" {
			hIdx = i + 1
			break
		}
	}

	treeFileByteContent := treeFileBytes[hIdx:]
	treefileLines := []string{}

	start := 0
	for i := 0; i < len(treeFileByteContent); {
		if string(treeFileByteContent[i]) == "\x00" {
			initial := 0
			char := string(treeFileByteContent[start])
			if char == "0" || char == "2" {
				initial = 1
			}

			treefileLines = append(treefileLines,
				fmt.Sprintf("%d%s %x\n", initial, treeFileByteContent[start:i], treeFileByteContent[i+1:i+21]))
			i += 22
			start = i
			continue
		}
		i++
	}

	var sepLine []string
	for _, line := range treefileLines {
		sepLine = strings.Split(line, " ")
		fmt.Printf("%s %s %s %s\n", sepLine[0], getEntryType(sepLine[0]), strings.TrimSuffix(sepLine[2], "\n"), sepLine[1])
	}
}
