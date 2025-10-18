package cli

import (
	"cortex/scanner"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Run is the main entry point for the CLI application.
// It parses command-line flags and arguments, validates them,
// and orchestrates the scanning process.
func Run() {
	jsonOutput := flag.Bool("json", false, "Output results in JSON format")
	synScan := flag.Bool("sS", false, "Use SYN scan (requires root/admin)")
	flag.BoolVar(synScan, "syn-scan", *synScan, "Use SYN scan (requires root/admin)")
	udpScan := flag.Bool("sU", false, "Use UDP scan (requires root/admin)")
	flag.BoolVar(udpScan, "udp-scan", *udpScan, "Use UDP scan (requires root/admin)")
	flag.Parse()

	args := flag.Args()
	if len(args) < 2 {
		printUsage()
		return
	}

	// Validate that only one scan mode is selected
	scanModeCount := 0
	var selectedMode scanner.ScanMode
	if *synScan {
		scanModeCount++
		selectedMode = scanner.ModeSYN
	}
	if *udpScan {
		scanModeCount++
		selectedMode = scanner.ModeUDP
	}

	if scanModeCount > 1 {
		fmt.Println("Error: Cannot use multiple scan modes simultaneously. Choose one: Connect, SYN (-sS), or UDP (-sU)")
		return
	}

	// Default to Connect scan if no mode specified
	if scanModeCount == 0 {
		selectedMode = scanner.ModeConnect
	}

	portRange := args[len(args)-1]
	hosts := args[:len(args)-1]

	startPort, endPort, err := parsePortRange(portRange)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Execute the scan
	scanResults, err := scanner.ExecuteScan(hosts, startPort, endPort, selectedMode)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		if selectedMode == scanner.ModeSYN {
			fmt.Println("SYN scan requires elevated privileges. Try: sudo cortex -sS ...")
		} else if selectedMode == scanner.ModeUDP {
			fmt.Println("UDP scan requires elevated privileges. Try: sudo cortex -sU ...")
		}
		os.Exit(1)
	}

	// Output results
	if *jsonOutput {
		outputJSON(scanResults)
	} else {
		outputPlainText(scanResults, selectedMode != scanner.ModeConnect)
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
// Displays banner information for open ports and status for all ports.
// Adapts output based on scan mode (Connect vs SYN/UDP).
func outputPlainText(results []scanner.ScanResult, isStealth bool) {
	for _, result := range results {
		if result.State == "Open" && result.Service != "" && !isStealth {
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
