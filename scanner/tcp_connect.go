package scanner

import (
	"net"
	"strconv"
	"strings"
	"time"
)

// TCPConnectWorker processes scan jobs using TCP Connect Scan.
// Establishes full connection and retrieves service banner if available.
func TCPConnectWorker(jobs <-chan ScanJob, results chan<- ScanResult) {
	for job := range jobs {
		address := job.Host + ":" + strconv.Itoa(job.Port)
		conn, err := net.DialTimeout("tcp", address, 2*time.Second)

		var result ScanResult
		if err != nil {
			result = ScanResult{Host: job.Host, Port: job.Port, State: "Closed"}
		} else {
			banner := grabBanner(conn, job.Port)
			result = ScanResult{Host: job.Host, Port: job.Port, State: "Open", Service: banner}
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
func grabBanner(conn net.Conn, port int) string {
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))

	var banner string

	switch port {
	case 80, 443, 8080, 8443, 3000, 5000:
		banner = fetchHTTPBanner(conn)
	case 21, 22, 25, 110, 143:
		banner = readAutoBanner(conn)
	default:
		return ""
	}

	return cleanBanner(banner)
}

// fetchHTTPBanner sends HTTP GET request and reads the response.
func fetchHTTPBanner(conn net.Conn) string {
	_, err := conn.Write([]byte("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n"))
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
