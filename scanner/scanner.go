package scanner

import (
	"fmt"
	"net"
	"strconv"
	"time"
)

type ScanJob struct {
	Host string
	Port int
}

func Worker(jobs <-chan ScanJob, results chan<- string) {
	for job := range jobs {
		address := job.Host + ":" + strconv.Itoa(job.Port)
		_, err := net.DialTimeout("tcp", address, 2*time.Second)

		var message string
		if err != nil {
			message = fmt.Sprintf("%s:%d - Closed/Unavailable (%v)", job.Host, job.Port, err)
		} else {
			message = fmt.Sprintf("%s:%d - Opening", job.Host, job.Port)
		}
		results <- message
	}
}
