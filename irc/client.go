package irc

import (
	"sync"
	"time"
)

// Client represents an IRC client connection
type Client struct {
	ID          string            // Unique client ID
	Nick        string            // Nickname
	User        string            // Username
	RealName    string            // Real name
	WorkspaceID string            // Workspace this client belongs to
	Connected   time.Time         // Connection time
	Channels    map[string]bool   // Set of channel keys (workspace-scoped)
	mu          sync.RWMutex
}

// NewClient creates a new client
func NewClient(id, nick string) *Client {
	return &Client{
		ID:          id,
		Nick:        nick,
		WorkspaceID: "default",
		Connected:   time.Now(),
		Channels:    make(map[string]bool),
	}
}

// NewClientWithWorkspace creates a new client with workspace context
func NewClientWithWorkspace(id, nick, workspaceID string) *Client {
	c := NewClient(id, nick)
	if workspaceID != "" {
		c.WorkspaceID = workspaceID
	}
	return c
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
