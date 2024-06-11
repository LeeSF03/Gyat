package main

import (
	"compress/zlib"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Object struct {
	mode    int
	objType string
	shaHash string
	name    string
}

// * retrieve the full file path to the obejct file using the partial hash (atleast 6 characters) given
func getObjectFile(partialHash string) (string, error) {
	objectDirectory := fmt.Sprintf(".gyat/objects/%v", partialHash[:2])
	partialObjectName := partialHash[2:]
	partialObjectNameLength := len(partialHash) - 2
	hasFoundPattern := false
	var blobFileName string

	entries, err := os.ReadDir(objectDirectory)
	if err != nil {
		return "", err
	}

	// * check if only one file match the hash pattern
	for _, e := range entries {
		partialEntryName := e.Name()[0:partialObjectNameLength]

		if hasFoundPattern && partialObjectName == partialEntryName {
			return "", errors.New("multiple blob file has the same pattern provided")
		}

		if !hasFoundPattern && partialObjectName == partialEntryName {
			hasFoundPattern = true
			blobFileName = e.Name()
		}
	}

	return filepath.Join(objectDirectory, blobFileName), nil
}

// * check if the .gyat folder exist
func isGyatFolderExist() bool {
	entryInfo, err := os.Stat("./gyat")
	if err != nil {
		return false
	}
	if entryInfo.IsDir() {
		return false
	}

	return true
}

func getEntryType(n string) string {
	switch n {
	case "100644":
		return "blob"
	case "100755":
		return "blob"
	case "120000":
		return "blob"
	case "040000":
		return "tree"
	default:
		return "blob"
	}
}

// retrieve blob file name
func catSingleFile(hash string) string {
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

	return blob[1]
}

func lsTreeEntrys(treePath string) []Object {
	treeFile, err := os.Open(treePath)
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

	objs := []Object{}

	for _, line := range treefileLines {
		sepLine := strings.Split(line, " ")
		mode, _ := strconv.Atoi(sepLine[0])
		objs = append(objs,
			Object{mode, getEntryType(sepLine[0]), sepLine[2], sepLine[1]},
		)
	}

	return objs
}
