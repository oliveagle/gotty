package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// AGUIEvent represents an AG-UI protocol event
type AGUIEvent struct {
	Type        string      `json:"type"`
	ID          string      `json:"id,omitempty"`
	Role        string      `json:"role,omitempty"`
	Content     string      `json:"content,omitempty"`
	Timestamp   int64       `json:"timestamp"`
	Name        string      `json:"name,omitempty"`
	Args        interface{} `json:"args,omitempty"`
	Result      interface{} `json:"result,omitempty"`
	Key         string      `json:"key,omitempty"`
	Value       interface{} `json:"value,omitempty"`
	Operation   string      `json:"operation,omitempty"`
	Message     string      `json:"message,omitempty"`
	RequestType string      `json:"requestType,omitempty"`
	Schema      interface{} `json:"schema,omitempty"`
}

// AGUIChatRequest represents a chat message request
type AGUIChatRequest struct {
	Content   string `json:"content"`
	Timestamp int64  `json:"timestamp"`
}

// AGUIToolResultRequest represents a tool result submission
type AGUIToolResultRequest struct {
	CallID string      `json:"callId"`
	Result interface{} `json:"result"`
}

// AGUIHumanResponseRequest represents a human-in-the-loop response
type AGUIHumanResponseRequest struct {
	RequestID string      `json:"requestId"`
	Data      interface{} `json:"data"`
}

// AGUIStateRequest represents a state update request
type AGUIStateRequest struct {
	Key       string      `json:"key"`
	Value     interface{} `json:"value"`
	Operation string      `json:"operation"` // "set", "delete", "merge"
}

// handleAGUI handles AG-UI SSE connections
func (server *Server) handleAGUI(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// Get flusher
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Get auth token from query parameter
	token := r.URL.Query().Get("token")

	// Validate token if auth is enabled
	if server.options.EnableAuth {
		if token == "" {
			http.Error(w, "Unauthorized: token required", http.StatusUnauthorized)
			return
		}
		if !server.authSessionMgr.ValidateToken(token) {
			http.Error(w, "Unauthorized: invalid token", http.StatusUnauthorized)
			return
		}
	}

	log.Printf("[AG-UI] New SSE connection from %s", r.RemoteAddr)

	// Send initial connection confirmation
	sendSSEEvent(w, flusher, "message", AGUIEvent{
		Type:      "message",
		ID:        "sys_001",
		Role:      "system",
		Content:   "Connected to AG-UI endpoint",
		Timestamp: time.Now().UnixMilli(),
	})

	// Send demo events to show functionality
	sendDemoEvents(w, flusher)

	// Keep connection alive
	<-r.Context().Done()
	log.Printf("[AG-UI] SSE connection closed: %s", r.RemoteAddr)
}

// sendSSEEvent sends a single SSE event
func sendSSEEvent(w http.ResponseWriter, flusher http.Flusher, eventType string, event AGUIEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("[AG-UI] Failed to marshal event: %v", err)
		return
	}

	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, data)
	flusher.Flush()
}

// sendDemoEvents sends demonstration events to showcase AG-UI functionality
func sendDemoEvents(w http.ResponseWriter, flusher http.Flusher) {
	// Welcome message
	time.Sleep(500 * time.Millisecond)
	sendSSEEvent(w, flusher, "message", AGUIEvent{
		Type:      "message",
		ID:        "msg_001",
		Role:      "assistant",
		Content:   "👋 你好！我是你的 AI 助手。我可以通过工具调用来帮助你完成各种任务。",
		Timestamp: time.Now().UnixMilli(),
	})

	// Show tool call example
	time.Sleep(800 * time.Millisecond)
	sendSSEEvent(w, flusher, "tool_call", AGUIEvent{
		Type:      "tool_call",
		ID:        "demo_call_001",
		Name:      "get_weather",
		Args:      map[string]string{"location": "北京"},
		Timestamp: time.Now().UnixMilli(),
	})

	// Show state update
	time.Sleep(500 * time.Millisecond)
	sendSSEEvent(w, flusher, "state_update", AGUIEvent{
		Type:      "state_update",
		Key:       "demo.status",
		Value:     "查询天气中...",
		Operation: "set",
		Timestamp: time.Now().UnixMilli(),
	})

	// Show another message
	time.Sleep(1000 * time.Millisecond)
	sendSSEEvent(w, flusher, "message", AGUIEvent{
		Type:      "message",
		ID:        "msg_002",
		Role:      "assistant",
		Content:   "📊 北京当前天气：晴朗，温度 25°C，空气质量：优\n\n这是一个演示消息。要连接真实的 AI Agent，请在上方输入真实的 AG-UI 端点地址。",
		Timestamp: time.Now().UnixMilli(),
	})
}

// handleAGUIChat handles chat message submissions
func (server *Server) handleAGUIChat(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Validate auth if enabled
	if server.options.EnableAuth {
		token := r.URL.Query().Get("token")
		if token == "" || !server.authSessionMgr.ValidateToken(token) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	var req AGUIChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if req.Content == "" {
		http.Error(w, "Content is required", http.StatusBadRequest)
		return
	}

	// Log the message (in real implementation, forward to Agent)
	log.Printf("[AG-UI] Received chat message: %s", req.Content)

	// In a real implementation, this would forward to the Agent
	// For now, just acknowledge receipt
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"message": "Message received (demo mode)",
	})
}

// handleAGUIToolResult handles tool result submissions
func (server *Server) handleAGUIToolResult(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Validate auth if enabled
	if server.options.EnableAuth {
		token := r.URL.Query().Get("token")
		if token == "" || !server.authSessionMgr.ValidateToken(token) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	var req AGUIToolResultRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if req.CallID == "" {
		http.Error(w, "callId is required", http.StatusBadRequest)
		return
	}

	// Log the tool result (in real implementation, forward to Agent)
	resultJSON, _ := json.Marshal(req.Result)
	log.Printf("[AG-UI] Received tool result for %s: %s", req.CallID, string(resultJSON))

	// In a real implementation, this would forward to the Agent
	// For now, just acknowledge receipt
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"message": "Tool result received (demo mode)",
	})
}

// handleAGUIHumanResponse handles human-in-the-loop responses
func (server *Server) handleAGUIHumanResponse(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Validate auth if enabled
	if server.options.EnableAuth {
		token := r.URL.Query().Get("token")
		if token == "" || !server.authSessionMgr.ValidateToken(token) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	var req AGUIHumanResponseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if req.RequestID == "" {
		http.Error(w, "requestId is required", http.StatusBadRequest)
		return
	}

	// Log the response (in real implementation, forward to Agent)
	dataJSON, _ := json.Marshal(req.Data)
	log.Printf("[AG-UI] Received HITL response for %s: %s", req.RequestID, string(dataJSON))

	// In a real implementation, this would forward to the Agent
	// For now, just acknowledge receipt
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"message": "Human response received (demo mode)",
	})
}

// handleAGUIState handles state update requests
func (server *Server) handleAGUIState(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Validate auth if enabled
	if server.options.EnableAuth {
		token := r.URL.Query().Get("token")
		if token == "" || !server.authSessionMgr.ValidateToken(token) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	var req AGUIStateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if req.Key == "" {
		http.Error(w, "key is required", http.StatusBadRequest)
		return
	}

	// Log the state update (in real implementation, sync with Agent)
	valueJSON, _ := json.Marshal(req.Value)
	log.Printf("[AG-UI] Received state update: %s = %s (op: %s)", req.Key, string(valueJSON), req.Operation)

	// In a real implementation, this would update shared state
	// For now, just acknowledge receipt
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"message": "State update received (demo mode)",
	})
}
