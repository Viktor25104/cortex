package api

import (
	"time"

	"cortex/scanner"
)

// ScanTask represents a scanning job managed by the API service.
type ScanTask struct {
	ID          string               `json:"id"`
	Status      string               `json:"status"`
	Hosts       []string             `json:"hosts"`
	Ports       string               `json:"ports"`
	Mode        string               `json:"mode"`
	Results     []scanner.ScanResult `json:"results,omitempty"`
	CreatedAt   time.Time            `json:"created_at"`
	CompletedAt *time.Time           `json:"completed_at,omitempty"`
	Error       string               `json:"error,omitempty"`
}

// CreateScanRequest is the payload for creating new scan tasks.
type CreateScanRequest struct {
	Hosts []string `json:"hosts" binding:"required,min=1" example:"scanme.nmap.org,127.0.0.1"`
	Ports string   `json:"ports" binding:"required" example:"22-80"`
	Mode  string   `json:"mode" binding:"required,oneof=connect syn udp" example:"connect"`
}
