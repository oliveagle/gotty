# IRC 聊天室集成设计文档

## 1. 概述

本文档描述如何在 GoTTY 项目中实现基于 Web 的聊天室功能。

### 1.1 项目目标

- 通过 Web 浏览器提供 IRC 风格的聊天界面
- 支持多用户同时在线聊天
- 保持 GoTTY 的轻量级特性
- 可选嵌入 Ergo IRC 服务器或独立运行

### 1.2 技术选型

| 组件 | 技术 | 说明 |
|------|------|------|
| IRC 服务器 | 内置轻量服务器 | 简化的 IRC 协议实现 |
| 终端前端 | xterm.js | GoTTY 现有组件 |
| WebSocket | gorilla/websocket | GoTTY 现有组件 |
| IRC 客户端 | 原生 JS 实现 | 基于 WebSocket 的 IRC 协议 |

### 1.3 架构方案

由于 Ergo IRC 服务器需要 Go 1.26+，而 GoTTY 使用 Go 1.23，我们采用以下方案：

1. **方案 A（推荐）**: 实现一个简化的内置 IRC 服务器，仅支持基本聊天功能
2. **方案 B**: 使用独立的 Ergo 进程，GoTTY 作为 WebSocket 代理
3. **方案 C**: 使用 `ergochat/irc-go` 协议库实现轻量级服务器

---

## 10. 实现进度

### Phase 1: 基础框架 ✅

- [x] 创建 irc 目录结构
- [x] 实现 IRC 配置模块 (`irc/config.go`)
- [x] 实现 IRC 客户端模型 (`irc/client.go`)
- [x] 实现 IRC 频道模型 (`irc/channel.go`)
- [x] 实现 IRC 服务器核心 (`irc/server.go`)
- [x] 实现 IRC 消息格式 (`irc/message.go`)
- [x] 实现 WebSocket 处理器 (`irc/handler.go`)
- [x] 创建前端 IRC 客户端 (`js/src/irc/irc-client.ts`)
- [x] 创建测试 HTML 页面 (`irc/test-chat.html`)

### Phase 2: 待实现

- [ ] 集成到 GoTTY 主服务器
- [ ] 添加命令行参数支持
- [ ] 添加配置文件支持
- [ ] 实现 xterm.js 终端渲染
- [ ] 添加用户认证
- [ ] 添加频道权限管理

---

## 2. 架构设计

### 2.1 整体架构

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   Web Browser   │     │     GoTTY       │     │   Ergo IRC      │
│                 │     │                 │     │   Server        │
│  ┌───────────┐  │     │  ┌───────────┐  │     │  ┌───────────┐  │
│  │ xterm.js  │  │     │  │ WebSocket │  │     │  │ IRC Core  │  │
│  │   + IRC   │◄─┼─────┼─┤  Handler  │◄─┼─────┼─┤  Engine   │  │
│  │  Client   │  │     │  │           │  │     │  │           │  │
│  └───────────┘  │     │  └───────────┘  │     │  └───────────┘  │
│                 │     │                 │     │                 │
└─────────────────┘     └─────────────────┘     └─────────────────┘
```

### 2.2 组件说明

#### 2.2.1 前端 IRC 客户端
- 基于 xterm.js 实现 IRC 协议解析
- 处理 IRC 命令（/join, /msg, /nick 等）
- 显示频道消息和系统通知

#### 2.2.2 WebSocket 处理器
- 复用 GoTTY 现有 WebSocket 基础设施
- 新增 IRC 协议命令处理
- 管理用户会话状态

#### 2.2.3 嵌入式 Ergo 服务器
- 作为 Go 库运行在 GoTTY 进程中
- 处理 IRC 协议和频道管理
- 支持用户注册和认证

---

## 3. 目录结构

```
gotty/
├── irc/                      # 新增：IRC 相关代码
│   ├── server.go             # Ergo 服务器包装
│   ├── config.go             # IRC 配置
│   ├── channel.go            # 频道管理
│   └── client.go             # IRC 客户端接口
├── js/src/irc/               # 前端 IRC 客户端
│   ├── irc-client.ts         # IRC 协议实现
│   ├── commands.ts           # IRC 命令处理
│   └── ui.ts                 # UI 渲染
├── server/
│   └── irc_handler.go        # IRC WebSocket 处理器
├── vendor/github.com/ergochat/ergo/  # Ergo 子模块
└── docs/
    └── IRC_CHATROOM_DESIGN.md  # 本文档
```

---

## 4. 详细设计

### 4.1 Ergo 服务器集成

#### 4.1.1 配置结构

```go
package irc

type Config struct {
    // 网络配置
    ListenAddress string `json:"listen_address"`
    ListenPort    int    `json:"listen_port"`

    // 服务器信息
    NetworkName   string `json:"network_name"`
    ServerName    string `json:"server_name"`

    // 频道配置
    DefaultChannel string  `json:"default_channel"`
    AutoCreateChan bool    `json:"auto_create_chan"`

    // 用户配置
    AllowRegister  bool    `json:"allow_register"`
    NickLen        int     `json:"nick_len"`

    // TLS 配置
    EnableTLS      bool    `json:"enable_tls"`
    TLSCertFile    string  `json:"tls_cert_file"`
    TLSKeyFile     string  `json:"tls_key_file"`
}
```

#### 4.1.2 服务器包装

```go
type ErgoServer struct {
    config     *Config
    ircServer  *ergo.Server  // Ergo 核心
    listener   net.Listener
    clients    map[string]*Client
    mu         sync.RWMutex
}

func NewErgoServer(cfg *Config) (*ErgoServer, error) {
    // 初始化 Ergo 服务器
    // 返回包装后的服务器实例
}

func (s *ErgoServer) Start() error {
    // 启动 IRC 服务器
}

func (s *ErgoServer) Stop() error {
    // 停止 IRC 服务器
}
```

### 4.2 WebSocket 处理器

#### 4.2.1 IRC 消息格式

```go
type IRCMessage struct {
    Type      string                 `json:"type"`  // irc_in, irc_out, sys
    Command   string                 `json:"command"`
    Params    []string               `json:"params,omitempty"`
    Tags      map[string]string      `json:"tags,omitempty"`
    Data      string                 `json:"data,omitempty"`
    Timestamp int64                  `json:"timestamp"`
}
```

#### 4.2.2 处理器接口

```go
type IRCHandler struct {
    server    *ErgoServer
    upgrader  websocket.Upgrader
    sessions  map[*websocket.Conn]*IRCSession
}

type IRCSession struct {
    conn      *websocket.Conn
    nick      string
    channels  map[string]bool
    mu        sync.Mutex
}

func (h *IRCHandler) HandleWS(conn *websocket.Conn) {
    // 处理 WebSocket 连接
    // 启动读写 goroutine
}
```

### 4.3 前端 IRC 客户端

#### 4.3.1 IRC 协议实现

```typescript
class IRCClient {
    private ws: WebSocket;
    private nick: string;
    private channels: Set<string>;
    private onMessage: (msg: IRCMessage) => void;

    connect(url: string): Promise<void>;
    join(channel: string): void;
    sendMessage(target: string, message: string): void;
    setNick(nick: string): void;
    disconnect(): void;
}
```

#### 4.3.2 IRC 命令

| 命令 | 语法 | 说明 |
|------|------|------|
| /nick | /nick <nickname> | 更改昵称 |
| /join | /join #channel | 加入频道 |
| /part | /part [#channel] | 离开频道 |
| /msg | /msg <nick> <message> | 发送私聊 |
| /me | /me <action> | 发送动作 |
| /list | /list | 列出频道 |
| /whois | /whois <nick> | 查询用户 |
| /quit | /quit [message] | 断开连接 |

---

## 5. 接口设计

### 5.1 启动参数

```bash
# 启动带 IRC 聊天室的 GoTTY
gotty --irc --irc-port 6667 --default-channel "#general" bash
```

### 5.2 配置选项

```yaml
# .gotty IRC 配置

# 启用 IRC 聊天室
enable_irc = true

# IRC 服务器端口
irc_port = "6667"

# 默认频道
irc_default_channel = "#general"

# 允许用户注册
irc_allow_register = true

# 网络名称
irc_network_name = "GoTTY Network"
```

### 5.3 环境变量

```bash
export GOTTY_IRC_ENABLE=true
export GOTTY_IRC_PORT=6667
export GOTTY_IRC_DEFAULT_CHANNEL="#general"
```

---

## 6. 安全考虑

### 6.1 认证机制
- 支持匿名连接（默认昵称 guest_XXX）
- 支持昵称注册（可选）
- 支持密码保护频道

### 6.2 访问控制
- 禁止跨用户消息注入
- 限制消息频率（防洪水）
- 支持操作员命令（kick, ban）

### 6.3 数据隔离
- 每个 GoTTY 实例运行独立的 IRC 服务器
- 频道数据不持久化（重启清除）
- 可选的日志记录功能

---

## 7. 实现计划

### Phase 1: 基础集成 (1-2 周)
- [ ] 添加 Ergo 子模块
- [ ] 创建 IRC 服务器包装器
- [ ] 实现基本 WebSocket 处理器
- [ ] 创建前端 IRC 客户端框架

### Phase 2: 核心功能 (2-3 周)
- [ ] 实现 IRC 命令解析
- [ ] 支持频道操作（join/part/msg）
- [ ] 实现用户昵称管理
- [ ] 添加系统消息通知

### Phase 3: 增强功能 (1-2 周)
- [ ] 支持 TLS 加密连接
- [ ] 实现用户注册系统
- [ ] 添加频道密码保护
- [ ] 实现日志记录

### Phase 4: 优化与测试 (1 周)
- [ ] 性能优化
- [ ] 单元测试
- [ ] 集成测试
- [ ] 文档完善

---

## 8. 参考资料

- [Ergo IRC 服务器文档](https://ergo.chat/doc.html)
- [IRC 协议 RFC 1459](https://datatracker.ietf.org/doc/html/rfc1459)
- [IRCv3 扩展](https://ircv3.net/)
- [xterm.js 文档](https://xtermjs.org/docs/)
- [gorilla/websocket](https://github.com/gorilla/websocket)

---

## 9. 附录

### 9.1 Ergo 配置示例

```yaml
network:
    name: "GoTTY Network"

server:
    name: "gotty.local"
    listeners:
        - "127.0.0.1:6667"

channels:
    registration:
        allowed: true
    default:
        - "#general"
```

### 9.2 消息格式示例

```json
{
    "type": "irc_out",
    "command": "PRIVMSG",
    "params": ["#general"],
    "data": "Hello, World!",
    "timestamp": 1709856000
}
```
