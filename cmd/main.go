package main

import (
	"os"
)

func main() {
	config := &Config{}
	rootCmd := newRootCommand(config)
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
