# Go HTTP Server - Worker Pool Architecture

A production-ready Go HTTP server demonstrating the **Worker Pool + Async Logging** architecture pattern. Built as a reference implementation for concurrent request processing with real-time monitoring.

![Architecture Diagram](docs/architecture.png)

## Features

- **Worker Pool Pattern**: K configurable worker goroutines process requests from a shared channel
- **Async Logging**: Dedicated log writer goroutine for non-blocking file I/O
- **Request Queuing**: Buffered channel distributes load across workers
- **Real-time Monitoring**: SSE (Server-Sent Events) for live job notifications
- **Health Checks**: Production-ready health endpoint for load balancers
- **Graceful Shutdown**: Proper draining of queues and worker termination
- **React Dashboard**: Modern UI for visualizing worker pool status

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         HTTP Server                              │
│                    (Public Entry Point)                          │
└──────────────────────────┬──────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│                   httpRequestsChannel                            │
│                   (Buffered Channel)                             │
└──────────────────────────┬──────────────────────────────────────┘
                           │
        ┌──────────────────┼──────────────────┐
        ▼                  ▼                  ▼
┌───────────────┐  ┌───────────────┐  ┌───────────────┐
│   Worker 1    │  │   Worker 2    │  │   Worker K    │
│  (goroutine)  │  │  (goroutine)  │  │  (goroutine)  │
└───────┬───────┘  └───────┬───────┘  └───────┬───────┘
        │                  │                  │
        └──────────────────┼──────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│                      logChannel                                  │
│                   (Buffered Channel)                             │
└──────────────────────────┬──────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Log Writer                                  │
│                     (goroutine)                                  │
└──────────────────────────┬──────────────────────────────────────┘
                           │
                           ▼
                      [ logs.txt ]
```

### Request Flow

1. **Entry**: HTTP request hits the server
2. **Logging (Entry)**: Middleware logs request start
3. **Queue**: Handler wraps request into `RequestTask`, pushes to `httpRequestsChannel`
4. **Process**: Available worker picks up task, processes business logic
5. **Logging (Exit)**: Middleware logs request completion with duration
6. **Response**: Result sent back to client
7. **Notify**: SSE broadcast notifies connected clients

## Project Structure

```
.
├── cmd/
│   └── server/
│       └── main.go           # Entry point, server setup
├── internal/
│   └── app/
│       ├── config.go         # Configuration with env var support
│       ├── server.go         # Server struct, initialization, shutdown
│       ├── models.go         # All data structures/types
│       ├── handlers.go       # HTTP handlers (/add, /stats, /clear, etc.)
│       ├── workers.go        # Worker pool dispatcher and task processing
│       ├── middleware.go     # Logging, CORS, recovery middleware
│       ├── logger.go         # Async log writer
│       ├── sse.go            # Server-Sent Events handling
│       └── server_test.go    # Unit tests
├── frontend/                 # React + TypeScript dashboard
├── go.mod
└── README.md
```

## Quick Start

### Prerequisites

- Go 1.21+
- Node.js 18+ (for frontend)

### Build & Run

```bash
# Build the server
go build -o server ./cmd/server

# Run with defaults
./server

# Or run with custom configuration
NUM_WORKERS=8 REQUEST_BUFFER_SIZE=500 ./server
```

### Build Frontend

```bash
cd frontend
npm install
npm run build
cd ..
```

### Access

- **Dashboard**: http://localhost:8080
- **Health Check**: http://localhost:8080/health
- **Queue Status**: http://localhost:8080/queue-status

## Configuration

All configuration is via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_PORT` | `:8080` | HTTP server port |
| `NUM_WORKERS` | `4` | Number of worker goroutines |
| `REQUEST_BUFFER_SIZE` | `100` | Request queue capacity |
| `LOG_BUFFER_SIZE` | `100` | Log queue capacity |
| `LOG_FILE_PATH` | `logs.txt` | Path to log file |
| `STATIC_FILES_PATH` | `./frontend/dist` | Frontend build directory |
| `CORS_ALLOW_ORIGIN` | `*` | CORS allowed origins |
| `READ_TIMEOUT_SECONDS` | `30` | HTTP read timeout |
| `WRITE_TIMEOUT_SECONDS` | `30` | HTTP write timeout |

## API Endpoints

### `POST /add`

Add a number to the array.

```bash
curl -X POST http://localhost:8080/add \
  -H "Content-Type: application/json" \
  -d '{"number": 42}'
```

Response:
```json
{
  "success": true,
  "message": "Added 42 to array",
  "count": 1,
  "request_id": "req-1234567890"
}
```

### `GET /stats`

Get current array contents and stats.

```bash
curl http://localhost:8080/stats
```

Response:
```json
{
  "numbers": [42, 17, 99],
  "processed_logs": 156
}
```

### `POST /clear`

Clear the array.

```bash
curl -X POST http://localhost:8080/clear
```

### `GET /queue-status`

Get worker pool and queue status.

```bash
curl http://localhost:8080/queue-status
```

Response:
```json
{
  "queue_depth": 3,
  "queue_capacity": 100,
  "active_workers": 2,
  "total_workers": 4,
  "processed_logs": 156
}
```

### `GET /health`

Health check for load balancers/orchestrators.

```bash
curl http://localhost:8080/health
```

Response:
```json
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00Z",
  "version": "1.0.0",
  "uptime": "2h15m30s",
  "workers_active": 1,
  "workers_total": 4,
  "queue_depth": 0
}
```

### `GET /events`

Server-Sent Events stream for real-time job notifications.

```bash
curl -N http://localhost:8080/events
```

## Testing

```bash
# Run all tests
go test -v ./...

# Run tests with coverage
go test -cover ./...

# Run benchmarks
go test -bench=. -benchmem ./...
```

## Graceful Shutdown

The server handles `SIGINT` and `SIGTERM` signals gracefully:

1. Stop accepting new HTTP connections
2. Close `httpRequestsChannel` (no new tasks)
3. Wait for workers to drain remaining tasks
4. Close `logChannel` (no new log entries)
5. Wait for log writer to finish
6. Close log file
7. Exit

This ensures no data loss during deployments or restarts.

## Production Deployment

### Docker

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o server ./cmd/server

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/server .
COPY --from=builder /app/frontend/dist ./frontend/dist
EXPOSE 8080
CMD ["./server"]
```

### Kubernetes Health Probes

```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 10

readinessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
```

### Recommended Production Settings

```bash
NUM_WORKERS=16              # Based on CPU cores
REQUEST_BUFFER_SIZE=1000    # Handle traffic bursts
LOG_BUFFER_SIZE=500         # Prevent log backpressure
CORS_ALLOW_ORIGIN=https://yourdomain.com
```

## License

MIT License - See LICENSE file for details.
