package scanner

import (
	"sync"
)

// ScanJob represents a single port scanning task.
type ScanJob struct {
	Host string
	Port int
}

// ScanResult represents the outcome of a port scan attempt.
type ScanResult struct {
        Host    string `json:"host" example:"scanme.nmap.org" description:"Target host that produced the observation. Mirrors the input host field so clients can join results back to their original request."`
        Port    int    `json:"port" example:"443" description:"Network port that was probed. Expressed as an integer in the 0-65535 range."`
        State   string `json:"state" enums:"Open,Closed,Filtered" example:"Open" description:"Resulting port disposition derived from worker probes. Open indicates a responsive service, Closed means the port rejected connections, and Filtered signifies intermediary packet filtering."`
        Service string `json:"service,omitempty" example:"http (nginx)" description:"Optional service fingerprint (if detected) describing application protocol and banner. Empty when the probe could not identify an application."`
}

// WorkerFunc is the signature for scanner worker functions.
type WorkerFunc func(jobs <-chan ScanJob, results chan<- ScanResult, cache *ProbeCache, wg *sync.WaitGroup)

// ExecuteScan is the universal scan orchestrator.
// It manages workers, distributes tasks, and collects results.
func ExecuteScan(hosts []string, startPort int, endPort int, worker WorkerFunc, workerCount int, cache *ProbeCache) []ScanResult {
	var wg sync.WaitGroup
	jobs := make(chan ScanJob, 1000)
	totalJobs := len(hosts) * (endPort - startPort + 1)
	results := make(chan ScanResult, totalJobs)

	for w := 0; w < workerCount; w++ {
		go worker(jobs, results, cache, &wg)
	}

	wg.Add(totalJobs)
	go func() {
		for _, host := range hosts {
			for port := startPort; port <= endPort; port++ {
				jobs <- ScanJob{Host: host, Port: port}
			}
		}
		close(jobs)
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	// Pre-allocate slice with exact capacity to avoid reallocations
	scanResults := make([]ScanResult, 0, totalJobs)
	for result := range results {
		scanResults = append(scanResults, result)
	}

	return scanResults
}
