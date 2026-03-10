package irc

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ChannelKey generates a workspace-scoped channel key for internal storage
// For default workspace, returns the channel name unchanged (backward compatibility)
func ChannelKey(workspaceID, channelName string) string {
	if workspaceID == "" || workspaceID == "default" {
		return channelName
	}
	return workspaceID + ":" + channelName
}

// ParseChannelKey extracts workspace ID and channel name from a scoped key
func ParseChannelKey(key string) (workspaceID, channelName string) {
	parts := strings.SplitN(key, ":", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "default", key // Legacy format
}

// Channel 表示一个 IRC 频道
type Channel struct {
	Name        string            // Display name (#general)
	WorkspaceID string            // Workspace ID this channel belongs to
	Key         string            // Full key for storage (workspace_id:#general or #general)
	Topic       string            // Channel topic
	Users       map[string]*Client // Users in channel (key: client ID)
	Messages    []Message         // Message history
	Created     time.Time         // Creation time
	mu          sync.RWMutex
}

// Message 表示一条 IRC 消息
type Message struct {
	Prefix    string            // 消息来源（昵称）
	Command   string            // 命令（PRIVMSG, NOTICE 等）
	Params    []string          // 参数
	Tags      map[string]string // IRCv3 标签
	Data      string            // 消息内容
	Timestamp time.Time         // 时间戳
}

// NewChannel creates a new channel with optional workspace context
func NewChannel(name string) *Channel {
	return &Channel{
		Name:        name,
		WorkspaceID: "default",
		Key:         name,
		Users:       make(map[string]*Client),
		Messages:    make([]Message, 0),
		Created:     time.Now(),
	}
}

// NewChannelWithWorkspace creates a new channel with workspace context
func NewChannelWithWorkspace(name, workspaceID string) *Channel {
	ch := NewChannel(name)
	ch.WorkspaceID = workspaceID
	ch.Key = ChannelKey(workspaceID, name)
	return ch
}

// AddUser 添加用户到频道
func (ch *Channel) AddUser(client *Client) {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	ch.Users[client.ID] = client
	client.JoinChannel(ch.Name)
}

// RemoveUser 从频道移除用户
func (ch *Channel) RemoveUser(client *Client) {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	delete(ch.Users, client.ID)
	client.PartChannel(ch.Name)
}

// GetUser 获取频道中的用户
func (ch *Channel) GetUser(id string) *Client {
	ch.mu.RLock()
	defer ch.mu.RUnlock()
	return ch.Users[id]
}

// GetUsers 获取所有用户
func (ch *Channel) GetUsers() []*Client {
	ch.mu.RLock()
	defer ch.mu.RUnlock()
	users := make([]*Client, 0, len(ch.Users))
	for _, user := range ch.Users {
		users = append(users, user)
	}
	return users
}

// AddMessage 添加消息到频道历史
func (ch *Channel) AddMessage(msg Message, limit int) {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	ch.Messages = append(ch.Messages, msg)
	// 限制历史消息数量
	if len(ch.Messages) > limit {
		ch.Messages = ch.Messages[len(ch.Messages)-limit:]
	}
}

// GetMessages 获取频道历史消息
func (ch *Channel) GetMessages() []Message {
	ch.mu.RLock()
	defer ch.mu.RUnlock()
	result := make([]Message, len(ch.Messages))
	copy(result, ch.Messages)
	return result
}

// SetTopic 设置频道主题
func (ch *Channel) SetTopic(topic string) {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	ch.Topic = topic
}

// GetTopic 获取频道主题
func (ch *Channel) GetTopic() string {
	ch.mu.RLock()
	defer ch.mu.RUnlock()
	return ch.Topic
}

// messageJSON is the JSON format for message persistence
type messageJSON struct {
	Prefix    string            `json:"prefix"`
	Command   string            `json:"command"`
	Params    []string          `json:"params,omitempty"`
	Tags      map[string]string `json:"tags,omitempty"`
	Data      string            `json:"data"`
	Timestamp time.Time         `json:"timestamp"`
}

// SaveMessages saves channel messages to a JSONL file
func (ch *Channel) SaveMessages(dataDir string) error {
	ch.mu.RLock()
	defer ch.mu.RUnlock()

	if len(ch.Messages) == 0 {
		return nil // Nothing to save
	}

	// Determine workspace directory
	wsID := ch.WorkspaceID
	if wsID == "" {
		wsID = "default"
	}

	// Create directory: dataDir/irc/<workspace_id>/
	dir := filepath.Join(dataDir, "irc", wsID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// File path: dataDir/irc/<workspace_id>/<channel>.jsonl
	// Sanitize channel name for filesystem
	channelFile := strings.TrimPrefix(ch.Name, "#") + ".jsonl"
	filePath := filepath.Join(dir, channelFile)

	// Write all messages (overwrite existing file)
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	for _, msg := range ch.Messages {
		mj := messageJSON{
			Prefix:    msg.Prefix,
			Command:   msg.Command,
			Params:    msg.Params,
			Tags:      msg.Tags,
			Data:      msg.Data,
			Timestamp: msg.Timestamp,
		}
		if err := encoder.Encode(mj); err != nil {
			return err
		}
	}

	return nil
}

// LoadMessages loads channel messages from a JSONL file
func (ch *Channel) LoadMessages(dataDir string, limit int) error {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	// Determine workspace directory
	wsID := ch.WorkspaceID
	if wsID == "" {
		wsID = "default"
	}

	// File path
	channelFile := strings.TrimPrefix(ch.Name, "#") + ".jsonl"
	filePath := filepath.Join(dataDir, "irc", wsID, channelFile)

	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No saved messages, that's OK
		}
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	messages := make([]Message, 0)

	for decoder.More() {
		var mj messageJSON
		if err := decoder.Decode(&mj); err != nil {
			continue // Skip malformed entries
		}
		messages = append(messages, Message{
			Prefix:    mj.Prefix,
			Command:   mj.Command,
			Params:    mj.Params,
			Tags:      mj.Tags,
			Data:      mj.Data,
			Timestamp: mj.Timestamp,
		})
	}

	// Apply limit
	if len(messages) > limit {
		messages = messages[len(messages)-limit:]
	}

	ch.Messages = messages
	return nil
}
