package app

import (
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration values.
// Values can be set via environment variables or fall back to defaults.
type Config struct {
	// Server settings
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration

	// Worker pool settings
	NumWorkers        int
	RequestBufferSize int

	// Logging settings
	LogBufferSize int
	LogFilePath   string

	// Frontend settings
	StaticFilesPath string

	// CORS settings
	CORSAllowOrigin string
}

// DefaultConfig returns a Config with sensible default values.
func DefaultConfig() *Config {
	return &Config{
		Port:              ":8080",
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
		NumWorkers:        4,
		RequestBufferSize: 100,
		LogBufferSize:     100,
		LogFilePath:       "logs.txt",
		StaticFilesPath:   "./frontend/dist",
		CORSAllowOrigin:   "*",
	}
}

// LoadConfig loads configuration from environment variables,
// falling back to defaults for any unset values.
func LoadConfig() *Config {
	cfg := DefaultConfig()

	if port := os.Getenv("SERVER_PORT"); port != "" {
		cfg.Port = port
	}

	if workers := os.Getenv("NUM_WORKERS"); workers != "" {
		if n, err := strconv.Atoi(workers); err == nil && n > 0 {
			cfg.NumWorkers = n
		}
	}

	if bufSize := os.Getenv("REQUEST_BUFFER_SIZE"); bufSize != "" {
		if n, err := strconv.Atoi(bufSize); err == nil && n > 0 {
			cfg.RequestBufferSize = n
		}
	}

	if bufSize := os.Getenv("LOG_BUFFER_SIZE"); bufSize != "" {
		if n, err := strconv.Atoi(bufSize); err == nil && n > 0 {
			cfg.LogBufferSize = n
		}
	}

	if logPath := os.Getenv("LOG_FILE_PATH"); logPath != "" {
		cfg.LogFilePath = logPath
	}

	if staticPath := os.Getenv("STATIC_FILES_PATH"); staticPath != "" {
		cfg.StaticFilesPath = staticPath
	}

	if corsOrigin := os.Getenv("CORS_ALLOW_ORIGIN"); corsOrigin != "" {
		cfg.CORSAllowOrigin = corsOrigin
	}

	if timeout := os.Getenv("READ_TIMEOUT_SECONDS"); timeout != "" {
		if n, err := strconv.Atoi(timeout); err == nil && n > 0 {
			cfg.ReadTimeout = time.Duration(n) * time.Second
		}
	}

	if timeout := os.Getenv("WRITE_TIMEOUT_SECONDS"); timeout != "" {
		if n, err := strconv.Atoi(timeout); err == nil && n > 0 {
			cfg.WriteTimeout = time.Duration(n) * time.Second
		}
	}

	return cfg
}

