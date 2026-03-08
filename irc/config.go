package irc

// Config 包含 IRC 服务器的配置选项
type Config struct {
	// 启用 IRC 聊天室
	Enable bool `json:"enable"`

	// 网络名称
	NetworkName string `json:"network_name"`

	// 服务器名称
	ServerName string `json:"server_name"`

	// 默认频道
	DefaultChannel string `json:"default_channel"`

	// 允许用户注册昵称
	AllowRegister bool `json:"allow_register"`

	// 最大昵称长度
	NickLen int `json:"nick_len"`

	// 消息历史限制（条数）
	HistoryLimit int `json:"history_limit"`

	// 启用调试日志
	Debug bool `json:"debug"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Enable:         false,
		NetworkName:    "GoTTY Network",
		ServerName:     "gotty",
		DefaultChannel: "#general",
		AllowRegister:  true,
		NickLen:        30,
		HistoryLimit:   100,
		Debug:          false,
	}
}
