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

// RemoveClient 从服务器移除客户端
func (s *Server) RemoveClient(client *Client) {
	s.clientMutex.Lock()
	defer s.clientMutex.Unlock()

	// 离开所有频道
	for channel := range client.Channels {
		s.GetOrCreateChannel(channel).RemoveUser(client)
	}

	delete(s.clients, client.ID)
}

// GetClient 获取客户端
func (s *Server) GetClient(id string) *Client {
	s.clientMutex.RLock()
	defer s.clientMutex.RUnlock()
	return s.clients[id]
}

// GetClientByNick 通过昵称获取客户端
func (s *Server) GetClientByNick(nick string) *Client {
	s.clientMutex.RLock()
	defer s.clientMutex.RUnlock()
	for _, client := range s.clients {
		if client.GetNick() == nick {
			return client
		}
	}
	return nil
}

// GetOrCreateChannel 获取或创建频道
func (s *Server) GetOrCreateChannel(name string) *Channel {
	s.channelMutex.Lock()
	defer s.channelMutex.Unlock()

	if ch, ok := s.channels[name]; ok {
		return ch
	}

	ch := NewChannel(name)
	s.channels[name] = ch
	return ch
}

// GetChannel 获取频道
func (s *Server) GetChannel(name string) *Channel {
	s.channelMutex.RLock()
	defer s.channelMutex.RUnlock()
	return s.channels[name]
}

// GetChannels 获取所有频道
func (s *Server) GetChannels() []*Channel {
	s.channelMutex.RLock()
	defer s.channelMutex.RUnlock()

	channels := make([]*Channel, 0, len(s.channels))
	for _, ch := range s.channels {
		channels = append(channels, ch)
	}
	return channels
}

// RemoveChannel 删除频道
func (s *Server) RemoveChannel(name string) {
	s.channelMutex.Lock()
	defer s.channelMutex.Unlock()
	delete(s.channels, name)
}

// Broadcast 向所有客户端广播消息
func (s *Server) Broadcast(msg Message) {
	s.clientMutex.RLock()
	defer s.clientMutex.RUnlock()

	for _, client := range s.clients {
		// TODO: 发送消息到客户端
		_ = client
	}
}

// BroadcastToChannel 向频道广播消息
func (s *Server) BroadcastToChannel(channelName string, msg Message) {
	ch := s.GetChannel(channelName)
	if ch == nil {
		return
	}

	// 添加消息到频道历史
	ch.AddMessage(msg, s.historyLimit)

	// 发送给频道中的所有用户
	for _, user := range ch.GetUsers() {
		// TODO: 发送消息到用户
		_ = user
	}
}

// GetNickList 获取频道中的用户列表
func (s *Server) GetNickList(channelName string) []string {
	ch := s.GetChannel(channelName)
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
