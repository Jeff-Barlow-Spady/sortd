package main

import (
	"fmt"
	"os"
)

func main() {
	cmd := NewRootCmd()
	if err := cmd.Execute(); err != nil {
		// Print to both stderr and stdout to ensure tests can capture it
		errMsg := fmt.Sprintf("Error: %s", err)
		fmt.Fprintln(os.Stderr, errMsg)
		fmt.Println(errMsg) // Also print to stdout for test capturing
		os.Exit(1)
	}
}
