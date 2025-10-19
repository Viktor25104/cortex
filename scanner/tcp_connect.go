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
// For each probe, it establishes a fresh connection to ensure clean state and accurate results.
// This "one probe - one connection" approach mirrors nmap's methodology and prevents
// issues with buffered data or connection state corruption.
// Returns service name and raw response banner.
func probeService(host string, port int, cache *ProbeCache) (string, string) {
	// Retrieve all TCP probes from cache
	tcpProbes := cache.GetTCPProbes()
	address := host + ":" + strconv.Itoa(port)

	// Try each probe with a fresh connection
	for _, probe := range tcpProbes {
		// Establish new connection for this probe
		conn, err := net.DialTimeout("tcp", address, 2*time.Second)
		if err != nil {
			continue // Skip to next probe if connection fails
		}

		// Send probe payload if available
		if len(probe.Data) > 0 {
			_, err := conn.Write(probe.Data)
			if err != nil {
				_ = conn.Close()
				continue // Skip to next probe on send failure
			}
		}

		// Set read timeout for response collection
		_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))

		// Collect server response
		buffer := make([]byte, 4096)
		n, err := conn.Read(buffer)
		_ = conn.Close() // Always close connection after probe

		if err != nil || n == 0 {
			continue // Skip to next probe on read failure or empty response
		}

		response := buffer[:n]

		// Match response against this probe's service patterns
		for _, match := range probe.Matches {
			if match.Pattern.Match(response) {
				// Service identified successfully
				return match.ServiceName, string(response)
			}
		}
	}

	// No service identified
	return "", ""
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
			// TCP handshake succeeded - close initial connection
			_ = conn.Close()

			// Perform probe-based service identification
			serviceName, rawBanner := probeService(job.Host, job.Port, cache)

			// Multi-level state detection:
			if serviceName != "" || rawBanner != "" {
				// Service responded - port is truly open with active service
				serviceDescription := serviceName
				if serviceDescription == "" {
					serviceDescription = rawBanner
				}
				result = ScanResult{Host: job.Host, Port: job.Port, State: "Open", Service: serviceDescription}
			} else {
				// TCP handshake succeeded but no service response
				// This indicates a reverse proxy/firewall accepting connections
				// but with no backend service running on this port
				result = ScanResult{Host: job.Host, Port: job.Port, State: "Filtered"}
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
