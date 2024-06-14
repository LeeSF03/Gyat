package main

import (
	"bytes"
	"fmt"
	"os"
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

func catFile(args ...string) {
	isGit := isGyatFolderExist()
	if isGit {
		fmt.Fprintf(os.Stderr, "Fatal: .gyat folder not found in working directory")
		os.Exit(1)
	}

	hashs := []string{}
	options := make(map[string]string)
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "-p" || arg == "-e" {
			options[arg] = ""
			hashs = append(hashs, args[i+1])
			continue
		}

		if string(arg[0]) == "-" {
			options[arg] = ""
			continue
		}
	}

	if len(hashs) == 0 {
		fmt.Fprintf(os.Stderr, "Error: insufficient arguments")
		os.Exit(1)
	}

	for _, hash := range hashs {
		content := getBlobContent(hash)
		fmt.Print(content)
	}
}

func hashObject(args ...string) {
	arg := args[2]
	if arg != "-w" {
		fmt.Fprintf(os.Stderr, "Fatal: missng -w argument")
		os.Exit(1)
	}

	isGit := isGyatFolderExist()
	if isGit {
		fmt.Fprintf(os.Stderr, "Fatal: .gyat folder not found in working directory")
		os.Exit(1)
	}

	options := make(map[string]string)
	var idx int
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if string(arg[0]) == "-" {
			options[arg] = ""
			continue
		}
		idx = i
		break
	}

	// retrieve file path
	filePath := args[idx]

	// retrieve content from file
	content, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading from file: %s\n", err)
		os.Exit(1)
	}

	// convert content into blob format
	blob := fmt.Sprintf("blob %d\x00%s", len(content), content)
	hash := getHashFromBlob(blob)

	_, ok := options["-w"]
	if ok {
		writeBlobToFile(hash, blob)
		os.Exit(0)
	}

	fmt.Println(hash)
}

func lsTree(args ...string) {
	options := make(map[string]string)
	var idx int
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if string(arg[0]) == "-" {
			options[arg] = ""
			continue
		}
		idx = i
		break
	}

	treeHash := args[idx]

	isGit := isGyatFolderExist()
	if isGit {
		fmt.Fprintf(os.Stderr, "Fatal: .gyat folder not found in working directory")
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

func stageFiles(args ...string) {
	file := args[0]
	var buf bytes.Buffer
	writeIndexContent(file, buf)
}
