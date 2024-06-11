package main

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"fmt"
	"os"
	"path/filepath"
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
	isGit := isGyatFolderExist()
	if isGit {
		fmt.Fprintf(os.Stderr, "Fatal: .git folder not found in working directory")
		os.Exit(1)
	}

	// use map for more arguments options
	hashs := []string{}

	for i, arg := range os.Args {
		if arg == "-p" {
			hashs = append(hashs, os.Args[i+1])
		}
	}

	if len(hashs) == 0 {
		fmt.Fprintf(os.Stderr, "Error: insufficient arguments")
		os.Exit(1)
	}

	for _, hash := range hashs {
		content := catSingleFile(hash)
		fmt.Println(content)
	}
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

func lsTree() {
	options := make(map[string]string)
	var idx int
	for i := 2; i < len(os.Args); i++ {
		arg := os.Args[i]
		if string(arg[0]) == "-" {
			options[arg] = ""
			continue
		}
		idx = i
		break
	}

	treeHash := os.Args[idx]

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
	objects := lsTreeEntrys(treeFilePath)

	_, ok := options["-d"]
	if ok {
		n := 0
		for _, obj := range objects {
			if obj.objType == "tree" {
				objects[n] = obj
				n++
				continue
			}
		}

		objects = objects[:n]
	}

	_, ok = options["--name-only"]
	if ok {
		for _, obj := range objects {
			fmt.Println(obj.name)
		}
		os.Exit(0)
	}

	_, ok = options["--name-status"]
	if ok {
		for _, obj := range objects {
			fmt.Println(obj.name)
		}
		os.Exit(0)
	}

	_, ok = options["--object-only"]
	if ok {
		for _, obj := range objects {
			fmt.Println(obj.objType)
		}
		os.Exit(0)
	}

	for _, obj := range objects {
		fmt.Printf("%d %s %s %s\n", obj.mode, obj.objType, obj.shaHash, obj.name)
	}
	os.Exit(0)
}
