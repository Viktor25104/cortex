package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"strconv"
	"strings"

	"cortex/scanner"
)

func main() {
	// 1. Define the --json flag
	jsonOutput := flag.Bool("json", false, "Output results in JSON format")
	flag.Parse()

	// 2. Get targets (hosts and ports)
	args := flag.Args()
	if len(args) < 2 {
		fmt.Println("Usage: cortex [--json] host1 host2... startPort-endPort")
		fmt.Println("Example: cortex --json 127.0.0.1 scanme.nmap.org 22-80")
		return
	}

	// Extract port range (last argument)
	portRange := args[len(args)-1]

	// Extract hosts list (all arguments except the last one)
	hosts := args[:len(args)-1]

	// 3. Parse port range
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
	results := make(chan scanner.ScanResult)

	// Start 100 workers
	for w := 0; w < 100; w++ {
		go scanner.Worker(jobs, results)
	}

	// Send scan jobs to the channel
	for _, host := range hosts {
		for port := startPort; port <= endPort; port++ {
			jobs <- scanner.ScanJob{Host: host, Port: port}
		}
	}
	close(jobs)

	// 4. Collect all results into a slice
	var scanResults []scanner.ScanResult
	for i := 0; i < totalRoutines; i++ {
		result := <-results
		scanResults = append(scanResults, result)
	}

	// 5. Output results based on the flag
	if *jsonOutput {
		// Encode the slice as JSON
		jsonData, err := json.MarshalIndent(scanResults, "", "  ")
		if err != nil {
			fmt.Println("Error encoding to JSON:", err)
			return
		}
		fmt.Println(string(jsonData))
	} else {
		// Output as plain text
		for _, result := range scanResults {
			fmt.Printf("%s:%d - %s\n", result.Host, result.Port, result.State)
		}
	}
}
