package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"cortex/scanner"
	"github.com/redis/go-redis/v9"
)

// TaskStore defines persistence operations for scan tasks.
type TaskStore interface {
	CreateTask(task *ScanTask) error
	GetTask(id string) (*ScanTask, error)
	UpdateTask(task *ScanTask) error
	PushToQueue(taskID string) error
	PopFromQueue() (string, error)
}

var (
	// ErrTaskNotFound indicates the requested task doesn't exist in the store.
	ErrTaskNotFound = errors.New("task not found")
)

// RedisStore implements TaskStore using Redis as backend.
type RedisStore struct {
	client *redis.Client
}

// NewRedisStore constructs a Redis-backed task store.
func NewRedisStore(client *redis.Client) *RedisStore {
	return &RedisStore{client: client}
}

func (s *RedisStore) taskKey(id string) string {
	return fmt.Sprintf("scan:%s", id)
}

// CreateTask persists a new scan task in Redis.
func (s *RedisStore) CreateTask(task *ScanTask) error {
	data, err := serializeTask(task)
	if err != nil {
		return err
	}
	return s.client.HSet(context.Background(), s.taskKey(task.ID), data).Err()
}

// GetTask retrieves a task by ID.
func (s *RedisStore) GetTask(id string) (*ScanTask, error) {
	res, err := s.client.HGetAll(context.Background(), s.taskKey(id)).Result()
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, ErrTaskNotFound
	}
	return deserializeTask(res)
}

// UpdateTask updates an existing task in Redis.
func (s *RedisStore) UpdateTask(task *ScanTask) error {
	data, err := serializeTask(task)
	if err != nil {
		return err
	}
	return s.client.HSet(context.Background(), s.taskKey(task.ID), data).Err()
}

// PushToQueue enqueues a task ID for workers to process.
func (s *RedisStore) PushToQueue(taskID string) error {
	return s.client.LPush(context.Background(), "scans:queue", taskID).Err()
}

// PopFromQueue blocks until a task ID is available.
func (s *RedisStore) PopFromQueue() (string, error) {
	res, err := s.client.BRPop(context.Background(), 0, "scans:queue").Result()
	if err != nil {
		return "", err
	}
	if len(res) != 2 {
		return "", errors.New("unexpected response size from BRPOP")
	}
	return res[1], nil
}

func serializeTask(task *ScanTask) (map[string]interface{}, error) {
	hosts, err := json.Marshal(task.Hosts)
	if err != nil {
		return nil, err
	}

	var resultsData string
	if task.Results != nil {
		encoded, err := json.Marshal(task.Results)
		if err != nil {
			return nil, err
		}
		resultsData = string(encoded)
	}

	createdAt := task.CreatedAt.Format(time.RFC3339Nano)
	completedAt := ""
	if task.CompletedAt != nil {
		completedAt = task.CompletedAt.Format(time.RFC3339Nano)
	}

	return map[string]interface{}{
		"id":           task.ID,
		"status":       task.Status,
		"hosts":        string(hosts),
		"ports":        task.Ports,
		"mode":         task.Mode,
		"results":      resultsData,
		"created_at":   createdAt,
		"completed_at": completedAt,
		"error":        task.Error,
	}, nil
}

func deserializeTask(data map[string]string) (*ScanTask, error) {
	var hosts []string
	if raw, ok := data["hosts"]; ok && raw != "" {
		if err := json.Unmarshal([]byte(raw), &hosts); err != nil {
			return nil, err
		}
	}

	var results []scanner.ScanResult
	if raw, ok := data["results"]; ok && raw != "" {
		if err := json.Unmarshal([]byte(raw), &results); err != nil {
			return nil, err
		}
	}

	createdAt := time.Time{}
	if raw, ok := data["created_at"]; ok && raw != "" {
		t, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			return nil, err
		}
		createdAt = t
	}

	var completedAt *time.Time
	if raw, ok := data["completed_at"]; ok && raw != "" {
		t, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			return nil, err
		}
		completedAt = &t
	}

	task := &ScanTask{
		ID:          data["id"],
		Status:      data["status"],
		Hosts:       hosts,
		Ports:       data["ports"],
		Mode:        data["mode"],
		Results:     results,
		CreatedAt:   createdAt,
		CompletedAt: completedAt,
		Error:       data["error"],
	}

	return task, nil
}
