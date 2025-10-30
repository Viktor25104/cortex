package main

import (
	"log"
	"os"

	"cortex/api"
	"cortex/cli"
)

func main() {
	if isServerMode(os.Args[1:]) {
		if err := api.Run(); err != nil {
			log.Fatalf("failed to start API server: %v", err)
		}
		return
	}

	cli.Run()
}

func isServerMode(args []string) bool {
	for _, arg := range args {
		if arg == "--server" {
			return true
		}
	}
	return false
}
