package main

import (
	"fmt"
	"os"
)

func main() {

	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: gyat <commands> [<args>...]\n")
		os.Exit(1)
	}

	switch command := os.Args[1]; command {
	case "init":
		gyatInit()

	case "cat-file":
		catFile(os.Args[2:]...)

	case "hash-object":
		hashObject(os.Args[2:]...)

	case "ls-tree":
		lsTree(os.Args[2:]...)

	case "add":
		stageFiles(os.Args[2:]...)

	default:
		fmt.Println("Initialized gyat directory")
		os.Exit(1)
	}
}
