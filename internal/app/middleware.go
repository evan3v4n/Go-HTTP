package app

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// responseWriter wraps http.ResponseWriter to capture the status code.
// It also implements http.Flusher to support SSE streaming.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

// WriteHeader captures the status code before writing it.
func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.written = true
	}
	rw.ResponseWriter.WriteHeader(code)
}

// Write ensures WriteHeader is called before writing the body.
func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}

// Flush implements http.Flusher for SSE support.
func (rw *responseWriter) Flush() {
	if flusher, ok := rw.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// LoggingMiddleware logs the entry and exit of each HTTP request.
// It generates a unique request ID and records timing information.
func (s *Server) LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip logging for static files, SSE, and high-frequency monitoring endpoints
		if isStaticFile(r.URL.Path) || r.URL.Path == "/events" ||
			r.URL.Path == "/stats" || r.URL.Path == "/queue-status" || r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}

		requestID := generateRequestID()
		startTime := time.Now()

		// Log request entry
		s.logChannel <- LogEntry{
			Timestamp: startTime,
			Action:    LogActionRequestStarted,
			RequestID: requestID,
			Message:   fmt.Sprintf("%s %s", r.Method, r.URL.Path),
		}

		// Wrap response writer to capture status code
		rw := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Add request metadata to context
		ctx := context.WithValue(r.Context(), contextKeyRequestID, requestID)
		ctx = context.WithValue(ctx, contextKeyStartTime, startTime)

		// Process the request
		next.ServeHTTP(rw, r.WithContext(ctx))

		// Log request exit
		duration := time.Since(startTime)
		s.logChannel <- LogEntry{
			Timestamp: time.Now(),
			Action:    LogActionRequestFinished,
			RequestID: requestID,
			Message:   fmt.Sprintf("%s %s", r.Method, r.URL.Path),
			Status:    rw.statusCode,
			Duration:  duration.String(),
		}
	})
}

// CORSMiddleware adds CORS headers to allow cross-origin requests.
// In production, CORSAllowOrigin should be set to specific domains.
func (s *Server) CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", s.config.CORSAllowOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RecoveryMiddleware catches panics and returns a 500 error instead of crashing.
func (s *Server) RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				s.logger.Error("panic recovered", "error", err, "path", r.URL.Path)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// isStaticFile checks if the path is for a static file.
func isStaticFile(path string) bool {
	staticExtensions := []string{".js", ".css", ".html", ".svg", ".png", ".jpg", ".ico", ".woff", ".woff2"}
	for _, ext := range staticExtensions {
		if len(path) > len(ext) && path[len(path)-len(ext):] == ext {
			return true
		}
	}
	return path == "/" || path == "/index.html"
}

// generateRequestID creates a unique request ID based on timestamp.
func generateRequestID() string {
	return fmt.Sprintf("req-%d", time.Now().UnixNano())
}

