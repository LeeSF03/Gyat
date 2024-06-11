package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

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
