# IRC 聊天室使用指南

## 快速开始

### 启动 IRC 聊天室

```bash
# 基本启动
./gotty --irc bash

# 指定默认频道和网络名称
./gotty --irc --irc-channel "#lobby" --irc-network "My Chat Network" bash

# 使用配置文件
cat > ~/.gotty <<EOF
enable_irc = true
irc_default_channel = "#general"
irc_network_name = "GoTTY Network"
EOF
./gotty bash
```

### 访问聊天室

启动后，在浏览器中访问：
- 主终端：`http://localhost:13782/`
- **IRC 聊天室：`http://localhost:13782/irc/`**

## IRC 命令

在聊天室中输入以下命令：

| 命令 | 说明 | 示例 |
|------|------|------|
| `/nick <昵称>` | 更改昵称 | `/nick Alice` |
| `/join <频道>` | 加入频道 | `/join #help` |
| `/part [频道]` | 离开频道 | `/part` 或 `/part #general` |
| `/msg <目标> <消息>` | 发送消息 | `/msg #general 大家好！` |
| `/me <动作>` | 发送动作 | `/me 挥挥手` |
| `/topic [主题]` | 设置主题 | `/topic 欢迎来到聊天室！` |
| `/list` | 列出频道 | `/list` |
| `/whois <昵称>` | 查询用户 | `/whois Alice` |
| `/quit` | 断开连接 | `/quit 再见` |
| `/help` | 显示帮助 | `/help` |

## 配置选项

### 命令行参数

```bash
--irc                 启用 IRC 聊天室模式
--irc-channel #name   设置默认频道（默认：#general）
--irc-network name    设置网络名称（默认：GoTTY Network）
```

### 环境变量

```bash
export GOTTY_IRC=true
export GOTTY_IRC_CHANNEL="#lobby"
export GOTTY_IRC_NETWORK="My Network"
./gotty bash
```

### 配置文件 (~/.gotty)

```yaml
# 启用 IRC 聊天室
enable_irc = true

# 默认频道
irc_default_channel = "#general"

# 网络名称
irc_network_name = "GoTTY Network"
```

## 功能特性

- ✅ 多用户同时在线聊天
- ✅ 频道管理（加入/离开）
- ✅ 实时消息广播
- ✅ 昵称管理（更改昵称）
- ✅ 用户列表显示
- ✅ 频道主题设置
- ✅ 系统消息通知
- ✅ 消息历史记录

## 技术架构

- **后端**: Go 语言实现的轻量级 IRC 服务器
- **前端**: 原生 JavaScript WebSocket 客户端
- **通信**: WebSocket 全双工通信
- **协议**: 简化版 IRC 协议

## 常见问题

### Q: IRC 聊天室和终端模式有什么区别？

A: IRC 聊天室是专门用于聊天的界面，不支持终端命令执行。终端模式 (`/`) 可以运行 shell 命令。

### Q: 消息会保存吗？

A: 当前实现中，消息仅保存在内存中，重启服务器后消息会清除。

### Q: 支持私聊吗？

A: 当前版本支持基本的私聊命令 (`/msg <昵称> <消息>`)，但私聊消息不会显示给其他人。

### Q: 可以多个 GoTTY 实例互联吗？

A: 当前版本每个 GoTTY 实例运行独立的 IRC 服务器，不支持实例间互联。

## 开发计划

- [ ] 消息持久化（Redis/数据库）
- [ ] 多服务器互联
- [ ] 频道权限管理（操作员、禁言）
- [ ] 文件传输
- [ ] 表情符号支持
- [ ] 消息加密（TLS）

## 日志和调试

启动时添加 `--debug` 参数查看详细信息：

```bash
./gotty --irc --debug bash
```

查看日志：
```
IRC chatroom enabled. Network: GoTTY Network, Default channel: #general
HTTP server is listening at: http://0.0.0.0:13782/
```

## 贡献

欢迎提交 Issue 和 Pull Request 来改进 IRC 聊天室功能！
