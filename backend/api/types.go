package api

import (
	"time"

	"cortex/scanner"
)

// ScanTask represents a scanning job managed by the API service.
type ScanTask struct {
        // ID is the immutable identifier of the scan task (UUID v4).
        ID string `json:"id" format:"uuid" example:"a3f5c62e-1234-4f72-a84a-1c2d3e4f5678" description:"Immutable UUIDv4 identifier assigned when the task is accepted. Persist this value and reuse it for subsequent polling requests."`
        // Status reflects the asynchronous lifecycle state of the task.
        Status string `json:"status" enums:"pending,running,completed,failed" example:"pending" description:"Current processing state. pending indicates the request is queued, running signals active probing, completed denotes success with results attached, and failed highlights an unrecoverable worker-side issue."`
        // Hosts captures every hostname or IP submitted for the scan.
        Hosts []string `json:"hosts" example:"[\"scanme.nmap.org\",\"192.0.2.10\"]" description:"List of destination targets. Supports IPv4/IPv6 literals and resolvable domain names. The order is preserved so results can be mapped back to the original submission."`
        // Ports defines the requested port selection as comma-separated values and ranges.
        Ports string `json:"ports" example:"22,80,443,1000-1100" description:"Port expression combining single ports and inclusive ranges using commas (for example 22,80,443,1000-1100). Whitespace is ignored and duplicate ports are automatically de-duplicated by the scheduler."`
        // Mode determines the underlying probing strategy executed by workers.
        Mode string `json:"mode" enums:"connect,syn,udp" example:"syn" description:"Scanner transport mode. Use connect for TCP connect() handshakes, syn for half-open SYN scanning against TCP endpoints, or udp for stateless UDP datagram probes."`
        // Results becomes populated with port findings once the task completes.
        Results []scanner.ScanResult `json:"results,omitempty" example:"[{\\\"host\\\":\\\"scanme.nmap.org\\\",\\\"port\\\":443,\\\"state\\\":\\\"Open\\\",\\\"service\\\":\\\"https\\\"}]" description:"Collection of port states collected during scanning. Present only after the task reaches the completed status. The array is sorted by host then port for easy rendering."`
        // CreatedAt records when the task was created.
        CreatedAt time.Time `json:"created_at" format:"date-time" example:"2024-01-02T15:04:05Z" description:"Timestamp (UTC, RFC3339 format) when the API accepted the scan request."`
        // CompletedAt is set once the task transitions to a terminal state.
        CompletedAt *time.Time `json:"completed_at,omitempty" format:"date-time" example:"2024-01-02T15:06:30Z" description:"Timestamp (UTC, RFC3339 format) indicating when the task finished processing. Empty while the task is pending or running."`
        // Error contains context when a task fails.
        Error string `json:"error,omitempty" example:"failed to resolve target host" description:"Diagnostic message describing why the task entered the failed status. Present only when status equals failed."`
}

// CreateScanRequest is the payload for creating new scan tasks.
type CreateScanRequest struct {
        // Hosts enumerates every hostname or IP address the scanner should probe.
        Hosts []string `json:"hosts" binding:"required,min=1" example:"[\"scanme.nmap.org\",\"203.0.113.50\"]" description:"Targets to scan. Accepts IPv4/IPv6 addresses and domain names that resolve via DNS. Provide at least one entry; multiple hosts are processed concurrently."`
        // Ports expresses the desired port selection using comma-separated values and ranges.
        Ports string `json:"ports" binding:"required" example:"443,8443,10000-10100" description:"Combination of single ports and inclusive ranges (e.g. 80,443,1000-1050). Leave no spaces for best readability; ranges must use a hyphen."`
        // Mode selects which worker implementation will be used for probing.
        Mode string `json:"mode" binding:"required,oneof=connect syn udp" enums:"connect,syn,udp" example:"connect" description:"Scanning strategy. connect performs TCP connect() handshakes suitable for banner grabbing, syn uses half-open SYN probes for fast TCP discovery, udp sends UDP payloads to uncover datagram services."`
}

// ScanAcceptedResponse captures the asynchronous acknowledgement returned after job submission.
type ScanAcceptedResponse struct {
        // ID mirrors the queued task identifier returned to clients for polling.
        ID string `json:"id" format:"uuid" example:"a3f5c62e-1234-4f72-a84a-1c2d3e4f5678" description:"Identifier clients must supply to GET /scans/{id} when polling for status."`
        // Status is always pending immediately after acceptance.
        Status string `json:"status" enums:"pending" example:"pending" description:"Initial queue state assigned to every newly accepted scan request."`
}

// ErrorResponse provides a consistent structure for API error payloads.
type ErrorResponse struct {
        // Error is a human-readable explanation of why the request failed.
        Error string `json:"error" example:"task not found" description:"Human readable error message describing why the request was rejected. The value is localized for operators rather than end users."`
}
