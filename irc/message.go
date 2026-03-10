package irc

import (
	"encoding/json"
	"strings"
	"time"
)

// WSMessage represents a WebSocket message format
type WSMessage struct {
	Type        string            `json:"type"`                  // irc_in, irc_out, sys, nick, join, part, msg, topic
	Command     string            `json:"command,omitempty"`
	Params      []string          `json:"params,omitempty"`
	Tags        map[string]string `json:"tags,omitempty"`
	Data        string            `json:"data,omitempty"`
	Channel     string            `json:"channel,omitempty"`
	Nick        string            `json:"nick,omitempty"`
	WorkspaceID string            `json:"workspace_id,omitempty"` // Workspace context for this message
	Timestamp   int64             `json:"timestamp"`
}

// NewWSMessage 创建一个新的 WebSocket 消息
func NewWSMessage(msgType, command, data string) *WSMessage {
	return &WSMessage{
		Type:      msgType,
		Command:   command,
		Data:      data,
		Timestamp: time.Now().UnixNano() / 1e6,
	}
}

// NewSysMessage 创建系统消息
func NewSysMessage(data string) *WSMessage {
	return NewWSMessage("sys", "", data)
}

// NewIRCOutMessage 创建 IRC 输出消息
func NewIRCOutMessage(command, prefix string, params []string, data string) *WSMessage {
	msg := &WSMessage{
		Type:      "irc_out",
		Command:   command,
		Params:    params,
		Data:      data,
		Timestamp: time.Now().UnixNano() / 1e6,
	}
	if prefix != "" {
		if msg.Tags == nil {
			msg.Tags = make(map[string]string)
		}
		msg.Tags["prefix"] = prefix
	}
	return msg
}

// ParseWSMessage 解析 WebSocket 消息
func ParseWSMessage(data []byte) (*WSMessage, error) {
	var msg WSMessage
	err := json.Unmarshal(data, &msg)
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

// IRCCommand 表示一个 IRC 协议命令
type IRCCommand struct {
	Prefix   string
	Command  string
	Params   []string
	Trailing string
	Tags     map[string]string
}

// ParseIRCLine 解析 IRC 协议行
func ParseIRCLine(line string) *IRCCommand {
	cmd := &IRCCommand{
		Tags: make(map[string]string),
	}

	remaining := strings.TrimSpace(line)

	// 解析标签 (@tag1=value1;tag2:value2)
	if strings.HasPrefix(remaining, "@") {
		endTags := strings.Index(remaining, " ")
		if endTags == -1 {
			return nil
		}
		tagStr := remaining[1:endTags]
		remaining = strings.TrimSpace(remaining[endTags+1:])

		for _, tag := range strings.Split(tagStr, ";") {
			parts := strings.SplitN(tag, "=", 2)
			if len(parts) == 2 {
				cmd.Tags[parts[0]] = parts[1]
			} else {
				cmd.Tags[parts[0]] = ""
			}
		}
	}

	// 解析前缀 (:prefix)
	if strings.HasPrefix(remaining, ":") {
		endPrefix := strings.Index(remaining, " ")
		if endPrefix == -1 {
			return nil
		}
		cmd.Prefix = remaining[1:endPrefix]
		remaining = strings.TrimSpace(remaining[endPrefix+1:])
	}

	// 解析命令和参数
	parts := strings.Split(remaining, " ")
	if len(parts) == 0 {
		return nil
	}

	cmd.Command = parts[0]

	// 解析参数
	for i := 1; i < len(parts); i++ {
		if strings.HasPrefix(parts[i], ":") {
			// 剩余的是 trailing 参数
			cmd.Trailing = strings.Join(parts[i:], " ")[1:]
			break
		}
		cmd.Params = append(cmd.Params, parts[i])
	}

	return cmd
}

// BuildIRCLine 构建 IRC 协议行
func BuildIRCLine(cmd *IRCCommand) string {
	var sb strings.Builder

	// 添加标签
	if len(cmd.Tags) > 0 {
		sb.WriteString("@")
		tagParts := make([]string, 0, len(cmd.Tags))
		for k, v := range cmd.Tags {
			if v != "" {
				tagParts = append(tagParts, k+"="+v)
			} else {
				tagParts = append(tagParts, k)
			}
		}
		sb.WriteString(strings.Join(tagParts, ";"))
		sb.WriteString(" ")
	}

	// 添加前缀
	if cmd.Prefix != "" {
		sb.WriteString(":")
		sb.WriteString(cmd.Prefix)
		sb.WriteString(" ")
	}

	// 添加命令
	sb.WriteString(cmd.Command)

	// 添加参数
	for _, param := range cmd.Params {
		sb.WriteString(" ")
		sb.WriteString(param)
	}

	// 添加 trailing 参数
	if cmd.Trailing != "" {
		sb.WriteString(" :")
		sb.WriteString(cmd.Trailing)
	}

	sb.WriteString("\r\n")
	return sb.String()
}
