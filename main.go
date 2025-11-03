package main

import (
	"os"

	"cortex/api"
	"cortex/cli"
	"cortex/logging"
)

func main() {
	logging.Configure()
	if isServerMode(os.Args[1:]) {
		if err := api.Run(); err != nil {
			logging.Logger().Error("failed to start API server", "error", err)
			os.Exit(1)
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
