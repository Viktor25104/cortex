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
	jsonOutput := flag.Bool("json", false, "Output results in JSON format")
	flag.Parse()

	args := flag.Args()
	if len(args) < 2 {
		fmt.Println("Usage: cortex [--json] host1 host2... startPort-endPort")
		fmt.Println("Example: cortex --json 127.0.0.1 scanme.nmap.org 22-80")
		return
	}

	portRange := args[len(args)-1]
	hosts := args[:len(args)-1]

	startPort, endPort, err := parsePortRange(portRange)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	scanResults := executeScanning(hosts, startPort, endPort)

	if *jsonOutput {
		outputJSON(scanResults)
	} else {
		outputPlainText(scanResults)
	}
}

// parsePortRange extracts start and end port from string format "start-end".
func parsePortRange(portRange string) (int, int, error) {
	parts := strings.Split(portRange, "-")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid port range format. Use startPort-endPort")
	}

	startPort, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("start port is not a number: %s", parts[0])
	}

	endPort, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("end port is not a number: %s", parts[1])
	}

	return startPort, endPort, nil
}

// executeScanning distributes scan jobs to workers and collects results.
func executeScanning(hosts []string, startPort int, endPort int) []scanner.ScanResult {
	jobs := make(chan scanner.ScanJob, 1000)
	totalRoutines := len(hosts) * (endPort - startPort + 1)
	results := make(chan scanner.ScanResult)

	// Start worker goroutines.
	for w := 0; w < 100; w++ {
		go scanner.Worker(jobs, results)
	}

	// Distribute scan jobs.
	for _, host := range hosts {
		for port := startPort; port <= endPort; port++ {
			jobs <- scanner.ScanJob{Host: host, Port: port}
		}
	}
	close(jobs)

	// Collect results.
	var scanResults []scanner.ScanResult
	for i := 0; i < totalRoutines; i++ {
		result := <-results
		scanResults = append(scanResults, result)
	}

	return scanResults
}

// outputJSON marshals and prints results in JSON format.
func outputJSON(results []scanner.ScanResult) {
	jsonData, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		fmt.Printf("Error encoding to JSON: %v\n", err)
		return
	}
	fmt.Println(string(jsonData))
}

// outputPlainText prints results in human-readable format.
// Displays banner information for open ports and status for all ports.
func outputPlainText(results []scanner.ScanResult) {
	for _, result := range results {
		if result.State == "Open" && result.Service != "" {
			bannerLine := extractFirstLine(result.Service)
			if len(bannerLine) > 100 {
				bannerLine = bannerLine[:100] + "..."
			}
			fmt.Printf("%s:%d - %s - %s\n", result.Host, result.Port, result.State, bannerLine)
		} else {
			fmt.Printf("%s:%d - %s\n", result.Host, result.Port, result.State)
		}
	}
}

// extractFirstLine returns the first line of a multi-line string.
func extractFirstLine(s string) string {
	lines := strings.Split(s, "\n")
	return lines[0]
}
