package app

import (
	"encoding/json"
	"net/http"
	"time"
)

// HandleAdd handles POST /add requests.
// It validates the request, creates a task, and queues it for processing.
func (s *Server) HandleAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req AddRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	if err := req.Validate(); err != nil {
		s.writeError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Create and queue the task
	task := RequestTask{
		ID:   generateRequestID(),
		Type: TaskTypeAdd,
		Payload: map[string]interface{}{
			"number": float64(req.Number),
		},
		ResultChan: make(chan TaskResult, 1),
		StartTime:  time.Now(),
	}

	s.submitTask(w, task)
}

// HandleStats handles GET /stats requests.
// It returns the current array contents and processed logs count.
func (s *Server) HandleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	task := RequestTask{
		ID:         generateRequestID(),
		Type:       TaskTypeStats,
		Payload:    nil,
		ResultChan: make(chan TaskResult, 1),
		StartTime:  time.Now(),
	}

	s.submitTask(w, task)
}

// HandleClear handles POST /clear requests.
// It queues a task to clear the mainNumsArr.
func (s *Server) HandleClear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	task := RequestTask{
		ID:         generateRequestID(),
		Type:       TaskTypeClear,
		Payload:    nil,
		ResultChan: make(chan TaskResult, 1),
		StartTime:  time.Now(),
	}

	s.submitTask(w, task)
}

// HandleQueueStatus handles GET /queue-status requests.
// It returns current queue depth and worker status without going through the queue.
func (s *Server) HandleQueueStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := QueueStatusResponse{
		QueueDepth:    s.getQueueDepth(),
		QueueCapacity: s.config.RequestBufferSize,
		ActiveWorkers: s.getActiveWorkerCount(),
		TotalWorkers:  s.config.NumWorkers,
		ProcessedLogs: s.getProcessedLogsCount(),
	}

	s.writeJSON(w, http.StatusOK, response)
}

// HandleHealth handles GET /health requests.
// It returns server health status for monitoring and load balancers.
func (s *Server) HandleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := HealthResponse{
		Status:        "healthy",
		Timestamp:     time.Now().Format(time.RFC3339),
		Version:       "1.0.0",
		Uptime:        s.GetUptime().String(),
		WorkersActive: s.getActiveWorkerCount(),
		WorkersTotal:  s.config.NumWorkers,
		QueueDepth:    s.getQueueDepth(),
	}

	s.writeJSON(w, http.StatusOK, response)
}

// submitTask attempts to submit a task to the worker pool.
// If the queue is full, it returns a 503 Service Unavailable error.
func (s *Server) submitTask(w http.ResponseWriter, task RequestTask) {
	select {
	case s.httpRequestsChannel <- task:
		// Wait for result from worker
		result := <-task.ResultChan

		if !result.Success {
			s.writeJSON(w, result.Status, ErrorResponse{
				Error:     result.Error,
				RequestID: task.ID,
			})
			return
		}

		s.writeJSON(w, http.StatusOK, result.Data)

	default:
		// Queue is full
		s.writeJSON(w, http.StatusServiceUnavailable, ErrorResponse{
			Error:     "Server busy, request queue full",
			Code:      "QUEUE_FULL",
			RequestID: task.ID,
		})
	}
}

// writeJSON writes a JSON response with the given status code.
func (s *Server) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		s.logger.Error("failed to encode JSON response", "error", err)
	}
}

// writeError writes a JSON error response.
func (s *Server) writeError(w http.ResponseWriter, message string, status int) {
	s.writeJSON(w, status, ErrorResponse{Error: message})
}

