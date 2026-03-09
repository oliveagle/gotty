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

// IRCSession 表示一个 IRC WebSocket 会话
type IRCSession struct {
	conn      *websocket.Conn
	client    *Client
	server    *Server
	handler   *IRCHandler
	sendChan  chan *WSMessage
	mu        sync.Mutex
	closed    bool
}

// IRCHandler 处理 IRC WebSocket 连接
type IRCHandler struct {
	server   *Server
	upgrader websocket.Upgrader
	sessions map[*IRCSession]bool
	mu       sync.RWMutex
}

// NewIRCHandler 创建一个新的 IRC 处理器
func NewIRCHandler(server *Server) *IRCHandler {
	return &IRCHandler{
		server:   server,
		sessions: make(map[*IRCSession]bool),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // 允许所有来源
			},
		},
	}
}

// HandleWS 处理 WebSocket 连接
func (h *IRCHandler) HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}

	// 创建新会话
	session := &IRCSession{
		conn:     conn,
		server:   h.server,
		handler:  h,
		sendChan: make(chan *WSMessage, 256),
	}

	// 创建客户端 - 先用临时昵称，等待客户端发送 nick
	tempNick := h.server.GenerateNick()
	session.client = NewClient(fmt.Sprintf("%p", conn), tempNick)
	h.server.AddClient(session.client)

	// 注册会话
	h.mu.Lock()
	h.sessions[session] = true
	h.mu.Unlock()

	// 发送欢迎消息
	session.Send(NewSysMessage(fmt.Sprintf("Welcome to %s!", h.server.networkName)))

	// 启动读写 goroutine (先不加入频道，等待 nick 设置)
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

// BroadcastToChannel 向频道中的所有用户发送消息
func (h *IRCHandler) BroadcastToChannel(channelName string, msg *WSMessage, exclude *IRCSession) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for session := range h.sessions {
		if session.client.IsInChannel(channelName) {
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

// Close 关闭会话
func (s *IRCSession) Close() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	s.mu.Unlock()

	// 通知频道中的其他用户
	for channel := range s.client.Channels {
		ch := s.server.GetChannel(channel)
		if ch != nil {
			ch.RemoveUser(s.client)
			partMsg := NewIRCOutMessage("PART", s.client.GetNick(), []string{channel}, fmt.Sprintf("%s left the channel", s.client.GetNick()))
			s.handler.BroadcastToChannel(channel, partMsg, nil)
		}
	}

	// 从处理器移除
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

// handleNick 处理昵称更改
func (s *IRCSession) handleNick(newNick string) {
	if !s.server.ValidateNick(newNick) {
		s.Send(NewSysMessage("Invalid nickname. Use letters, numbers, underscore, and hyphen."))
		return
	}

	// 检查昵称是否已被使用
	if s.server.GetClientByNick(newNick) != nil && s.client.GetNick() != newNick {
		s.Send(NewSysMessage("Nickname already in use"))
		return
	}

	oldNick := s.client.GetNick()
	s.client.SetNick(newNick)
	s.Send(NewSysMessage(fmt.Sprintf("Your nickname is %s", newNick)))

	// 如果是第一次设置昵称（从 guest_ 改为真实昵称），自动加入默认频道
	if strings.HasPrefix(oldNick, "guest_") && len(s.client.Channels) == 0 {
		s.Send(NewSysMessage(fmt.Sprintf("Joining default channel %s", s.server.config.DefaultChannel)))
		s.handleJoin(s.server.config.DefaultChannel)
		return // 不需要广播昵称更改
	}

	// 通知频道中的其他用户
	for channel := range s.client.Channels {
		joinMsg := NewIRCOutMessage("NICK", oldNick, []string{newNick}, fmt.Sprintf("%s is now known as %s", oldNick, newNick))
		s.handler.BroadcastToChannel(channel, joinMsg, nil)
	}
}

// handleJoin 处理加入频道
func (s *IRCSession) handleJoin(channel string) {
	// 验证频道名称
	if !strings.HasPrefix(channel, "#") {
		channel = "#" + channel
	}

	ch := s.server.GetOrCreateChannel(channel)
	ch.AddUser(s.client)

	// 发送加入确认
	s.Send(NewSysMessage(fmt.Sprintf("Joined channel %s", channel)))
	if topic := ch.GetTopic(); topic != "" {
		s.Send(NewSysMessage(fmt.Sprintf("Topic: %s", topic)))
	}

	// 通知频道中的其他用户
	nick := s.client.GetNick()
	joinMsg := NewIRCOutMessage("JOIN", nick, []string{channel}, fmt.Sprintf("%s joined the channel", nick))
	s.handler.BroadcastToChannel(channel, joinMsg, s)

	// 发送频道中的用户列表
	users := s.server.GetNickList(channel)
	s.Send(NewSysMessage(fmt.Sprintf("Users in %s: %s", channel, strings.Join(users, ", "))))

	// 发送历史消息
	for _, msg := range ch.GetMessages() {
		s.Send(NewIRCOutMessage(msg.Command, msg.Prefix, msg.Params, msg.Data))
	}
}

// handlePart 处理离开频道
func (s *IRCSession) handlePart(channel string) {
	ch := s.server.GetChannel(channel)
	if ch == nil {
		s.Send(NewSysMessage("Not in channel " + channel))
		return
	}

	ch.RemoveUser(s.client)

	// 发送离开确认
	s.Send(NewSysMessage(fmt.Sprintf("Left channel %s", channel)))

	// 通知频道中的其他用户
	partMsg := NewIRCOutMessage("PART", s.client.GetNick(), []string{channel}, fmt.Sprintf("%s left the channel", s.client.GetNick()))
	s.handler.BroadcastToChannel(channel, partMsg, nil)
}

// handlePrivmsg 处理私信/频道消息
func (s *IRCSession) handlePrivmsg(target, message string) {
	if strings.HasPrefix(target, "#") {
		// 频道消息
		ch := s.server.GetChannel(target)
		if ch == nil {
			s.Send(NewSysMessage("Not in channel " + target))
			return
		}

		nick := s.client.GetNick()
		msg := NewIRCOutMessage("PRIVMSG", nick, []string{target}, message)

		// 添加到历史记录
		ch.AddMessage(Message{
			Prefix:    nick,
			Command:   "PRIVMSG",
			Params:    []string{target},
			Data:      message,
			Timestamp: time.Now(),
		}, s.server.historyLimit)

		s.handler.BroadcastToChannel(target, msg, nil)
	} else {
		// 私聊消息
		targetClient := s.server.GetClientByNick(target)
		if targetClient == nil {
			s.Send(NewSysMessage("User not found: " + target))
			return
		}

		s.Send(NewSysMessage(fmt.Sprintf("Message to %s: %s", target, message)))
		// TODO: 发送私聊消息到目标用户
	}
}

// handleTopic 处理频道主题
func (s *IRCSession) handleTopic(channel, topic string) {
	ch := s.server.GetChannel(channel)
	if ch == nil {
		s.Send(NewSysMessage("Not in channel " + channel))
		return
	}

	ch.SetTopic(topic)
	s.Send(NewSysMessage(fmt.Sprintf("Topic set to: %s", topic)))

	// 通知频道中的其他用户
	topicMsg := NewIRCOutMessage("TOPIC", s.client.GetNick(), []string{channel}, topic)
	s.handler.BroadcastToChannel(channel, topicMsg, nil)
}

// handleList 处理频道列表
func (s *IRCSession) handleList() {
	channels := s.server.GetChannels()
	var list []string
	for _, ch := range channels {
		users := len(ch.GetUsers())
		list = append(list, fmt.Sprintf("%s (%d users)", ch.Name, users))
	}
	s.Send(NewSysMessage("Channel list: " + strings.Join(list, ", ")))
}

// handleWhois 处理 WHOIS 查询
func (s *IRCSession) handleWhois(nick string) {
	client := s.server.GetClientByNick(nick)
	if client == nil {
		s.Send(NewSysMessage("User not found: " + nick))
		return
	}

	s.Send(NewSysMessage(fmt.Sprintf("%s connected since %s", nick, client.Connected.Format("2006-01-02 15:04:05"))))
}
