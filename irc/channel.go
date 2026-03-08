package irc

import (
	"sync"
	"time"
)

// Channel 表示一个 IRC 频道
type Channel struct {
	Name      string
	Topic     string
	Users     map[string]*Client
	Messages  []Message
	Created   time.Time
	mu        sync.RWMutex
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

// NewChannel 创建一个新的频道
func NewChannel(name string) *Channel {
	return &Channel{
		Name:     name,
		Users:    make(map[string]*Client),
		Messages: make([]Message, 0),
		Created:  time.Now(),
	}
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
