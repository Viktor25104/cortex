package cli

import (
	"cortex/scanner"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"strconv"
	"strings"
)

// Run is the main entry point for the CLI application.
// It parses command-line flags and arguments, validates them,
// and orchestrates the scanning process.
func Run() {
	jsonOutput := flag.Bool("json", false, "Output results in JSON format")
	synScan := flag.Bool("sS", false, "Use SYN scan (requires root/admin)")
	flag.BoolVar(synScan, "syn-scan", false, "Use SYN scan (requires root/admin)")
	udpScan := flag.Bool("sU", false, "Use UDP scan")
	flag.BoolVar(udpScan, "udp-scan", false, "Use UDP scan")
	flag.Parse()

	// Load probes for service detection
	var probeCache *scanner.ProbeCache
	probes, stats, err := scanner.LoadProbes("nmap-service-probes")
	if err != nil {
		log.Fatalf("Critical error loading probes file: %v", err)
	}

	// Display parsing errors if any occurred during probe file parsing
	if len(stats.ErrorLines) > 0 {
		fmt.Println("--- Warnings during probe file parsing ---")
		for _, e := range stats.ErrorLines {
			fmt.Printf("Line %d: %s\n", e.LineNumber, e.Message)
		}
		fmt.Println("----------------------------------------")
	}

	// Display final probe loading statistics
	fmt.Println("--- Probe Loading Summary ---")
	fmt.Printf("Total lines processed: %d\n", stats.TotalLines)
	fmt.Printf("Successfully loaded probes: %d\n", stats.ProbeCount)
	fmt.Printf("Successfully loaded match rules: %d\n", stats.MatchCount)
	fmt.Printf("Lines with parsing errors: %d\n", len(stats.ErrorLines))
	fmt.Println("---------------------------")

	probeCache = scanner.NewProbeCache(probes)

	args := flag.Args()
	if len(args) < 2 {
		printUsage()
		return
	}

	// Determine scan worker based on flags
	if *synScan && *udpScan {
		fmt.Println("Error: Cannot use multiple scan modes simultaneously. Choose one: Connect, SYN (-sS), or UDP (-sU)")
		return
	}

	var workerFunc scanner.WorkerFunc
	var workerCount int

	if *synScan {
		if err := scanner.InitSynScan(); err != nil {
			log.Fatalf("SYN scan initialization failed: %v\nTry running with sudo.", err)
		}
		workerFunc = scanner.TCPSynWorker
		workerCount = 50
	} else if *udpScan {
		if err := scanner.InitUdpScan(); err != nil {
			log.Fatalf("UDP scan initialization failed: %v\nTry running with sudo.", err)
		}
		workerFunc = scanner.UDPWorker
		workerCount = 50
	} else {
		// Default: TCP Connect scan
		workerFunc = scanner.TCPConnectWorker
		workerCount = 100
	}

	portRange := args[len(args)-1]
	hosts := args[:len(args)-1]

	startPort, endPort, err := parsePortRange(portRange)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Execute the scan with probe cache
	scanResults := scanner.ExecuteScan(hosts, startPort, endPort, workerFunc, workerCount, probeCache)

	// Output results
	if *jsonOutput {
		outputJSON(scanResults)
	} else {
		outputPlainText(scanResults)
	}
}

// printUsage displays the help message.
func printUsage() {
	fmt.Println("Usage: cortex [--json] [-sS|--syn-scan|-sU|--udp-scan] host1 host2... startPort-endPort")
	fmt.Println("Example: cortex --json 127.0.0.1 scanme.nmap.org 22-80")
	fmt.Println("Example: cortex -sS 127.0.0.1 22-80")
	fmt.Println("Example: cortex -sU 127.0.0.1 53-53")
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
// Displays service information for open ports when available.
func outputPlainText(results []scanner.ScanResult) {
	for _, result := range results {
		// Print results for all port states: Open, Closed, Filtered
		if result.Service != "" {
			// If service information is available, display it
			bannerLine := extractFirstLine(result.Service)
			if len(bannerLine) > 100 {
				bannerLine = bannerLine[:100] + "..."
			}
			fmt.Printf("%s:%d - %s - %s\n", result.Host, result.Port, result.State, bannerLine)
		} else {
			// Otherwise, show only the port state
			fmt.Printf("%s:%d - %s\n", result.Host, result.Port, result.State)
		}
	}
}

// extractFirstLine extracts the first line from a multi-line string.
func extractFirstLine(s string) string {
	lines := strings.Split(s, "\n")
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0])
	}
	return s
}
