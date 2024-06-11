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

func catFile() []string {
	hashs := []string{}

	isGit := isGyatFolderExist()
	if isGit {
		fmt.Fprintf(os.Stderr, "Fatal: .git folder not found in working directory")
		os.Exit(1)
	}

	for i, arg := range os.Args {
		if arg == "-p" {
			hashs = append(hashs, os.Args[i+1])
		}
	}

	if len(hashs) == 0 {
		fmt.Fprintf(os.Stderr, "Error: insufficient arguments")
		os.Exit(1)
	}

	var contents []string
	for _, hash := range hashs {
		content := catSingleFile(hash)
		fmt.Println(content)
		contents = append(contents, content)
	}

	return contents
}

func hashObject() string {
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
	return shaHashHex
}

func lsTree() []string {
	// arg := os.Args[2]
	// if arg != "--name-only" {
	// 	fmt.Fprintf(os.Stderr, "Fatal: missng --name-only argument")
	// 	return
	// }

	treeHash := os.Args[2]

	isGit := isGyatFolderExist()
	if isGit {
		fmt.Fprintf(os.Stderr, "Fatal: .git folder not found in working directory")
		os.Exit(1)
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
		fmt.Println(string(v), i)
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
				strings.TrimSuffix(
					fmt.Sprintf("%d%s %x\n", initial, treeFileByteContent[start:i], treeFileByteContent[i+1:i+21]), "\n",
				),
			)
			i += 22
			start = i
			continue
		}
		i++
	}

	returnLines := []string{}

	for _, line := range treefileLines {
		sepLine := strings.Split(line, " ")
		returnLines = append(returnLines,
			fmt.Sprintf("%s %s %s %s\n", sepLine[0], getEntryType(sepLine[0]), sepLine[2], sepLine[1]),
		)
	}

	return returnLines
}
