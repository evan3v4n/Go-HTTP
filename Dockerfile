# Build stage
FROM golang:1.21-alpine AS builder
WORKDIR /app

# Copy go mod file
COPY go.mod ./

# Download dependencies (none currently, but good practice)
RUN go mod download

# Copy source code
COPY cmd/ ./cmd/
COPY internal/ ./internal/

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o server ./cmd/server

# Runtime stage - minimal image
FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/server .

# Copy frontend static files
COPY frontend/dist ./frontend/dist

# Environment defaults
ENV SERVER_PORT=:8080
ENV STATIC_FILES_PATH=./frontend/dist
ENV LOG_FILE_PATH=/app/logs.txt
ENV NUM_WORKERS=4

EXPOSE 8080

CMD ["./server"]

