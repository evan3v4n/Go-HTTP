// Package main is the entry point for the Go HTTP server.
// It loads configuration, creates the server, and handles graceful shutdown.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-http-server/internal/app"
)

func main() {
	// Load configuration from environment
	cfg := app.LoadConfig()

	// Create logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Create server
	server, err := app.NewServer(cfg)
	if err != nil {
		logger.Error("failed to create server", "error", err)
		os.Exit(1)
	}

	// Start background services (log writer, workers)
	server.Start()

	// Set up HTTP routes
	mux := http.NewServeMux()

	// API endpoints
	mux.HandleFunc("/add", server.HandleAdd)
	mux.HandleFunc("/stats", server.HandleStats)
	mux.HandleFunc("/clear", server.HandleClear)
	mux.HandleFunc("/queue-status", server.HandleQueueStatus)
	mux.HandleFunc("/health", server.HandleHealth)
	mux.HandleFunc("/events", server.HandleSSE)

	// Serve static files for the frontend
	fs := http.FileServer(http.Dir(cfg.StaticFilesPath))
	mux.Handle("/", fs)

	// Apply middleware chain: Recovery -> CORS -> Logging -> Handler
	handler := server.RecoveryMiddleware(
		server.CORSMiddleware(
			server.LoggingMiddleware(mux),
		),
	)

	// Create HTTP server
	httpServer := &http.Server{
		Addr:         cfg.Port,
		Handler:      handler,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	// Channel to listen for server errors
	serverErrors := make(chan error, 1)

	// Start HTTP server
	go func() {
		logger.Info("========================================")
		logger.Info("  Go HTTP Server - Worker Pool Edition")
		logger.Info("========================================")
		logger.Info("configuration",
			"port", cfg.Port,
			"workers", cfg.NumWorkers,
			"request_buffer", cfg.RequestBufferSize,
			"log_buffer", cfg.LogBufferSize,
		)
		logger.Info("server starting", "url", "http://localhost"+cfg.Port)
		serverErrors <- httpServer.ListenAndServe()
	}()

	// Channel to listen for shutdown signals
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Block until signal or error
	select {
	case err := <-serverErrors:
		if err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}

	case sig := <-shutdown:
		logger.Info("shutdown signal received", "signal", sig)

		// Create context with timeout for graceful shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Shutdown HTTP server (stop accepting new connections)
		logger.Info("shutting down HTTP server")
		if err := httpServer.Shutdown(ctx); err != nil {
			logger.Error("HTTP server shutdown error", "error", err)
			httpServer.Close()
		}

		// Shutdown application server (workers, log writer)
		server.Shutdown()
	}

	logger.Info("server stopped")
}

