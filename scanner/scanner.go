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
	Host    string `json:"host"`
	Port    int    `json:"port"`
	State   string `json:"state"`
	Service string `json:"service,omitempty"`
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
