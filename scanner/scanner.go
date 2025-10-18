package scanner

import (
	"net"
	"strconv"
	"time"
)

type ScanJob struct {
	Host string
	Port int
}

// ScanResult represents a single port scan result
type ScanResult struct {
	Host    string `json:"host"`
	Port    int    `json:"port"`
	State   string `json:"state"`
	Service string `json:"service,omitempty"`
}

func grabBanner(conn net.Conn) string {
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))

	_, err := conn.Write([]byte("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n"))
	if err != nil {
		return ""
	}

	buffer := make([]byte, 256)
	n, err := conn.Read(buffer)
	if err != nil {
		return ""
	}

	return string(buffer[:n])
}

// Worker processes scan jobs from the jobs channel and sends results through the results channel
func Worker(jobs <-chan ScanJob, results chan<- ScanResult) {
	for job := range jobs {
		address := job.Host + ":" + strconv.Itoa(job.Port)
		conn, err := net.DialTimeout("tcp", address, 2*time.Second)

		var result ScanResult
		if err != nil {
			result = ScanResult{Host: job.Host, Port: job.Port, State: "Closed"}
		} else {
			defer conn.Close()
			banner := grabBanner(conn)
			result = ScanResult{Host: job.Host, Port: job.Port, State: "Open", Service: banner}
		}
		results <- result
	}
}
