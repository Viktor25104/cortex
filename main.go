package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"cortex/scanner"
)

func main() {

	// 1. Check if there are enough arguments
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run main.go host1 host2 ... startPort-endPort")
		fmt.Println("Example: go run main.go 127.0.0.1 scanme.nmap.org 22-80")
		return
	}

	// 2. Extract the port range (last argument)
	portRange := os.Args[len(os.Args)-1]

	// 3. Extract the list of hosts (from the second argument to the one before last)
	hosts := os.Args[1 : len(os.Args)-1]

	// 4. Parse the port range
	parts := strings.Split(portRange, "-")
	if len(parts) != 2 {
		fmt.Println("Error: invalid port range format. Use startPort-endPort")
		return
	}

	startPort, err := strconv.Atoi(parts[0])
	if err != nil {
		fmt.Println("Error: start port is not a number:", parts[0])
		return
	}

	endPort, err := strconv.Atoi(parts[1])
	if err != nil {
		fmt.Println("Error: end port is not a number:", parts[1])
		return
	}

	jobs := make(chan scanner.ScanJob, 1000)
	totalRoutines := len(hosts) * (endPort - startPort + 1)
	results := make(chan string)

	for w := 0; w < 100; w++ {
		go scanner.Worker(jobs, results)
	}

	for _, host := range hosts {
		for port := startPort; port <= endPort; port++ {
			jobs <- scanner.ScanJob{Host: host, Port: port}
		}
	}
	close(jobs)

	for i := 0; i < totalRoutines; i++ {
		fmt.Println(<-results)
	}
}
