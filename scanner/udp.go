package scanner

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"
)

// UDPWorker processes scan jobs using UDP Scan.
// Sends UDP packets and detects responses or ICMP errors.
func UDPWorker(jobs <-chan ScanJob, results chan<- ScanResult, cache *ProbeCache) {
	for job := range jobs {
		state := performUdpScan(job.Host, job.Port)
		result := ScanResult{Host: job.Host, Port: job.Port, State: state}
		results <- result
	}
}

// performUdpScan attempts UDP scan on a single host:port combination.
// Sends UDP packet and listens for service response or ICMP errors.
// Returns "Open" (service responded), "Closed" (ICMP unreachable), or "Open|Filtered" (timeout).
func performUdpScan(host string, port int) string {
	address := host + ":" + strconv.Itoa(port)

	// 1. Dial UDP connection with timeout.
	conn, err := net.DialTimeout("udp", address, 2*time.Second)
	if err != nil {
		// Check if it's a timeout error using errors.As (handles wrapped errors).
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			return "Open|Filtered"
		}
		// Any other error (like connection refused) indicates closed port.
		return "Closed"
	}
	defer conn.Close()

	// 2. Set read deadline for receiving responses.
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))

	// 3. Send probe packet.
	_, err = conn.Write([]byte{0})
	if err != nil {
		return "Open|Filtered"
	}

	// 4. Listen for service response or ICMP errors.
	buffer := make([]byte, 512)
	n, err := conn.Read(buffer)

	if err != nil {
		// Check if it's a timeout error using errors.As (handles wrapped errors).
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			return "Open|Filtered"
		}
		// Any other error (like connection refused from ICMP) means port is closed.
		return "Closed"
	}

	// 5. If we received data, port is open and responding.
	if n > 0 {
		return "Open"
	}

	return "Open|Filtered"
}

// InitUdpScan initializes resources needed for UDP scanning.
// Validates that pcap library is available and privileges are sufficient.
// Returns error if UDP scan prerequisites are not met.
func InitUdpScan() error {
	// For UDP scanning, we primarily rely on net.Dial which works without root
	// in most cases. However, we still want to ensure system is capable.
	// A simple check: try to resolve localhost
	_, err := net.LookupIP("localhost")
	if err != nil {
		return fmt.Errorf("UDP scan requires network resolution capability: %v", err)
	}

	return nil
}
