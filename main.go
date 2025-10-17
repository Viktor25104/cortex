package main

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
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

	// 5. Calculate the total number of goroutines
	totalRoutines := len(hosts) * (endPort - startPort + 1)

	results := make(chan string)

	// 6. Outer loop over hosts
	for _, host := range hosts {
		// Inner loop over ports within the range
		for port := startPort; port <= endPort; port++ {
			go scanPort(port, host, results)
		}
	}

	// 7. Collect results from all goroutines
	for i := 0; i < totalRoutines; i++ {
		fmt.Println(<-results)
	}

}

func scanPort(port int, host string, results chan string) {
	result, err := net.DialTimeout("tcp", host+":"+strconv.Itoa(port), 2*time.Second)

	if err != nil {
		message := fmt.Sprintf("%s: The port %d isn't available", host, port)
		results <- message
	} else {
		message := fmt.Sprintf("%s: The port %d is available", host, port)
		results <- message
		_ = result.Close()
	}
}
