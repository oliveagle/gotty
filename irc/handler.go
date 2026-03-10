package irc

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// IRCSession represents an IRC WebSocket session
type IRCSession struct {
	conn        *websocket.Conn
	client      *Client
	server      *Server
	handler     *IRCHandler
	workspaceID string            // Workspace ID for this session
	sendChan    chan *WSMessage
	mu          sync.Mutex
	closed      bool
}

// IRCHandler 处理 IRC WebSocket 连接
type IRCHandler struct {
	server     *Server
	upgrader   websocket.Upgrader
	sessions   map[*IRCSession]bool
	mu         sync.RWMutex
	allowOrigins []string // Allowed origins for CSRF protection
}

// NewIRCHandler 创建一个新的 IRC 处理器
func NewIRCHandler(server *Server) *IRCHandler {
	return &IRCHandler{
		server:   server,
		sessions: make(map[*IRCSession]bool),
		allowOrigins: []string{}, // Will be set via SetAllowedOrigins
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				// Security: Validate Origin header to prevent CSWSH attacks
				origin := r.Header.Get("Origin")

				// Allow requests with no Origin (e.g., from CLI tools or same-origin)
				if origin == "" {
					return true
				}

				// Allow same-origin requests
				host := r.Host
				if strings.HasPrefix(origin, "http://"+host) || strings.HasPrefix(origin, "https://"+host) {
					return true
				}

				// Allow localhost for development
				if strings.HasPrefix(origin, "http://localhost") || strings.HasPrefix(origin, "https://localhost") {
					return true
				}

				// Allow 127.0.0.1 for development
				if strings.HasPrefix(origin, "http://127.0.0.1") || strings.HasPrefix(origin, "https://127.0.0.1") {
					return true
				}

				// Reject all other origins
				log.Printf("[IRC] Rejected WebSocket connection from Origin: %s", origin)
				return false
			},
		},
	}
}

// SetAllowedOrigins sets additional allowed origins
func (h *IRCHandler) SetAllowedOrigins(origins []string) {
	h.allowOrigins = origins
}

// HandleWS handles WebSocket connections with workspace context
func (h *IRCHandler) HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}

	// Extract workspace_id from query params
	workspaceID := r.URL.Query().Get("workspace_id")
	if workspaceID == "" {
		workspaceID = "default"
	}

	// Create new session
	session := &IRCSession{
		conn:        conn,
		server:      h.server,
		handler:     h,
		workspaceID: workspaceID,
		sendChan:    make(chan *WSMessage, 256),
	}

	// Create client with workspace context - use temporary nick, wait for client to send nick
	tempNick := h.server.GenerateNick()
	session.client = NewClientWithWorkspace(fmt.Sprintf("%p", conn), tempNick, workspaceID)
	h.server.AddClient(session.client)

	// Register session
	h.mu.Lock()
	h.sessions[session] = true
	h.mu.Unlock()

	// Send welcome message
	session.Send(NewSysMessage(fmt.Sprintf("Welcome to %s! (Workspace: %s)", h.server.networkName, workspaceID)))

	// Start read/write goroutines (don't join channel yet, wait for nick)
	go session.readLoop()
	go session.writeLoop()
}

// RemoveSession 移除会话
func (h *IRCHandler) RemoveSession(session *IRCSession) {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.sessions, session)
	h.server.RemoveClient(session.client)
}

// BroadcastToChannel broadcasts a message to all users in a workspace-scoped channel
func (h *IRCHandler) BroadcastToChannel(workspaceID, channelName string, msg *WSMessage, exclude *IRCSession) {
	channelKey := ChannelKey(workspaceID, channelName)

	h.mu.RLock()
	defer h.mu.RUnlock()

	for session := range h.sessions {
		if session.client.IsInChannel(channelKey) {
			if exclude != nil && session == exclude {
				continue
			}
			session.Send(msg)
		}
	}
}

// Send 发送消息到客户端
func (s *IRCSession) Send(msg *WSMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return
	}

	select {
	case s.sendChan <- msg:
	default:
		// 发送队列已满
		log.Println("Send queue full for client", s.client.GetNick())
	}
}

// readLoop 读取 WebSocket 消息
func (s *IRCSession) readLoop() {
	defer func() {
		s.conn.Close()
		s.Close()
	}()

	for {
		_, message, err := s.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Println("WebSocket read error:", err)
			}
			break
		}

		// 解析消息
		wsMsg, err := ParseWSMessage(message)
		if err != nil {
			s.Send(NewSysMessage("Error parsing message: " + err.Error()))
			continue
		}

		// 处理消息
		s.handleMessage(wsMsg)
	}
}

// writeLoop 写入 WebSocket 消息
func (s *IRCSession) writeLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case msg := <-s.sendChan:
			data, err := json.Marshal(msg)
			if err != nil {
				log.Println("Error marshaling message:", err)
				continue
			}

			s.mu.Lock()
			err = s.conn.WriteMessage(websocket.TextMessage, data)
			s.mu.Unlock()

			if err != nil {
				log.Println("WebSocket write error:", err)
				return
			}

		case <-ticker.C:
			// 发送 ping
			s.mu.Lock()
			err := s.conn.WriteMessage(websocket.PingMessage, nil)
			s.mu.Unlock()
			if err != nil {
				log.Println("Ping error:", err)
				return
			}
		}
	}
}

// Close closes the session
func (s *IRCSession) Close() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	s.mu.Unlock()

	// Notify other users in channels
	for channelKey := range s.client.Channels {
		workspaceID, channelName := ParseChannelKey(channelKey)
		ch := s.server.GetChannel(workspaceID, channelName)
		if ch != nil {
			ch.RemoveUser(s.client)
			partMsg := NewIRCOutMessage("PART", s.client.GetNick(), []string{channelName}, fmt.Sprintf("%s left the channel", s.client.GetNick()))
			s.handler.BroadcastToChannel(workspaceID, channelName, partMsg, nil)
		}
	}

	// Remove from handler
	s.handler.RemoveSession(s)
}

// handleMessage 处理接收到的消息
func (s *IRCSession) handleMessage(msg *WSMessage) {
	switch msg.Type {
	case "irc_in":
		s.handleIRCIn(msg.Data)
	case "nick":
		s.handleNick(msg.Nick)
	case "join":
		s.handleJoin(msg.Channel)
	case "part":
		s.handlePart(msg.Channel)
	case "msg":
		s.handlePrivmsg(msg.Channel, msg.Data)
	case "topic":
		s.handleTopic(msg.Channel, msg.Data)
	default:
		s.Send(NewSysMessage("Unknown message type: " + msg.Type))
	}
}

// handleIRCIn 处理原始 IRC 输入
func (s *IRCSession) handleIRCIn(data string) {
	cmd := ParseIRCLine(data)
	if cmd == nil {
		s.Send(NewSysMessage("Error parsing IRC command"))
		return
	}

	switch strings.ToUpper(cmd.Command) {
	case "NICK":
		if len(cmd.Params) > 0 {
			s.handleNick(cmd.Params[0])
		}
	case "JOIN":
		if len(cmd.Params) > 0 {
			s.handleJoin(cmd.Params[0])
		}
	case "PART":
		if len(cmd.Params) > 0 {
			s.handlePart(cmd.Params[0])
		}
	case "PRIVMSG":
		if len(cmd.Params) >= 2 {
			target := cmd.Params[0]
			message := cmd.Trailing
			s.handlePrivmsg(target, message)
		}
	case "TOPIC":
		if len(cmd.Params) >= 1 {
			s.handleTopic(cmd.Params[0], cmd.Trailing)
		}
	case "LIST":
		s.handleList()
	case "WHOIS":
		if len(cmd.Params) > 0 {
			s.handleWhois(cmd.Params[0])
		}
	case "QUIT":
		s.Close()
	case "PING":
		// 响应 PING
		s.Send(NewWSMessage("irc_out", "PONG", cmd.Trailing))
	default:
		s.Send(NewSysMessage("Unknown command: " + cmd.Command))
	}
}

// handleNick handles nickname changes
func (s *IRCSession) handleNick(newNick string) {
	if !s.server.ValidateNick(newNick) {
		s.Send(NewSysMessage("Invalid nickname. Use letters, numbers, underscore, and hyphen."))
		return
	}

	// Check if nickname is already in use within the same workspace
	if s.server.GetClientByNick(newNick, s.workspaceID) != nil && s.client.GetNick() != newNick {
		s.Send(NewSysMessage("Nickname already in use"))
		return
	}

	oldNick := s.client.GetNick()
	s.client.SetNick(newNick)
	s.Send(NewSysMessage(fmt.Sprintf("Your nickname is %s", newNick)))

	// If this is the first nick change (from guest_ to real nick), auto-join default channel
	if strings.HasPrefix(oldNick, "guest_") && len(s.client.Channels) == 0 {
		s.Send(NewSysMessage(fmt.Sprintf("Joining default channel %s", s.server.config.DefaultChannel)))
		s.handleJoin(s.server.config.DefaultChannel)
		return // No need to broadcast nick change
	}

	// Notify other users in channels
	for channelKey := range s.client.Channels {
		workspaceID, channelName := ParseChannelKey(channelKey)
		joinMsg := NewIRCOutMessage("NICK", oldNick, []string{newNick}, fmt.Sprintf("%s is now known as %s", oldNick, newNick))
		s.handler.BroadcastToChannel(workspaceID, channelName, joinMsg, nil)
	}
}

// handleJoin handles joining a channel
func (s *IRCSession) handleJoin(channel string) {
	// Validate channel name
	if !strings.HasPrefix(channel, "#") {
		channel = "#" + channel
	}

	// Get or create workspace-scoped channel
	ch := s.server.GetOrCreateChannel(s.workspaceID, channel)
	ch.AddUser(s.client)

	// Send join confirmation
	s.Send(NewSysMessage(fmt.Sprintf("Joined channel %s", channel)))
	if topic := ch.GetTopic(); topic != "" {
		s.Send(NewSysMessage(fmt.Sprintf("Topic: %s", topic)))
	}

	// Notify other users in the channel
	nick := s.client.GetNick()
	joinMsg := NewIRCOutMessage("JOIN", nick, []string{channel}, fmt.Sprintf("%s joined the channel", nick))
	s.handler.BroadcastToChannel(s.workspaceID, channel, joinMsg, s)

	// Send user list in the channel
	users := s.server.GetNickList(s.workspaceID, channel)
	s.Send(NewSysMessage(fmt.Sprintf("Users in %s: %s", channel, strings.Join(users, ", "))))

	// Send message history
	for _, msg := range ch.GetMessages() {
		s.Send(NewIRCOutMessage(msg.Command, msg.Prefix, msg.Params, msg.Data))
	}
}

// handlePart handles leaving a channel
func (s *IRCSession) handlePart(channel string) {
	ch := s.server.GetChannel(s.workspaceID, channel)
	if ch == nil {
		s.Send(NewSysMessage("Not in channel " + channel))
		return
	}

	ch.RemoveUser(s.client)

	// Send part confirmation
	s.Send(NewSysMessage(fmt.Sprintf("Left channel %s", channel)))

	// Notify other users in the channel
	partMsg := NewIRCOutMessage("PART", s.client.GetNick(), []string{channel}, fmt.Sprintf("%s left the channel", s.client.GetNick()))
	s.handler.BroadcastToChannel(s.workspaceID, channel, partMsg, nil)
}

// handlePrivmsg handles private messages and channel messages
func (s *IRCSession) handlePrivmsg(target, message string) {
	if strings.HasPrefix(target, "#") {
		// Channel message
		ch := s.server.GetChannel(s.workspaceID, target)
		if ch == nil {
			s.Send(NewSysMessage("Not in channel " + target))
			return
		}

		nick := s.client.GetNick()
		msg := NewIRCOutMessage("PRIVMSG", nick, []string{target}, message)

		// Add to history
		ch.AddMessage(Message{
			Prefix:    nick,
			Command:   "PRIVMSG",
			Params:    []string{target},
			Data:      message,
			Timestamp: time.Now(),
		}, s.server.historyLimit)

		// Save to disk (async)
		if s.server.dataDir != "" {
			go ch.SaveMessages(s.server.dataDir)
		}

		s.handler.BroadcastToChannel(s.workspaceID, target, msg, nil)
	} else {
		// Private message
		targetClient := s.server.GetClientByNick(target, s.workspaceID)
		if targetClient == nil {
			s.Send(NewSysMessage("User not found: " + target))
			return
		}

		s.Send(NewSysMessage(fmt.Sprintf("Message to %s: %s", target, message)))
		// TODO: Send private message to target user
	}
}

// handleTopic handles channel topic changes
func (s *IRCSession) handleTopic(channel, topic string) {
	ch := s.server.GetChannel(s.workspaceID, channel)
	if ch == nil {
		s.Send(NewSysMessage("Not in channel " + channel))
		return
	}

	ch.SetTopic(topic)
	s.Send(NewSysMessage(fmt.Sprintf("Topic set to: %s", topic)))

	// Notify other users in the channel
	topicMsg := NewIRCOutMessage("TOPIC", s.client.GetNick(), []string{channel}, topic)
	s.handler.BroadcastToChannel(s.workspaceID, channel, topicMsg, nil)
}

// handleList handles channel list request
func (s *IRCSession) handleList() {
	channels := s.server.GetChannelsByWorkspace(s.workspaceID)
	var list []string
	for _, ch := range channels {
		users := len(ch.GetUsers())
		list = append(list, fmt.Sprintf("%s (%d users)", ch.Name, users))
	}
	s.Send(NewSysMessage("Channel list: " + strings.Join(list, ", ")))
}

// handleWhois handles WHOIS query
func (s *IRCSession) handleWhois(nick string) {
	client := s.server.GetClientByNick(nick, s.workspaceID)
	if client == nil {
		s.Send(NewSysMessage("User not found: " + nick))
		return
	}

	s.Send(NewSysMessage(fmt.Sprintf("%s connected since %s", nick, client.Connected.Format("2006-01-02 15:04:05"))))
}
