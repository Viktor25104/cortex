package scanner

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// probeService - the final intelligent banner collector.
// Sends probe data and matches response against probe patterns.
func probeService(conn net.Conn, host string, port int, cache *ProbeCache) string {
	if cache == nil || len(cache.GetTCPProbes()) == 0 {
		// If no cache, fallback to old simple method.
		return grabBanner(conn, host, port)
	}

	// Get all TCP probes from cache
	tcpProbes := cache.GetTCPProbes()

	// Try each probe in sequence
	for _, probe := range tcpProbes {
		// If probe has data to send, send it.
		if len(probe.Data) > 0 {
			_, err := conn.Write(probe.Data)
			if err != nil {
				continue // If failed to send, try next probe
			}
		}

		// Set read deadline for this specific probe
		_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))

		// Read response from server
		buffer := make([]byte, 4096)
		n, err := conn.Read(buffer)
		if err != nil || n == 0 {
			continue // If failed to read or empty response, try next probe
		}

		response := buffer[:n]

		// Match response ONLY against patterns of this specific probe
		for _, match := range probe.Matches {
			if match.Pattern.Match(response) {
				// Found a match!
				// Return service name. (Later can add version info)
				return match.ServiceName
			}
		}
	}

	// If no probe matched, return empty string.
	return ""
}

// TCPConnectWorker processes scan jobs using TCP Connect Scan.
// Establishes full connection and retrieves service banner if available.
func TCPConnectWorker(jobs <-chan ScanJob, results chan<- ScanResult, cache *ProbeCache) {
	for job := range jobs {
		address := job.Host + ":" + strconv.Itoa(job.Port)
		conn, err := net.DialTimeout("tcp", address, 2*time.Second)

		var result ScanResult
		if err != nil {
			result = ScanResult{Host: job.Host, Port: job.Port, State: "Closed"}
		} else {
			service := probeService(conn, job.Host, job.Port, cache)
			result = ScanResult{Host: job.Host, Port: job.Port, State: "Open", Service: service}
			_ = conn.Close()
		}
		results <- result
	}
}

// grabBanner retrieves service banner based on port protocol.
// Implements protocol-specific communication strategies:
// - HTTP ports: send GET request and read response
// - SSH/FTP/SMTP: read auto-generated banner
// - Others: return empty string
func grabBanner(conn net.Conn, host string, port int) string {
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))

	var banner string

	switch port {
	case 80, 443, 8080, 8443, 3000, 5000:
		banner = fetchHTTPBanner(conn, host)
	case 21, 22, 25, 110, 143:
		banner = readAutoBanner(conn)
	default:
		return ""
	}

	return cleanBanner(banner)
}

// fetchHTTPBanner sends HTTP GET request and reads the response.
func fetchHTTPBanner(conn net.Conn, host string) string {
	request := fmt.Sprintf("GET / HTTP/1.1\r\nHost: %s\r\n\r\n", host)
	_, err := conn.Write([]byte(request))

	if err != nil {
		return ""
	}

	buffer := make([]byte, 512)
	n, err := conn.Read(buffer)
	if err != nil {
		return ""
	}

	return string(buffer[:n])
}

// readAutoBanner reads banner sent by service upon connection.
// Used for SSH, FTP, SMTP, POP3, IMAP protocols.
func readAutoBanner(conn net.Conn) string {
	buffer := make([]byte, 512)
	n, err := conn.Read(buffer)
	if err != nil {
		return ""
	}

	return string(buffer[:n])
}

// cleanBanner normalizes banner format by replacing line endings.
func cleanBanner(banner string) string {
	cleaned := strings.ReplaceAll(banner, "\r\n", "\n")
	cleaned = strings.ReplaceAll(cleaned, "\r", "\n")
	return cleaned
}
