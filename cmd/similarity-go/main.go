package main

import (
	"os"
)

func main() {
	args := &CLIArgs{}
	rootCmd := newRootCommand(args)
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
