package app

import (
	"time"
)

// =============================================================================
// Request/Response Models
// =============================================================================

// AddRequest represents the JSON payload for POST /add endpoint.
type AddRequest struct {
	Number int `json:"number"`
}

// Validate checks if the AddRequest contains valid data.
func (r *AddRequest) Validate() error {
	// Add any validation rules here
	// Currently accepting any integer
	return nil
}

// AddResponse represents the JSON response for POST /add endpoint.
type AddResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	Count     int    `json:"count"`
	RequestID string `json:"request_id"`
}

// StatsResponse represents the JSON response for GET /stats endpoint.
type StatsResponse struct {
	Numbers       []int `json:"numbers"`
	ProcessedLogs int64 `json:"processed_logs"`
}

// ClearResponse represents the JSON response for POST /clear endpoint.
type ClearResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

// QueueStatusResponse represents the queue depth and worker status info.
type QueueStatusResponse struct {
	QueueDepth    int   `json:"queue_depth"`
	QueueCapacity int   `json:"queue_capacity"`
	ActiveWorkers int   `json:"active_workers"`
	TotalWorkers  int   `json:"total_workers"`
	ProcessedLogs int64 `json:"processed_logs"`
}

// HealthResponse represents the response for the health check endpoint.
type HealthResponse struct {
	Status        string `json:"status"`
	Timestamp     string `json:"timestamp"`
	Version       string `json:"version"`
	Uptime        string `json:"uptime"`
	WorkersActive int    `json:"workers_active"`
	WorkersTotal  int    `json:"workers_total"`
	QueueDepth    int    `json:"queue_depth"`
}

// ErrorResponse represents a standardized error response.
type ErrorResponse struct {
	Error     string `json:"error"`
	Code      string `json:"code,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

// =============================================================================
// Internal Task Models
// =============================================================================

// TaskType represents the type of task to be processed.
type TaskType string

const (
	TaskTypeAdd   TaskType = "add"
	TaskTypeClear TaskType = "clear"
	TaskTypeStats TaskType = "stats"
)

// RequestTask represents a task to be processed by a worker goroutine.
// Tasks are queued in httpRequestsChannel and processed asynchronously.
type RequestTask struct {
	// ID is a unique identifier for tracking this request
	ID string

	// Type indicates what operation to perform
	Type TaskType

	// Payload contains request-specific data
	Payload map[string]interface{}

	// ResultChan is used to send the result back to the HTTP handler
	ResultChan chan TaskResult

	// StartTime records when the task was created for duration tracking
	StartTime time.Time
}

// TaskResult represents the result of a processed task.
type TaskResult struct {
	Success bool
	Data    interface{}
	Error   string
	Status  int
}

// =============================================================================
// Logging Models
// =============================================================================

// LogAction represents the type of action being logged.
type LogAction string

const (
	LogActionRequestStarted  LogAction = "REQUEST_STARTED"
	LogActionRequestFinished LogAction = "REQUEST_FINISHED"
	LogActionProcessing      LogAction = "PROCESSING"
	LogActionAdd             LogAction = "ADD"
	LogActionClear           LogAction = "CLEAR"
)

// LogEntry represents a single log entry to be written to the log file.
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Action    LogAction `json:"action"`
	RequestID string    `json:"request_id,omitempty"`
	Value     int       `json:"value,omitempty"`
	Duration  string    `json:"duration,omitempty"`
	Status    int       `json:"status,omitempty"`
	Message   string    `json:"message,omitempty"`
}

// =============================================================================
// SSE Models
// =============================================================================

// JobNotification is sent via Server-Sent Events when a job completes.
type JobNotification struct {
	RequestID string      `json:"request_id"`
	Type      string      `json:"type"`
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
	Duration  string      `json:"duration"`
}

// SSEConnectedMessage is sent when a client first connects to the SSE endpoint.
type SSEConnectedMessage struct {
	Type    string `json:"type"`
	Workers int    `json:"workers"`
}

