package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// testConfig returns a Config suitable for testing.
func testConfig() *Config {
	return &Config{
		Port:              ":0",
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
		IdleTimeout:       5 * time.Second,
		NumWorkers:        2,
		RequestBufferSize: 10,
		LogBufferSize:     10,
		LogFilePath:       "test_logs.txt",
		StaticFilesPath:   "./frontend/dist",
		CORSAllowOrigin:   "*",
	}
}

// setupTestServer creates a server for testing and starts its background services.
func setupTestServer(t *testing.T) (*Server, func()) {
	t.Helper()

	cfg := testConfig()
	server, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("failed to create test server: %v", err)
	}

	server.Start()

	cleanup := func() {
		server.Shutdown()
		os.Remove(cfg.LogFilePath)
	}

	return server, cleanup
}

// TestNewServer tests server creation.
func TestNewServer(t *testing.T) {
	cfg := testConfig()
	server, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}
	defer os.Remove(cfg.LogFilePath)
	defer server.logFile.Close()

	if server.config != cfg {
		t.Error("server config not set correctly")
	}

	if server.mainNumsArr == nil {
		t.Error("mainNumsArr not initialized")
	}

	if server.httpRequestsChannel == nil {
		t.Error("httpRequestsChannel not initialized")
	}

	if server.logChannel == nil {
		t.Error("logChannel not initialized")
	}
}

// TestHandleAdd tests the /add endpoint.
func TestHandleAdd(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	tests := []struct {
		name       string
		method     string
		body       string
		wantStatus int
	}{
		{
			name:       "valid add request",
			method:     http.MethodPost,
			body:       `{"number": 42}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid method",
			method:     http.MethodGet,
			body:       "",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "invalid JSON",
			method:     http.MethodPost,
			body:       `{invalid}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/add", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			server.HandleAdd(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

// TestHandleStats tests the /stats endpoint.
func TestHandleStats(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	// First add some numbers
	for _, num := range []int{1, 2, 3} {
		req := httptest.NewRequest(http.MethodPost, "/add",
			strings.NewReader(`{"number": `+string(rune('0'+num))+`}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		server.HandleAdd(rec, req)
	}

	// Wait for workers to process
	time.Sleep(100 * time.Millisecond)

	// Now get stats
	req := httptest.NewRequest(http.MethodGet, "/stats", nil)
	rec := httptest.NewRecorder()

	server.HandleStats(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("got status %d, want %d", rec.Code, http.StatusOK)
	}

	var resp StatsResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Numbers) != 3 {
		t.Errorf("got %d numbers, want 3", len(resp.Numbers))
	}
}

// TestHandleClear tests the /clear endpoint.
func TestHandleClear(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	// Add a number first
	addReq := httptest.NewRequest(http.MethodPost, "/add",
		strings.NewReader(`{"number": 42}`))
	addReq.Header.Set("Content-Type", "application/json")
	addRec := httptest.NewRecorder()
	server.HandleAdd(addRec, addReq)

	time.Sleep(50 * time.Millisecond)

	// Clear
	clearReq := httptest.NewRequest(http.MethodPost, "/clear", nil)
	clearRec := httptest.NewRecorder()
	server.HandleClear(clearRec, clearReq)

	if clearRec.Code != http.StatusOK {
		t.Errorf("clear got status %d, want %d", clearRec.Code, http.StatusOK)
	}

	time.Sleep(50 * time.Millisecond)

	// Verify array is empty
	statsReq := httptest.NewRequest(http.MethodGet, "/stats", nil)
	statsRec := httptest.NewRecorder()
	server.HandleStats(statsRec, statsReq)

	var resp StatsResponse
	if err := json.NewDecoder(statsRec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Numbers) != 0 {
		t.Errorf("got %d numbers after clear, want 0", len(resp.Numbers))
	}
}

// TestHandleQueueStatus tests the /queue-status endpoint.
func TestHandleQueueStatus(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/queue-status", nil)
	rec := httptest.NewRecorder()

	server.HandleQueueStatus(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("got status %d, want %d", rec.Code, http.StatusOK)
	}

	var resp QueueStatusResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.TotalWorkers != 2 {
		t.Errorf("got %d total workers, want 2", resp.TotalWorkers)
	}

	if resp.QueueCapacity != 10 {
		t.Errorf("got %d queue capacity, want 10", resp.QueueCapacity)
	}
}

// TestHandleHealth tests the /health endpoint.
func TestHandleHealth(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	server.HandleHealth(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("got status %d, want %d", rec.Code, http.StatusOK)
	}

	var resp HealthResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Status != "healthy" {
		t.Errorf("got status %q, want %q", resp.Status, "healthy")
	}

	if resp.WorkersTotal != 2 {
		t.Errorf("got %d total workers, want 2", resp.WorkersTotal)
	}
}

// TestCORSMiddleware tests that CORS headers are set correctly.
func TestCORSMiddleware(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	handler := server.CORSMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if origin := rec.Header().Get("Access-Control-Allow-Origin"); origin != "*" {
		t.Errorf("got CORS origin %q, want %q", origin, "*")
	}
}

// TestCORSPreflightRequest tests OPTIONS preflight handling.
func TestCORSPreflightRequest(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	handler := server.CORSMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called for OPTIONS")
	}))

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("OPTIONS got status %d, want %d", rec.Code, http.StatusOK)
	}
}

// TestRecoveryMiddleware tests panic recovery.
func TestRecoveryMiddleware(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	handler := server.RecoveryMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	// Should not panic
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("got status %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

// TestLoadConfig tests configuration loading from environment.
func TestLoadConfig(t *testing.T) {
	// Set test environment variables
	os.Setenv("NUM_WORKERS", "8")
	os.Setenv("SERVER_PORT", ":9090")
	defer os.Unsetenv("NUM_WORKERS")
	defer os.Unsetenv("SERVER_PORT")

	cfg := LoadConfig()

	if cfg.NumWorkers != 8 {
		t.Errorf("got %d workers, want 8", cfg.NumWorkers)
	}

	if cfg.Port != ":9090" {
		t.Errorf("got port %q, want %q", cfg.Port, ":9090")
	}
}

// TestDefaultConfig tests default configuration values.
func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.NumWorkers != 4 {
		t.Errorf("default workers: got %d, want 4", cfg.NumWorkers)
	}

	if cfg.RequestBufferSize != 100 {
		t.Errorf("default buffer: got %d, want 100", cfg.RequestBufferSize)
	}

	if cfg.Port != ":8080" {
		t.Errorf("default port: got %q, want %q", cfg.Port, ":8080")
	}
}

// BenchmarkHandleAdd benchmarks the add handler performance.
func BenchmarkHandleAdd(b *testing.B) {
	cfg := testConfig()
	cfg.NumWorkers = 4
	cfg.RequestBufferSize = 1000

	server, err := NewServer(cfg)
	if err != nil {
		b.Fatalf("failed to create server: %v", err)
	}
	server.Start()
	defer func() {
		server.Shutdown()
		os.Remove(cfg.LogFilePath)
	}()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/add",
			strings.NewReader(`{"number": 42}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		server.HandleAdd(rec, req)
	}
}
