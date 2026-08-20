package main

import (
	"fmt"
	"os"
)

// Placeholder — replaced by the real CLI in Task 6.
func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: define <word>")
		os.Exit(2)
	}
	text, err := systemDictionary().Lookup(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(text)
}
