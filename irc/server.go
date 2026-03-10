package irc

import (
	"fmt"
	"regexp"
	"sync"
	"time"
)

// Server 是一个简化的 IRC 服务器
type Server struct {
	config       *Config
	name         string
	networkName  string
	dataDir      string            // Directory for message persistence
	clients      map[string]*Client
	channels     map[string]*Channel
	clientMutex  sync.RWMutex
	channelMutex sync.RWMutex
	historyLimit int
}

// NewServer 创建一个新的 IRC 服务器
func NewServer(config *Config) *Server {
	return &Server{
		config:       config,
		name:         config.ServerName,
		networkName:  config.NetworkName,
		dataDir:      config.DataDir,
		clients:      make(map[string]*Client),
		channels:     make(map[string]*Channel),
		historyLimit: config.HistoryLimit,
	}
}

// nickRegex 验证昵称格式
var nickRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]{2,30}$`)

// ValidateNick 验证昵称是否合法
func (s *Server) ValidateNick(nick string) bool {
	if len(nick) > s.config.NickLen {
		return false
	}
	return nickRegex.MatchString(nick)
}

// GenerateNick 生成一个临时昵称
func (s *Server) GenerateNick() string {
	return fmt.Sprintf("guest_%d", time.Now().UnixNano()%10000)
}

// AddClient 添加客户端到服务器
func (s *Server) AddClient(client *Client) {
	s.clientMutex.Lock()
	defer s.clientMutex.Unlock()
	s.clients[client.ID] = client
}

// RemoveClient removes a client from the server
func (s *Server) RemoveClient(client *Client) {
	s.clientMutex.Lock()
	defer s.clientMutex.Unlock()

	// Leave all channels (channels are stored with workspace-scoped keys)
	for channelKey := range client.Channels {
		if ch := s.getChannelByKey(channelKey); ch != nil {
			ch.RemoveUser(client)
		}
	}

	delete(s.clients, client.ID)
}

// GetClient 获取客户端
func (s *Server) GetClient(id string) *Client {
	s.clientMutex.RLock()
	defer s.clientMutex.RUnlock()
	return s.clients[id]
}

// GetClientByNick gets a client by nickname within a workspace
// If workspaceID is empty, searches globally (for backward compatibility)
func (s *Server) GetClientByNick(nick string, workspaceID string) *Client {
	s.clientMutex.RLock()
	defer s.clientMutex.RUnlock()
	for _, client := range s.clients {
		if client.GetNick() == nick {
			// If workspaceID specified, check workspace match
			if workspaceID != "" && client.WorkspaceID != workspaceID {
				continue
			}
			return client
		}
	}
	return nil
}

// GetClientByNickGlobal gets a client by nickname globally (legacy method)
func (s *Server) GetClientByNickGlobal(nick string) *Client {
	return s.GetClientByNick(nick, "")
}

// getChannelByKey gets a channel by its internal key (workspace-scoped)
func (s *Server) getChannelByKey(key string) *Channel {
	s.channelMutex.RLock()
	defer s.channelMutex.RUnlock()
	return s.channels[key]
}

// GetOrCreateChannel gets or creates a workspace-scoped channel
func (s *Server) GetOrCreateChannel(workspaceID, channelName string) *Channel {
	key := ChannelKey(workspaceID, channelName)

	s.channelMutex.Lock()
	defer s.channelMutex.Unlock()

	if ch, ok := s.channels[key]; ok {
		return ch
	}

	ch := NewChannelWithWorkspace(channelName, workspaceID)

	// Load message history if dataDir is set
	if s.dataDir != "" {
		ch.LoadMessages(s.dataDir, s.historyLimit)
	}

	s.channels[key] = ch
	return ch
}

// GetChannel gets a workspace-scoped channel
func (s *Server) GetChannel(workspaceID, channelName string) *Channel {
	key := ChannelKey(workspaceID, channelName)
	return s.getChannelByKey(key)
}

// GetChannelsByWorkspace gets all channels for a workspace
func (s *Server) GetChannelsByWorkspace(workspaceID string) []*Channel {
	s.channelMutex.RLock()
	defer s.channelMutex.RUnlock()

	var channels []*Channel
	for key, ch := range s.channels {
		wsID, _ := ParseChannelKey(key)
		if wsID == workspaceID || (workspaceID == "default" && wsID == "") {
			channels = append(channels, ch)
		}
	}
	return channels
}

// GetChannels gets all channels (for backward compatibility, returns default workspace channels)
func (s *Server) GetChannels() []*Channel {
	return s.GetChannelsByWorkspace("default")
}

// RemoveChannel removes a workspace-scoped channel
func (s *Server) RemoveChannel(workspaceID, channelName string) {
	key := ChannelKey(workspaceID, channelName)
	s.channelMutex.Lock()
	defer s.channelMutex.Unlock()
	delete(s.channels, key)
}

// Broadcast broadcasts a message to all clients
func (s *Server) Broadcast(msg Message) {
	s.clientMutex.RLock()
	defer s.clientMutex.RUnlock()

	for _, client := range s.clients {
		// TODO: 发送消息到客户端
		_ = client
	}
}

// BroadcastToChannel broadcasts a message to a workspace-scoped channel
func (s *Server) BroadcastToChannel(workspaceID, channelName string, msg Message) {
	ch := s.GetChannel(workspaceID, channelName)
	if ch == nil {
		return
	}

	// Add message to channel history
	ch.AddMessage(msg, s.historyLimit)

	// Save message to disk (async)
	if s.dataDir != "" {
		go ch.SaveMessages(s.dataDir)
	}

	// Send to all users in the channel
	for _, user := range ch.GetUsers() {
		// TODO: 发送消息到用户
		_ = user
	}
}

// GetNickList gets the list of nicknames in a workspace-scoped channel
func (s *Server) GetNickList(workspaceID, channelName string) []string {
	ch := s.GetChannel(workspaceID, channelName)
	if ch == nil {
		return nil
	}

	users := ch.GetUsers()
	nicks := make([]string, len(users))
	for i, user := range users {
		nicks[i] = user.GetNick()
	}
	return nicks
}
