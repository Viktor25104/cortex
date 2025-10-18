package main

import (
	"cortex/scanner"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func main() {
	jsonOutput := flag.Bool("json", false, "Output results in JSON format")
	synScan := flag.Bool("sS", false, "Use SYN scan (requires root/admin)")
	flag.BoolVar(synScan, "syn-scan", *synScan, "Use SYN scan (requires root/admin)")
	udpScan := flag.Bool("sU", false, "Use UDP scan (requires root/admin)")
	flag.BoolVar(udpScan, "udp-scan", *udpScan, "Use UDP scan (requires root/admin)")
	flag.Parse()

	args := flag.Args()
	if len(args) < 2 {
		fmt.Println("Usage: cortex [--json] [-sS|--syn-scan|-sU|--udp-scan] host1 host2... startPort-endPort")
		fmt.Println("Example: cortex --json 127.0.0.1 scanme.nmap.org 22-80")
		fmt.Println("Example: cortex -sS 127.0.0.1 22-80")
		fmt.Println("Example: cortex -sU 127.0.0.1 53-53")
		return
	}

	scanModeCount := 0
	if *synScan {
		scanModeCount++
	}
	if *udpScan {
		scanModeCount++
	}

	if scanModeCount > 1 {
		fmt.Println("Error: Cannot use multiple scan modes simultaneously. Choose one: Connect, SYN (-sS), or UDP (-sU)")
		return
	}

	portRange := args[len(args)-1]
	hosts := args[:len(args)-1]

	startPort, endPort, err := parsePortRange(portRange)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	var scanResults []scanner.ScanResult

	if *synScan {
		// Validate SYN scan prerequisites.
		if err := scanner.InitSynScan(); err != nil {
			fmt.Printf("Error: %v\n", err)
			fmt.Println("SYN scan requires elevated privileges. Try: sudo cortex -sS ...")
			os.Exit(1)
		}
		scanResults = executeSynScanning(hosts, startPort, endPort)
	} else if *udpScan {
		// Validate UDP scan prerequisites.
		if err := scanner.InitUdpScan(); err != nil {
			fmt.Printf("Error: %v\n", err)
			fmt.Println("UDP scan requires elevated privileges. Try: sudo cortex -sU ...")
			os.Exit(1)
		}
		scanResults = executeUdpScanning(hosts, startPort, endPort)
	} else {
		scanResults = executeConnectScanning(hosts, startPort, endPort)
	}

	if *jsonOutput {
		outputJSON(scanResults)
	} else {
		outputPlainText(scanResults, *synScan || *udpScan)
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

// executeConnectScanning performs TCP Connect Scan.
// Establishes full connection to each port and retrieves banners.
func executeConnectScanning(hosts []string, startPort int, endPort int) []scanner.ScanResult {
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

// executeSynScanning performs SYN Scan (stealth mode).
// Sends SYN packets without completing TCP handshake.
// No banner grabbing in SYN mode - only port state detection.
func executeSynScanning(hosts []string, startPort int, endPort int) []scanner.ScanResult {
	jobs := make(chan scanner.ScanJob, 1000)
	totalRoutines := len(hosts) * (endPort - startPort + 1)
	results := make(chan scanner.ScanResult)

	// Start SYN worker goroutines.
	workerCount := 50 // Fewer workers for SYN scan to avoid overwhelming network.
	for w := 0; w < workerCount; w++ {
		go scanner.SynWorker(jobs, results)
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
// Adapts output based on scan mode (Connect vs SYN).
func outputPlainText(results []scanner.ScanResult, isSynScan bool) {
	for _, result := range results {
		if result.State == "Open" && result.Service != "" && !isSynScan {
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

// executeUdpScanning performs UDP Scan.
// Sends UDP packets and detects responses.
func executeUdpScanning(hosts []string, startPort int, endPort int) []scanner.ScanResult {
	jobs := make(chan scanner.ScanJob, 1000)
	totalRoutines := len(hosts) * (endPort - startPort + 1)
	results := make(chan scanner.ScanResult)

	// Start UDP worker goroutines.
	workerCount := 50
	for w := 0; w < workerCount; w++ {
		go scanner.UdpWorker(jobs, results)
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
