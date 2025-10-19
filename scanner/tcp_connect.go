package scanner

import (
	"errors"
	"net"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// probeService performs intelligent service detection using probe-based fingerprinting.
// Reuses the already established connection to avoid connection failures and ensure consistency.
// Returns service name, raw response banner, and connection validity flag.
// If connectionValid is false, the connection was reset and port should be considered closed.
func probeService(conn net.Conn, cache *ProbeCache) (string, string, bool) {
	// Retrieve all TCP probes from cache
	tcpProbes := cache.GetTCPProbes()

	// First, check if connection is still alive by trying to read with very short timeout
	// This detects immediate RST from reverse proxies with no backend
	_ = conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	testBuffer := make([]byte, 1)
	_, err := conn.Read(testBuffer)

	// If we get a non-timeout error immediately, connection was reset
	if err != nil {
		var netErr net.Error
		if !errors.As(err, &netErr) || !netErr.Timeout() {
			// Non-timeout error means connection reset or closed
			return "", "", false
		}
		// Timeout is fine - just means no immediate data
	}

	// Try each probe on the existing connection
	for _, probe := range tcpProbes {
		// Send probe payload if available
		if len(probe.Data) > 0 {
			_, err := conn.Write(probe.Data)
			if err != nil {
				// Write failed - connection is dead
				return "", "", false
			}
		}

		// Set read timeout for response collection
		_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))

		// Collect server response
		buffer := make([]byte, 4096)
		n, err := conn.Read(buffer)

		if err != nil {
			// Check if it's a connection reset (not just timeout)
			var netErr net.Error
			if !errors.As(err, &netErr) || !netErr.Timeout() {
				// Connection was reset during probing
				return "", "", false
			}
			continue // Timeout - try next probe
		}

		if n == 0 {
			continue // Empty response - try next probe
		}

		response := buffer[:n]

		// Match response against this probe's service patterns
		for _, match := range probe.Matches {
			if match.Pattern.Match(response) {
				// Service identified successfully
				return match.ServiceName, string(response), true
			}
		}

		// Got a response but no match - return raw banner
		return "", string(response), true
	}

	// No service identified but connection is still valid
	return "", "", true
}

// TCPConnectWorker processes scan jobs using TCP Connect scan method.
// Establishes full TCP three-way handshake to verify port accessibility,
// then performs service detection using probe-based fingerprinting.
// Implements multi-level port state detection similar to nmap:
// - Closed: Connection actively refused (RST received)
// - Filtered: Timeout or no response (firewall blocking or accepting without backend)
// - Open: Connection accepted AND service responds
func TCPConnectWorker(jobs <-chan ScanJob, results chan<- ScanResult, cache *ProbeCache, wg *sync.WaitGroup) {
	for job := range jobs {
		address := job.Host + ":" + strconv.Itoa(job.Port)

		// Attempt TCP connection to determine basic accessibility
		conn, err := net.DialTimeout("tcp", address, 2*time.Second)

		var result ScanResult

		if err != nil {
			// Connection failed - need to determine if Closed or Filtered
			// Use the same error analysis approach as UDP scanner

			// Check for timeout error (indicates firewall dropping packets)
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Timeout() {
				// Timeout - packets are being silently dropped by firewall
				result = ScanResult{Host: job.Host, Port: job.Port, State: "Filtered"}
			} else if isConnectionRefused(err) {
				// Connection actively refused (RST) - port is definitively closed
				result = ScanResult{Host: job.Host, Port: job.Port, State: "Closed"}
			} else {
				// Other network errors - treat as filtered (unreachable, no route, etc.)
				result = ScanResult{Host: job.Host, Port: job.Port, State: "Filtered"}
			}
		} else {
			// TCP handshake succeeded - perform probe-based service identification
			serviceName, rawBanner, connValid := probeService(conn, cache)
			_ = conn.Close() // Close connection after probing

			// If connection was reset during probing, treat as closed
			// This handles reverse proxies that accept TCP but immediately RST
			if !connValid {
				result = ScanResult{Host: job.Host, Port: job.Port, State: "Closed"}
			} else {
				// Connection remained valid - port is OPEN
				serviceDescription := serviceName
				if serviceDescription == "" && rawBanner != "" {
					serviceDescription = rawBanner
				}
				result = ScanResult{Host: job.Host, Port: job.Port, State: "Open", Service: serviceDescription}
			}
		}

		results <- result
		wg.Done()
	}
}

// isConnectionRefused checks if the error is a connection refused error.
// Connection refused (RST packet) indicates the port is definitively closed.
func isConnectionRefused(err error) bool {
	// Check for syscall.ECONNREFUSED on Unix-like systems
	if errors.Is(err, syscall.ECONNREFUSED) {
		return true
	}

	// Check for Windows WSAECONNREFUSED error
	// On Windows, connection refused might appear as a different error
	errStr := err.Error()
	return strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "actively refused")
}
