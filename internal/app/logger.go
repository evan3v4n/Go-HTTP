package app

import (
	"fmt"
	"sync/atomic"
	"time"
)

// runLogWriter is the background goroutine that processes log entries.
// It reads from logChannel and writes formatted entries to the log file.
// This goroutine runs until logChannel is closed.
func (s *Server) runLogWriter() {
	defer s.logWriterWg.Done()
	s.logger.Info("log writer started")

	for entry := range s.logChannel {
		s.writeLogEntry(entry)
	}

	s.logger.Info("log writer stopped")
}

// writeLogEntry formats and writes a single log entry to the log file.
// It increments the processedLogs counter after successful write.
func (s *Server) writeLogEntry(entry LogEntry) {
	logLine := s.formatLogEntry(entry)

	if _, err := s.logFile.WriteString(logLine); err != nil {
		s.logger.Error("failed to write log entry", "error", err)
		return
	}

	// Sync to ensure data is written to disk
	if err := s.logFile.Sync(); err != nil {
		s.logger.Error("failed to sync log file", "error", err)
	}

	atomic.AddInt64(&s.processedLogs, 1)
}

// formatLogEntry formats a LogEntry into a human-readable string.
func (s *Server) formatLogEntry(entry LogEntry) string {
	timestamp := entry.Timestamp.Format(time.RFC3339)

	switch {
	case entry.RequestID != "" && entry.Duration != "":
		// Request finished log
		return fmt.Sprintf("[%s] [%s] %s: %s (status=%d, duration=%s)\n",
			timestamp,
			entry.RequestID,
			entry.Action,
			entry.Message,
			entry.Status,
			entry.Duration,
		)

	case entry.RequestID != "":
		// Request started or processing log
		return fmt.Sprintf("[%s] [%s] %s: %s\n",
			timestamp,
			entry.RequestID,
			entry.Action,
			entry.Message,
		)

	default:
		// Value-based log (ADD, CLEAR)
		return fmt.Sprintf("[%s] %s: value=%d\n",
			timestamp,
			entry.Action,
			entry.Value,
		)
	}
}

// getProcessedLogsCount returns the current count of processed log entries.
func (s *Server) getProcessedLogsCount() int64 {
	return atomic.LoadInt64(&s.processedLogs)
}

