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
	Host  string `json:"host"`
	Port  int    `json:"port"`
	State string `json:"state"`
}

// Worker processes scan jobs from the jobs channel and sends results through the results channel
func Worker(jobs <-chan ScanJob, results chan<- ScanResult) {
	for job := range jobs {
		address := job.Host + ":" + strconv.Itoa(job.Port)
		_, err := net.DialTimeout("tcp", address, 2*time.Second)

		var state string
		if err != nil {
			state = "Closing"
		} else {
			state = "Opening"
		}
		results <- ScanResult{Host: job.Host, Port: job.Port, State: state}
	}
}
