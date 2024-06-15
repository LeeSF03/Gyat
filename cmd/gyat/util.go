package main

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

type GyatObject struct {
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

func writeBlobToFile(hash string, blob string) {
	blobName := hash[2:]
	blobDir := filepath.Join(".gyat", "objects", hash[:2])

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

func lsTreeEntrys(treePath string) []GyatObject {
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

	objs := []GyatObject{}

	for _, line := range treefileLines {
		sepLine := strings.Split(line, " ")
		mode, _ := strconv.Atoi(sepLine[0])
		objs = append(objs,
			GyatObject{mode, getEntryType(sepLine[0]), sepLine[2], sepLine[1]},
		)
	}

	return objs
}

// each index entry format: 22 byte, \x00, filepath, 20 byte, \n
func writeIndexContent(entryPath string, f *os.File, n *uint32) {
	dirEntrys, err := os.ReadDir(entryPath)
	if err != nil {
		fmt.Printf("Error reading directory: %s\n", err)
	}

	for _, dirEntry := range dirEntrys {
		fullEntryName := entryPath + "/" + dirEntry.Name()

		if fullEntryName == entryPath+"/"+".gyat" || fullEntryName == entryPath+"/"+".git" {
			continue
		}

		if dirEntry.IsDir() {
			writeIndexContent(fullEntryName, f, n)
			continue
		}

		fi, err := os.Stat(fullEntryName)
		if err != nil {
			fmt.Printf("Error in finding file %s\n", err)
			os.Exit(1)
		}

		s := uint32(fi.Size())

		content, err := os.ReadFile(fullEntryName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading from file: %s\n", err)
			os.Exit(1)
		}

		blob := fmt.Sprintf("blob %d\x00%s", s, content)
		hash := getHashFromBlob(blob)
		// fmt.Println(hash)

		// skip file if already exists
		if isObjFileExist(hash) {
			continue
		}

		d := fi.Sys().(*syscall.Win32FileAttributeData)

		buf32 := make([]byte, 4)
		buf16 := make([]byte, 2)

		// 32 bit - 4 byte create time, seconds v
		// 32 bit - 4 byte create time fractions, nano secs v
		ctime_n := d.CreationTime.Nanoseconds()
		binary.BigEndian.PutUint32(buf32, uint32(ctime_n))
		f.Write(buf32)

		ctime_s := uint32(ctime_n % 1000_000_000)
		binary.BigEndian.PutUint32(buf32, uint32(ctime_s))
		f.Write(buf32)

		// fmt.Println(buf32)

		// 32 bit - 4 byte modfied time, seconds v
		// 32 bit - 4 byte modified time fractions, nano secs v
		mtime_n := d.LastWriteTime.Nanoseconds()
		binary.BigEndian.PutUint32(buf32, uint32(mtime_n))
		f.Write(buf32)

		mtime_s := uint32(mtime_n % 1000_000_000)
		binary.BigEndian.PutUint32(buf32, uint32(mtime_s))
		f.Write(buf32)

		// fmt.Println(buf32)

		// fmt.Println(fullEntryName, ctime_s, ctime_n, mtime_s, mtime_n)

		// 4 bit object type
		// 3 bit unused
		// 9 bit file permission based on type of blob
		// total: 2 byte
		var otp uint16
		p := fi.Mode()
		if p&os.ModeSymlink > 0 {
			otp = (uint16(10) << 12) | uint16(p)
		} else {
			otp = (uint16(8) << 12) | uint16(p)
		}
		binary.BigEndian.AppendUint16(buf16, otp)
		f.Write(buf16)
		// fmt.Printf("perm: %b\n", otp)

		// uid and gid is only I linux, I have no idea what windows uses
		// 4 byte uid
		// 4 byte gid

		// 4 byte file size
		// s
		binary.BigEndian.AppendUint32(buf32, s)
		f.Write(buf32)

		// filepath
		// entryName
		f.WriteString(fullEntryName + "\x00")

		// 20 byte hash
		shortenHash := []uint8{}
		for i := 0; i < len(hash); i += 2 {
			c1, err := aToHex(hash[i])
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			}
			c2, err := aToHex(hash[i+1])
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			}
			char := (c1 << 4) | c2
			shortenHash = append(shortenHash, char)
			// fmt.Printf("%b %b %s %s | %b %b %b %d\n", hash[i], hash[i+1], string(hash[i]), string(hash[i+1]), c1, c2, char, char)
		}
		// fmt.Println(shortenHash)
		f.Write(shortenHash)
		f.WriteString("\n")
		(*n)++
	}
}

func aToHex(a byte) (uint8, error) {
	if a >= 48 && a <= 57 {
		return uint8(a - 48), nil
	}

	if a >= 97 && a <= 102 {
		return uint8(a - 87), nil
	}
	return 0, errors.New("can't be converted to hex")
}

func isObjFileExist(hash string) bool {
	objFile := filepath.Join(".gyat", "objects", hash[:2], hash[2:])
	if _, err := os.Stat(objFile); errors.Is(err, os.ErrNotExist) {
		return false
	}
	return true
}
