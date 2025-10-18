package scanner

import "fmt"

// ScanJob represents a single port scanning task.
type ScanJob struct {
	Host string
	Port int
}

// ScanResult represents the outcome of a port scan attempt.
type ScanResult struct {
	Host    string `json:"host"`
	Port    int    `json:"port"`
	State   string `json:"state"`
	Service string `json:"service,omitempty"`
}

// ScanMode specifies the type of port scanning to perform.
type ScanMode int

const (
	ModeConnect ScanMode = iota
	ModeSYN
	ModeUDP
)

// WorkerFunc is the function signature for scan workers.
// Each worker processes jobs and sends results.
type WorkerFunc func(jobs <-chan ScanJob, results chan<- ScanResult, cache *ProbeCache)

// ExecuteScan orchestrates the scanning process based on the specified mode.
// It validates prerequisites, routes to the correct worker, and collects results.
func ExecuteScan(hosts []string, startPort, endPort int, mode ScanMode, cache *ProbeCache) ([]ScanResult, error) {
	var worker WorkerFunc
	var workerCount int

	switch mode {
	case ModeConnect:
		worker = TCPConnectWorker
		workerCount = 100
	case ModeSYN:
		if err := InitSynScan(); err != nil {
			return nil, err
		}
		worker = TCPSynWorker
		workerCount = 50
	case ModeUDP:
		if err := InitUdpScan(); err != nil {
			return nil, err
		}
		worker = UDPWorker
		workerCount = 50
	default:
		return nil, fmt.Errorf("unknown scan mode")
	}

	return executeScan(hosts, startPort, endPort, worker, workerCount, cache), nil
}

// executeScan is the universal scanning orchestrator.
// It takes any worker function and runs concurrent scans.
func executeScan(hosts []string, startPort int, endPort int, worker WorkerFunc, workerCount int, cache *ProbeCache) []ScanResult {
	jobs := make(chan ScanJob, 1000)
	totalRoutines := len(hosts) * (endPort - startPort + 1)
	results := make(chan ScanResult, totalRoutines)

	// Start worker goroutines.
	for w := 0; w < workerCount; w++ {
		go worker(jobs, results, cache)
	}

	// Distribute scan jobs.
	for _, host := range hosts {
		for port := startPort; port <= endPort; port++ {
			jobs <- ScanJob{Host: host, Port: port}
		}
	}
	close(jobs)

	// Collect results.
	var scanResults []ScanResult
	for i := 0; i < totalRoutines; i++ {
		result := <-results
		scanResults = append(scanResults, result)
	}

	return scanResults
}
