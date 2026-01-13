package app

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"time"
)

// startDispatcher spawns the configured number of worker goroutines.
// Each worker listens on httpRequestsChannel for tasks to process.
func (s *Server) startDispatcher() {
	s.logger.Info("starting worker pool", "workers", s.config.NumWorkers)

	for i := 1; i <= s.config.NumWorkers; i++ {
		s.workerWg.Add(1)
		go s.runWorker(i)
	}

	s.logger.Info("all workers started", "workers", s.config.NumWorkers)
}

// runWorker is the main loop for a worker goroutine.
// It processes tasks from httpRequestsChannel until shutdown.
func (s *Server) runWorker(id int) {
	defer s.workerWg.Done()
	s.logger.Info("worker started", "worker_id", id)

	for {
		select {
		case task, ok := <-s.httpRequestsChannel:
			if !ok {
				s.logger.Info("worker stopping (channel closed)", "worker_id", id)
				return
			}
			s.processTaskWithTracking(task, id)

		case <-s.shutdown:
			s.logger.Info("worker received shutdown signal", "worker_id", id)
			// Drain remaining tasks from channel
			for task := range s.httpRequestsChannel {
				s.processTaskWithTracking(task, id)
			}
			s.logger.Info("worker finished draining", "worker_id", id)
			return
		}
	}
}

// processTaskWithTracking wraps task processing with active worker tracking
// and SSE notification broadcasting.
func (s *Server) processTaskWithTracking(task RequestTask, workerID int) {
	atomic.AddInt32(&s.activeWorkers, 1)
	result := s.processTask(task, workerID)
	atomic.AddInt32(&s.activeWorkers, -1)

	// Send result back to the HTTP handler
	task.ResultChan <- result

	// Broadcast completion via SSE
	duration := time.Since(task.StartTime)
	notification := JobNotification{
		RequestID: task.ID,
		Type:      string(task.Type),
		Success:   result.Success,
		Data:      result.Data,
		Timestamp: time.Now(),
		Duration:  duration.String(),
	}
	s.broadcastSSE(notification)
}

// processTask handles the business logic for each task type.
// It logs the processing and delegates to type-specific handlers.
func (s *Server) processTask(task RequestTask, workerID int) TaskResult {
	// Log that we're processing this task
	s.logChannel <- LogEntry{
		Timestamp: time.Now(),
		Action:    LogActionProcessing,
		RequestID: task.ID,
		Message:   fmt.Sprintf("Worker %d processing %s task", workerID, task.Type),
	}

	switch task.Type {
	case TaskTypeAdd:
		return s.processAddTask(task)
	case TaskTypeClear:
		return s.processClearTask(task)
	case TaskTypeStats:
		return s.processStatsTask(task)
	default:
		return TaskResult{
			Success: false,
			Error:   fmt.Sprintf("unknown task type: %s", task.Type),
			Status:  http.StatusBadRequest,
		}
	}
}

// processAddTask handles the ADD operation.
// It extracts the number from the payload, adds it to mainNumsArr,
// and logs the action.
func (s *Server) processAddTask(task RequestTask) TaskResult {
	numVal, ok := task.Payload["number"].(float64)
	if !ok {
		return TaskResult{
			Success: false,
			Error:   "invalid number in payload",
			Status:  http.StatusBadRequest,
		}
	}
	num := int(numVal)

	// Add to shared array with write lock
	s.dataMu.Lock()
	s.mainNumsArr = append(s.mainNumsArr, num)
	currentLen := len(s.mainNumsArr)
	s.dataMu.Unlock()

	// Log the ADD action
	s.logChannel <- LogEntry{
		Timestamp: time.Now(),
		Action:    LogActionAdd,
		Value:     num,
	}

	return TaskResult{
		Success: true,
		Data: AddResponse{
			Success:   true,
			Message:   fmt.Sprintf("Added %d to array", num),
			Count:     currentLen,
			RequestID: task.ID,
		},
		Status: http.StatusOK,
	}
}

// processClearTask handles the CLEAR operation.
// It resets mainNumsArr to an empty slice and logs the action.
func (s *Server) processClearTask(task RequestTask) TaskResult {
	s.dataMu.Lock()
	s.mainNumsArr = make([]int, 0)
	s.dataMu.Unlock()

	// Log the CLEAR action
	s.logChannel <- LogEntry{
		Timestamp: time.Now(),
		Action:    LogActionClear,
		Value:     0,
	}

	return TaskResult{
		Success: true,
		Data: ClearResponse{
			Success:   true,
			Message:   "Array cleared",
			RequestID: task.ID,
		},
		Status: http.StatusOK,
	}
}

// processStatsTask handles the STATS operation.
// It returns a copy of mainNumsArr and the processed logs count.
func (s *Server) processStatsTask(task RequestTask) TaskResult {
	// Read with read lock
	s.dataMu.RLock()
	numbers := make([]int, len(s.mainNumsArr))
	copy(numbers, s.mainNumsArr)
	s.dataMu.RUnlock()

	return TaskResult{
		Success: true,
		Data: StatsResponse{
			Numbers:       numbers,
			ProcessedLogs: s.getProcessedLogsCount(),
		},
		Status: http.StatusOK,
	}
}

// getActiveWorkerCount returns the current number of workers processing tasks.
func (s *Server) getActiveWorkerCount() int {
	return int(atomic.LoadInt32(&s.activeWorkers))
}

// getQueueDepth returns the current number of tasks waiting in the queue.
func (s *Server) getQueueDepth() int {
	return len(s.httpRequestsChannel)
}

