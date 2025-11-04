package api

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/gin-gonic/gin"
)

// Server bundles dependencies for HTTP handlers.
type Server struct {
	store TaskStore
}

// NewServer creates a new API server instance.
func NewServer(store TaskStore) *Server {
	return &Server{store: store}
}

// RegisterRoutes attaches handlers to the provided Gin router group.
func (s *Server) RegisterRoutes(routes gin.IRoutes) {
	routes.POST("/scans", s.createScanHandler)
	routes.GET("/scans/:id", s.getScanHandler)
}

var uuidV4Pattern = regexp.MustCompile(`^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[1-5][a-fA-F0-9]{3}-[abAB89][a-fA-F0-9]{3}-[a-fA-F0-9]{12}$`)

// @Summary      Create a new scan task
// @Description  Submit a scan definition and let Cortex execute it asynchronously. The handler validates input, persists the task, and enqueues it for background workers before returning a UUID.
// @Description  **Lifecycle**: POST /scans immediately answers with HTTP 202 Accepted plus the task identifier. Clients must poll GET /scans/{id} to observe status transitions (pending → running → completed/failed). Actual port findings are attached only after completion.
// @Description  **Common pitfalls**: malformed JSON, unsupported modes, or exceeding rate limits will return structured error responses containing a human-readable explanation.
// @Tags         Scans
// @Accept       json
// @Produce      json
// @Param        scanRequest  body      CreateScanRequest      true  "Scan request parameters"
// @Success      202          {object}  ScanAcceptedResponse  "Scan accepted. Poll GET /scans/{id} to track progress. Example: {\"id\":\"a3f5c62e-1234-4f72-a84a-1c2d3e4f5678\",\"status\":\"pending\"}"
// @Failure      400          {object}  ErrorResponse         "Malformed JSON body or failed validation. Example: {\"error\":\"invalid request payload: validation failed on 'mode'\"}"
// @Failure      401          {object}  ErrorResponse         "Missing or incorrect API key. Example: {\"error\":\"unauthorized\"}"
// @Failure      429          {object}  ErrorResponse         "Rate limit exceeded for the calling client. Example: {\"error\":\"rate limit exceeded\"}"
// @Failure      500          {object}  ErrorResponse         "Internal error while persisting or queueing the task. Example: {\"error\":\"failed to persist task\"}"
// @Security     ApiKeyAuth
// @Router       /scans [post]
func (s *Server) createScanHandler(c *gin.Context) {
	var req CreateScanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: fmt.Sprintf("invalid request payload: %v", err)})
		return
	}

	taskID, err := generateUUID()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to generate task id"})
		return
	}

	task := &ScanTask{
		ID:        taskID,
		Status:    "pending",
		Hosts:     req.Hosts,
		Ports:     req.Ports,
		Mode:      req.Mode,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.store.CreateTask(task); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to persist task"})
		return
	}

	if err := s.store.PushToQueue(task.ID); err != nil {
		task.Status = "failed"
		task.Error = "failed to queue task"
		now := time.Now().UTC()
		task.CompletedAt = &now
		_ = s.store.UpdateTask(task)

		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to queue task"})
		return
	}

	c.JSON(http.StatusAccepted, ScanAcceptedResponse{ID: task.ID, Status: task.Status})
}

// @Summary      Get scan status and results
// @Description  Retrieve a live snapshot of a scan task. Supply the UUID obtained from POST /scans and poll this endpoint until the lifecycle reaches completed.
// @Description  **Polling guidance**: responses with status pending or running will include metadata but results remains empty. Once the task is completed, results contains every observed port state and optional service fingerprints. If the task fails, the error field clarifies the reason.
// @Description  **Error handling**: invalid UUIDs, missing authorization, rate limiting, or unknown tasks all return structured ErrorResponse payloads so clients can adjust behavior programmatically.
// @Tags         Scans
// @Produce      json
// @Param        id   path      string      true  "Scan Task ID (UUID v4)"
// @Success      200  {object}  ScanTask    "Current task snapshot including results when completed. Example: {\"id\":\"a3f5c62e-1234-4f72-a84a-1c2d3e4f5678\",\"status\":\"completed\",\"results\":[{\"host\":\"scanme.nmap.org\",\"port\":443,\"state\":\"Open\",\"service\":\"https\"}]}"
// @Failure      400  {object}  ErrorResponse  "Malformed task identifier. Example: {\"error\":\"invalid task id format\"}"
// @Failure      401  {object}  ErrorResponse  "Missing or incorrect API key. Example: {\"error\":\"unauthorized\"}"
// @Failure      404  {object}  ErrorResponse  "Task with the provided ID does not exist. Example: {\"error\":\"task not found\"}"
// @Failure      429  {object}  ErrorResponse  "Rate limit exceeded for the calling client. Example: {\"error\":\"rate limit exceeded\"}"
// @Failure      500  {object}  ErrorResponse  "Internal error when loading the task. Example: {\"error\":\"failed to load task\"}"
// @Security     ApiKeyAuth
// @Router       /scans/{id} [get]
func (s *Server) getScanHandler(c *gin.Context) {
	id := c.Param("id")
	if !uuidV4Pattern.MatchString(id) {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid task id format"})
		return
	}
	task, err := s.store.GetTask(id)
	if err != nil {
		if err == ErrTaskNotFound {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "task not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to load task"})
		return
	}

	c.JSON(http.StatusOK, task)
}

func generateUUID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	// Variant bits; version 4 UUID.
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}
