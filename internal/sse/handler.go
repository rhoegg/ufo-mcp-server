package sse

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/server"
)

// Handler provides Server-Sent Events endpoint for legacy MCP compatibility
type Handler struct {
	mcpServer *server.MCPServer
	sessions  sync.Map // sessionID -> *Session
}

// Session represents an SSE session with bidirectional communication
type Session struct {
	ID           string
	CreatedAt    time.Time
	ResponseChan chan json.RawMessage // Channel to send responses back via SSE
	mu           sync.Mutex
	Connected    bool // Track if SSE connection is active
}

// NewHandler creates a new SSE handler
func NewHandler(mcpServer *server.MCPServer) *Handler {
	h := &Handler{
		mcpServer: mcpServer,
	}
	
	// Start session cleanup goroutine
	go h.cleanupSessions()
	
	return h
}

// ServeHTTP handles SSE connections for MCP
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Ensure the writer supports flushing
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Create or retrieve session for this connection
	sessionID := uuid.New().String()
	
	// Check if client provided a session ID in query params
	if existingID := r.URL.Query().Get("sessionId"); existingID != "" {
		if sessionInterface, ok := h.sessions.Load(existingID); ok {
			// Reuse existing session
			sessionID = existingID
			session := sessionInterface.(*Session)
			session.mu.Lock()
			session.Connected = true
			session.mu.Unlock()
			log.Printf("Reusing existing session: %s", sessionID)
		} else {
			log.Printf("Session %s not found, creating new one", existingID)
		}
	}
	
	// Create new session if needed
	var session *Session
	if sessionInterface, ok := h.sessions.Load(sessionID); ok {
		session = sessionInterface.(*Session)
	} else {
		session = &Session{
			ID:           sessionID,
			CreatedAt:    time.Now(),
			ResponseChan: make(chan json.RawMessage, 10), // Buffered channel
			Connected:    true,
		}
		h.sessions.Store(sessionID, session)
	}
	
	// Mark disconnected when connection closes
	defer func() {
		session.mu.Lock()
		session.Connected = false
		session.mu.Unlock()
		log.Printf("SSE connection closed for session %s (session preserved)", sessionID)
	}()

	// Send endpoint event with session ID
	endpointPath := fmt.Sprintf("/sse/messages?session_id=%s", sessionID)
	
	fmt.Fprintf(w, "event: endpoint\n")
	fmt.Fprintf(w, "data: %s\n\n", endpointPath)
	flusher.Flush()

	log.Printf("SSE session created: %s", sessionID)

	// Keep connection alive and send responses
	ctx := r.Context()
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("SSE connection context done for session %s", sessionID)
			return
		case <-ticker.C:
			// Send keepalive comment only (not an event)
			fmt.Fprintf(w, ":\n\n")
			flusher.Flush()
		case response := <-session.ResponseChan:
			// Send response as SSE message event
			fmt.Fprintf(w, "event: message\n")
			fmt.Fprintf(w, "data: %s\n\n", response)
			flusher.Flush()
			log.Printf("SSE sent message to session %s: %s", sessionID, string(response))
		}
	}
}

// HandleMessages handles POST requests to send messages to the MCP server
func (h *Handler) HandleMessages(w http.ResponseWriter, r *http.Request) {
	// Handle CORS preflight
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get session ID
	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		http.Error(w, "Missing session_id", http.StatusBadRequest)
		return
	}

	// Get session
	sessionInterface, ok := h.sessions.Load(sessionID)
	if !ok {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}
	session := sessionInterface.(*Session)

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request", http.StatusBadRequest)
		return
	}

	// Parse JSON-RPC request to check if it's an initialize
	var jsonRPCReq struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      interface{}     `json:"id"`
		Method  string          `json:"method"`
		Params  json.RawMessage `json:"params"`
	}

	if err := json.Unmarshal(body, &jsonRPCReq); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Log the incoming request
	log.Printf("SSE message from session %s: %s", sessionID, string(body))
	
	// Process the request in a goroutine to avoid blocking
	go func() {
		// Use HandleMessage to process the request
		ctx := context.Background()
		responseMsg := h.mcpServer.HandleMessage(ctx, body)

		// Marshal the response to JSON
		responseJSON, err := json.Marshal(responseMsg)
		if err != nil {
			log.Printf("ERROR: Failed to marshal response for session %s: %v", sessionID, err)
			return
		}

		// Send response via SSE channel
		session.mu.Lock()
		defer session.mu.Unlock()
		
		select {
		case session.ResponseChan <- responseJSON:
			log.Printf("Queued SSE response for session %s", sessionID)
		default:
			log.Printf("WARNING: Response channel full for session %s", sessionID)
		}
	}()

	// For MuleSoft compatibility, return 202 Accepted immediately
	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("{}"))
}

// cleanupSessions periodically removes inactive sessions
func (h *Handler) cleanupSessions() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		now := time.Now()
		var toDelete []string
		
		h.sessions.Range(func(key, value interface{}) bool {
			sessionID := key.(string)
			session := value.(*Session)
			
			session.mu.Lock()
			// Remove sessions that have been disconnected for more than 30 minutes
			if !session.Connected && now.Sub(session.CreatedAt) > 30*time.Minute {
				toDelete = append(toDelete, sessionID)
			}
			session.mu.Unlock()
			
			return true
		})
		
		// Delete expired sessions
		for _, sessionID := range toDelete {
			if sessionInterface, ok := h.sessions.Load(sessionID); ok {
				session := sessionInterface.(*Session)
				close(session.ResponseChan)
				h.sessions.Delete(sessionID)
				log.Printf("Cleaned up expired session: %s", sessionID)
			}
		}
	}
}