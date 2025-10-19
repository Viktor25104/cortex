package scanner

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"
)

// UDPWorker processes scan jobs using UDP scan method.
// Sends UDP probe packets and analyzes responses or ICMP error messages
// to determine port state. UDP scanning is inherently less reliable than
// TCP scanning due to the connectionless nature of the protocol.
// Note: cache parameter is unused in current implementation.
// Future enhancement: UDP probes from nmap-service-probes could be utilized.
func UDPWorker(jobs <-chan ScanJob, results chan<- ScanResult, cache *ProbeCache, wg *sync.WaitGroup) {
	_ = cache // Unused: UDP service detection not yet implemented
	for job := range jobs {
		state := performUdpScan(job.Host, job.Port)
		result := ScanResult{Host: job.Host, Port: job.Port, State: state}
		results <- result
		wg.Done()
	}
}

// performUdpScan executes a UDP scan on a single target port.
// Sends a UDP probe packet and analyzes the response to determine port state.
// Returns:
// - "Open": Service responded with data
// - "Closed": ICMP port unreachable received
// - "Open|Filtered": No response (timeout) - port may be open or filtered by firewall
func performUdpScan(host string, port int) string {
	address := host + ":" + strconv.Itoa(port)

	// Establish UDP connection with timeout
	conn, err := net.DialTimeout("udp", address, 2*time.Second)
	if err != nil {
		// Check for timeout error (handles wrapped errors properly)
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			return "Open|Filtered"
		}
		// Other errors (e.g., ICMP port unreachable) indicate closed port
		return "Closed"
	}
	defer conn.Close()

	// Set read deadline for response collection
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))

	// Send UDP probe packet (single null byte)
	_, err = conn.Write([]byte{0})
	if err != nil {
		return "Open|Filtered"
	}

	// Listen for service response or ICMP error messages
	buffer := make([]byte, 512)
	n, err := conn.Read(buffer)

	if err != nil {
		// Check for timeout error (handles wrapped errors properly)
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			return "Open|Filtered"
		}
		// Other errors (e.g., ICMP port unreachable) indicate closed port
		return "Closed"
	}

	// If we received response data, the port is definitively open
	if n > 0 {
		return "Open"
	}

	return "Open|Filtered"
}

// InitUdpScan validates that the system meets prerequisites for UDP scanning.
// Unlike SYN scanning, UDP scanning through net.Dial doesn't require elevated
// privileges in most cases. Performs basic network capability check.
// Returns error if basic networking is unavailable.
func InitUdpScan() error {
	// Verify basic network resolution capability
	// UDP scanning uses standard sockets, no special privileges needed
	_, err := net.LookupIP("localhost")
	if err != nil {
		return fmt.Errorf("UDP scan requires network resolution capability: %v", err)
	}

	return nil
}
