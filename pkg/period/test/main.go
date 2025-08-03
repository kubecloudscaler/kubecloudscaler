package main

import (
	"fmt"
	"regexp"
)

func main() {
	ref := "test1/test2@v1.0.0"
	refReg := regexp.MustCompile(`^([\w\-]+)\/([\w\-]+)@([\w.\-_]+)$`)

	matches := refReg.FindStringSubmatch(ref)
	fmt.Printf("matches: %v\n", matches)

	ref = "test1test2/v1.0.0"
	refReg = regexp.MustCompile(`^([\w\-]+)\/([\w\-]+)@([\w.\-_]+)$`)

	matches = refReg.FindStringSubmatch(ref)
	fmt.Printf("matches: %v\n", matches)
}
