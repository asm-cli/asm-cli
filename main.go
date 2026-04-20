package main

import (
	"fmt"
	"os"

	"github.com/6xiaowu9/asm/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "asm:", err)
		os.Exit(1)
	}
}
