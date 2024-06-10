package main

import (
	"fmt"
	"os"
)

func main() {

	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: gyat <commands> [<args>...]\n")
		os.Exit(1)
	}

	switch command := os.Args[1]; command {
	case "init":
		gyatInit()

	case "cat-file":
		catFile()

	case "hash-object":
		hashObject()

	default:
		fmt.Println("Initialized gyat directory")
		os.Exit(1)
	}
}
