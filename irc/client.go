package irc

import (
	"sync"
	"time"
)

// Client 表示一个 IRC 客户端连接
type Client struct {
	ID        string
	Nick      string
	User      string
	RealName  string
	Connected time.Time
	Channels  map[string]bool
	mu        sync.RWMutex
}

// NewClient 创建一个新的客户端
func NewClient(id, nick string) *Client {
	return &Client{
		ID:        id,
		Nick:      nick,
		Connected: time.Now(),
		Channels:  make(map[string]bool),
	}
}

// SetNick 设置客户端昵称
func (c *Client) SetNick(nick string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Nick = nick
}

// GetNick 获取客户端昵称
func (c *Client) GetNick() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Nick
}

// JoinChannel 加入频道
func (c *Client) JoinChannel(channel string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Channels[channel] = true
}

// PartChannel 离开频道
func (c *Client) PartChannel(channel string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.Channels, channel)
}

// IsInChannel 检查是否在频道中
func (c *Client) IsInChannel(channel string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Channels[channel]
}
