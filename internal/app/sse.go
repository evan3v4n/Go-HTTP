package app

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// HandleSSE handles GET /events requests for Server-Sent Events.
// Clients can subscribe to real-time job completion notifications.
func (s *Server) HandleSSE(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", s.config.CORSAllowOrigin)

	// Check if streaming is supported
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	// Create client channel and register
	clientChan := make(chan JobNotification, 10)
	s.addSSEClient(clientChan)
	defer s.removeSSEClient(clientChan)

	// Send initial connection message
	connMsg := SSEConnectedMessage{
		Type:    "connected",
		Workers: s.config.NumWorkers,
	}
	if data, err := json.Marshal(connMsg); err == nil {
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}

	// Stream events until client disconnects or server shuts down
	for {
		select {
		case notification := <-clientChan:
			data, err := json.Marshal(notification)
			if err != nil {
				s.logger.Error("failed to marshal SSE notification", "error", err)
				continue
			}
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()

		case <-r.Context().Done():
			s.logger.Debug("SSE client disconnected")
			return

		case <-s.shutdown:
			s.logger.Debug("SSE shutting down")
			return
		}
	}
}

// addSSEClient registers a new SSE client channel.
func (s *Server) addSSEClient(ch chan JobNotification) {
	s.sseClientsMu.Lock()
	s.sseClients[ch] = true
	s.sseClientsMu.Unlock()
	s.logger.Debug("SSE client connected", "total_clients", len(s.sseClients))
}

// removeSSEClient unregisters an SSE client channel and closes it.
func (s *Server) removeSSEClient(ch chan JobNotification) {
	s.sseClientsMu.Lock()
	delete(s.sseClients, ch)
	s.sseClientsMu.Unlock()
	close(ch)
	s.logger.Debug("SSE client disconnected", "total_clients", len(s.sseClients))
}

// broadcastSSE sends a notification to all connected SSE clients.
// If a client's channel is full (slow consumer), the message is dropped.
func (s *Server) broadcastSSE(notification JobNotification) {
	s.sseClientsMu.RLock()
	defer s.sseClientsMu.RUnlock()

	for clientChan := range s.sseClients {
		select {
		case clientChan <- notification:
			// Message sent successfully
		default:
			// Client too slow, drop the message
			s.logger.Debug("SSE message dropped (slow client)")
		}
	}
}

