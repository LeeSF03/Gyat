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
	"strconv"
	"strings"
	"syscall"
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
func getBlobContent(hash string) string {
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

func getHashFromBlob(blob string) string {
	shaHash := sha1.Sum([]byte(blob))
	shaHashHex := fmt.Sprintf("%x", shaHash)
	return shaHashHex
}

func writeBlobToFile(shaHashHex string, blob string) {
	blobName := shaHashHex[2:]
	blobDir := filepath.Join(".gyat", "objects", shaHashHex[:2])

	// write compress blob into buffer
	var buffer bytes.Buffer
	w := zlib.NewWriter(&buffer)
	_, err := w.Write([]byte(blob))
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

func writeIndexContent(filePath string, content bytes.Buffer) {
	dirEntrys, err := os.ReadDir(filePath)
	if err != nil {
		fmt.Printf("Error reading directory: %s\n", err)
	}

	for _, dirEntry := range dirEntrys {
		entryName := filePath + "/" + dirEntry.Name()

		if entryName == filePath+"/"+".gyat" || entryName == filePath+"/"+".git" {
			continue
		}

		if dirEntry.IsDir() {
			writeIndexContent(entryName, content)
			continue
		}

		stat, err := os.Stat(entryName)
		if err != nil {
			fmt.Printf("Error in finding file %s\n", err)
			os.Exit(1)
		}

		d := stat.Sys().(*syscall.Win32FileAttributeData)

		// 32 bit create time, seconds v
		// 32 bit create time, nano secs v
		cns := d.CreationTime.Nanoseconds()
		ctime_s := cns / 1000_000_000
		ctime_n := cns

		// 32 bit modfied time, seconds v
		// 32 bit modified time, nano secs v
		mns := d.LastWriteTime.Nanoseconds()
		mtime_s := mns / 1000_000_000
		mtime_n := mns

		fmt.Println(entryName, ctime_s, ctime_n, mtime_s, mtime_n)

		// 4 bit object type
		// 3 bit unused
		// 9 bit file permission based on type of blob
		// 4 byte uid
		// 4 byte gid
		// 4 byte file size
		// filepath
		// 20 byte hash
		content, err := os.ReadFile(entryName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading from file: %s\n", err)
			os.Exit(1)
		}
		blob := fmt.Sprintf("blob %d\x00%s", len(content), content)
		hash := getHashFromBlob(blob)
		fmt.Println(hash)
		// c := make([]int8, 20)
		// b := []byte(hash)
		// for i := 0; i < len(hash); i += 2 {
		// 	var char int8
		// 	if i % 2 == 0 && string(hash[i]) <= "9" && string(hash[i]) >= "1" {

		// 	}
		// 	char := (int8(hash[i]) << 4) & int8(hash[i+1])
		// 	c = append(c, char)
		// 	fmt.Printf("%b %b %s %s | %b\n", hash[i], hash[i+1], string(hash[i]), string(hash[i+1]), char)
		// }
		// fmt.Println(c)
		// fmt.Println(b)
	}
}
