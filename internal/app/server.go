package app

import (
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"
)

// Server is the main application server that coordinates all components.
// It manages the worker pool, logging system, SSE clients, and shared state.
type Server struct {
	// config holds all configuration values
	config *Config

	// logger is the structured logger for server operations
	logger *slog.Logger

	// startTime records when the server started for uptime calculation
	startTime time.Time

	// Shared data (protected by mutex)
	mainNumsArr []int
	dataMu      sync.RWMutex

	// Request channel for the worker pool
	httpRequestsChannel chan RequestTask

	// Log channel and file for async logging
	logChannel    chan LogEntry
	processedLogs int64
	logFile       *os.File

	// SSE client management
	sseClients   map[chan JobNotification]bool
	sseClientsMu sync.RWMutex

	// Worker tracking
	activeWorkers int32

	// Synchronization primitives
	workerWg    sync.WaitGroup
	logWriterWg sync.WaitGroup
	shutdown    chan struct{}
}

// NewServer creates and initializes a new Server instance with the given configuration.
// It opens the log file and initializes all channels and maps.
// Returns an error if the log file cannot be opened.
func NewServer(cfg *Config) (*Server, error) {
	// Open log file
	logFile, err := os.OpenFile(cfg.LogFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file %s: %w", cfg.LogFilePath, err)
	}

	// Create structured logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	return &Server{
		config:              cfg,
		logger:              logger,
		startTime:           time.Now(),
		mainNumsArr:         make([]int, 0),
		httpRequestsChannel: make(chan RequestTask, cfg.RequestBufferSize),
		logChannel:          make(chan LogEntry, cfg.LogBufferSize),
		logFile:             logFile,
		sseClients:          make(map[chan JobNotification]bool),
		shutdown:            make(chan struct{}),
	}, nil
}

// Start initializes and starts all server components:
// - Log writer goroutine
// - Worker pool (dispatcher + workers)
func (s *Server) Start() {
	// Start the log writer goroutine
	s.logWriterWg.Add(1)
	go s.runLogWriter()

	// Start the worker pool
	s.startDispatcher()
}

// Shutdown performs a graceful shutdown of all server components.
// It ensures all pending tasks are processed and logs are written before returning.
func (s *Server) Shutdown() {
	s.logger.Info("initiating graceful shutdown")

	// Signal all goroutines to stop accepting new work
	close(s.shutdown)

	// Close the request channel to signal workers to drain and exit
	s.logger.Info("closing request channel")
	close(s.httpRequestsChannel)

	// Wait for all workers to finish processing remaining tasks
	s.logger.Info("waiting for workers to finish")
	s.workerWg.Wait()
	s.logger.Info("all workers finished")

	// Close the log channel (workers are done sending logs)
	s.logger.Info("closing log channel")
	close(s.logChannel)

	// Wait for log writer to finish
	s.logger.Info("waiting for log writer to finish")
	s.logWriterWg.Wait()
	s.logger.Info("log writer finished")

	// Close the log file
	if err := s.logFile.Close(); err != nil {
		s.logger.Error("error closing log file", "error", err)
	}

	// Close all SSE clients
	s.sseClientsMu.Lock()
	for ch := range s.sseClients {
		close(ch)
		delete(s.sseClients, ch)
	}
	s.sseClientsMu.Unlock()

	s.logger.Info("graceful shutdown complete")
}

// GetUptime returns the duration since the server started.
func (s *Server) GetUptime() time.Duration {
	return time.Since(s.startTime)
}

// contextKey is a custom type for context keys to avoid collisions.
type contextKey string

const (
	contextKeyRequestID contextKey = "requestID"
	contextKeyStartTime contextKey = "startTime"
)

