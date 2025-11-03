package api

import (
	"crypto/rand"
	"fmt"
	"net/http"
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

// @Summary      Create a new scan task
// @Description  Accepts a scan request, queues it for processing, and returns a task ID.
// @Tags         Scans
// @Accept       json
// @Produce      json
// @Param        scanRequest  body      CreateScanRequest  true  "Scan Request Parameters"
// @Success      202          {object}  gin.H              "Scan task accepted"
// @Failure      400          {object}  gin.H              "Invalid request payload"
// @Failure      401          {object}  gin.H              "Unauthorized"
// @Failure      429          {object}  gin.H              "Rate limit exceeded"
// @Failure      500          {object}  gin.H              "Internal server error"
// @Security     ApiKeyAuth
// @Router       /scans [post]
func (s *Server) createScanHandler(c *gin.Context) {
	var req CreateScanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	taskID, err := generateUUID()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate task id"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to persist task"})
		return
	}

	if err := s.store.PushToQueue(task.ID); err != nil {
		task.Status = "failed"
		task.Error = "failed to queue task"
		now := time.Now().UTC()
		task.CompletedAt = &now
		_ = s.store.UpdateTask(task)

		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to queue task"})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"id":     task.ID,
		"status": task.Status,
	})
}

// @Summary      Get scan status and results
// @Description  Retrieves the complete details of a scan task by its ID.
// @Tags         Scans
// @Produce      json
// @Param        id   path      string     true  "Scan Task ID (UUID)"
// @Success      200  {object}  ScanTask   "Full scan task object with results"
// @Failure      404  {object}  gin.H      "Task not found"
// @Failure      401  {object}  gin.H      "Unauthorized"
// @Failure      429  {object}  gin.H      "Rate limit exceeded"
// @Failure      500  {object}  gin.H      "Internal server error"
// @Security     ApiKeyAuth
// @Router       /scans/{id} [get]
func (s *Server) getScanHandler(c *gin.Context) {
	id := c.Param("id")
	task, err := s.store.GetTask(id)
	if err != nil {
		if err == ErrTaskNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load task"})
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
