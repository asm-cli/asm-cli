package main

import (
	"fmt"
	"os"

	"github.com/asm-cli/asm-cli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "asm:", err)
		os.Exit(1)
	}
}
