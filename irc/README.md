# GoTTY IRC 聊天室模块

这是 GoTTY 项目的 IRC 聊天室功能模块。

## 目录结构

```
irc/
├── config.go           # IRC 服务器配置
├── client.go           # IRC 客户端模型
├── channel.go          # IRC 频道模型
├── server.go           # IRC 服务器核心
├── message.go          # IRC 消息格式和解析
├── handler.go          # WebSocket 处理器
└── test-chat.html      # 测试页面
```

前端代码:
```
js/src/irc/
└── irc-client.ts       # 前端 IRC 客户端库
```

## 功能特性

### 已实现

- ✅ 基础 IRC 服务器框架
- ✅ WebSocket 连接处理
- ✅ 频道管理（加入/离开）
- ✅ 消息广播（频道消息）
- ✅ 昵称管理（更改昵称）
- ✅ 用户列表显示
- ✅ 频道主题设置
- ✅ 系统消息通知
- ✅ IRC 命令解析（/nick, /join, /part, /msg, /topic 等）

### 计划中

- ⏳ 集成到 GoTTY 主服务器
- ⏳ 用户认证系统
- ⏳ 频道权限管理
- ⏳ 私聊功能
- ⏳ 消息历史记录
- ⏳ xterm.js 终端渲染

## IRC 命令

| 命令 | 说明 |
|------|------|
| `/nick <昵称>` | 更改昵称 |
| `/join <频道>` | 加入频道 |
| `/part [频道]` | 离开频道 |
| `/msg <目标> <消息>` | 发送消息 |
| `/me <动作>` | 发送动作消息 |
| `/topic [主题]` | 设置/查看频道主题 |
| `/list` | 列出所有频道 |
| `/whois <昵称>` | 查询用户信息 |
| `/quit` | 断开连接 |
| `/help` | 显示帮助 |

## 使用示例

### 前端 JavaScript

```javascript
import { IRCClient, IRCCommandHandler } from './irc/irc-client';

// 创建客户端
const client = new IRCClient({
  wsUrl: 'ws://localhost:8080/ws',
  defaultChannel: '#general',
  onMessage: (msg) => {
    console.log('Received:', msg);
  }
});

// 连接
await client.connect();

// 发送消息
client.sendMessage('#general', 'Hello, World!');

// 更改昵称
client.setNick('MyNick');

// 加入频道
client.join('#new-channel');

// 断开连接
client.disconnect();
```

### 使用斜杠命令

在聊天界面中直接使用斜杠命令：

```
/nick MyNick
/join #general
/msg #general Hello everyone!
/topic Welcome to our chat!
/part
/quit
```

## 消息格式

### WebSocket 消息格式（客户端 -> 服务器）

```json
{
  "type": "msg",
  "channel": "#general",
  "data": "Hello, World!",
  "timestamp": 1709856000000
}
```

### WebSocket 消息格式（服务器 -> 客户端）

```json
{
  "type": "irc_out",
  "command": "PRIVMSG",
  "params": ["#general"],
  "tags": {
    "prefix": "username"
  },
  "data": "Hello, World!",
  "timestamp": 1709856000000
}
```

## 开发说明

### 编译 Go 代码

```bash
cd /path/to/gotty
make
```

### 编译 TypeScript 代码

```bash
cd /path/to/gotty/js
npm install
npm run build
```

### 运行测试页面

1. 启动 GoTTY 服务器
2. 打开浏览器访问 `irc/test-chat.html`

## API 参考

### IRCClient 类

| 方法 | 说明 |
|------|------|
| `connect()` | 连接到服务器 |
| `disconnect()` | 断开连接 |
| `send(msg)` | 发送消息 |
| `sendCommand(cmd, params, trailing)` | 发送 IRC 命令 |
| `setNick(nick)` | 更改昵称 |
| `join(channel)` | 加入频道 |
| `part(channel)` | 离开频道 |
| `sendMessage(target, message)` | 发送消息 |
| `setTopic(channel, topic)` | 设置频道主题 |
| `listChannels()` | 列出频道 |
| `whois(nick)` | WHOIS 查询 |
| `quit(message)` | 退出 |

## 配置文件

在 `.gotty` 配置文件中添加 IRC 设置：

```yaml
# 启用 IRC 聊天室
enable_irc = true

# IRC 路径
irc_path = "/irc"

# 默认频道
irc_default_channel = "#general"

# 网络名称
irc_network_name = "GoTTY Network"

# 允许用户注册昵称
irc_allow_register = true
```

## 环境变量

```bash
export GOTTY_IRC_ENABLE=true
export GOTTY_IRC_DEFAULT_CHANNEL="#general"
export GOTTY_IRC_NETWORK_NAME="GoTTY Network"
```

## 许可证

与 GoTTY 主项目相同（MIT License）
