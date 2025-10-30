package api

import (
	"log"
	"strings"
	"sync"
	"time"

	"cortex/scanner"
)

var (
	synInitOnce sync.Once
	synInitErr  error

	udpInitOnce sync.Once
	udpInitErr  error
)

// StartWorkers launches background goroutines that process scan tasks.
func StartWorkers(store TaskStore, probeCache *scanner.ProbeCache, numWorkers int) {
	for i := 0; i < numWorkers; i++ {
		go workerLoop(store, probeCache)
	}
}

func workerLoop(store TaskStore, probeCache *scanner.ProbeCache) {
	for {
		taskID, err := store.PopFromQueue()
		if err != nil {
			log.Printf("worker: failed to pop task: %v", err)
			time.Sleep(time.Second)
			continue
		}

		task, err := store.GetTask(taskID)
		if err != nil {
			if err == ErrTaskNotFound {
				log.Printf("worker: task %s disappeared", taskID)
				continue
			}
			log.Printf("worker: failed to load task %s: %v", taskID, err)
			continue
		}

		task.Status = "running"
		task.Error = ""
		task.Results = nil
		task.CompletedAt = nil
		if err := store.UpdateTask(task); err != nil {
			log.Printf("worker: failed to set task %s running: %v", taskID, err)
			continue
		}

		startPort, endPort, err := parsePortRange(task.Ports)
		if err != nil {
			failTask(task, store, err)
			continue
		}

		workerFunc, workerCount, err := selectWorker(task.Mode)
		if err != nil {
			failTask(task, store, err)
			continue
		}

		results := scanner.ExecuteScan(task.Hosts, startPort, endPort, workerFunc, workerCount, probeCache)

		task.Status = "completed"
		task.Results = results
		now := time.Now().UTC()
		task.CompletedAt = &now

		if err := store.UpdateTask(task); err != nil {
			log.Printf("worker: failed to update task %s: %v", task.ID, err)
		}
	}
}

func failTask(task *ScanTask, store TaskStore, err error) {
	log.Printf("worker: task %s failed: %v", task.ID, err)
	task.Status = "failed"
	task.Error = err.Error()
	task.Results = nil
	now := time.Now().UTC()
	task.CompletedAt = &now
	if updateErr := store.UpdateTask(task); updateErr != nil {
		log.Printf("worker: failed to persist failed task %s: %v", task.ID, updateErr)
	}
}

func selectWorker(mode string) (scanner.WorkerFunc, int, error) {
	switch strings.ToLower(mode) {
	case "syn":
		synInitOnce.Do(func() {
			synInitErr = scanner.InitSynScan()
		})
		if synInitErr != nil {
			return nil, 0, synInitErr
		}
		return scanner.TCPSynWorker, 50, nil
	case "udp":
		udpInitOnce.Do(func() {
			udpInitErr = scanner.InitUdpScan()
		})
		if udpInitErr != nil {
			return nil, 0, udpInitErr
		}
		return scanner.UDPWorker, 50, nil
	case "connect":
		fallthrough
	default:
		return scanner.TCPConnectWorker, 100, nil
	}
}
