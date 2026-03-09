# WebAuthn/Passkeys 认证系统

## 概述

GoTTY 使用 WebAuthn 标准实现 Passkeys 认证，支持以下认证器：

- **iCloud Keychain** (Apple 设备)
- **Google Password Manager** (Chrome 浏览器)
- **Windows Hello** (Windows 设备)
- **KeePassXC** (跨平台密码管理器，2.7.7+)
- **Bitwarden** (密码管理器)
- **YubiKey** / **Titan Key** (硬件安全密钥)

## 安全架构

### 统一认证层

所有敏感端点都通过统一认证中间件保护：

| 端点 | 认证要求 | 说明 |
|------|----------|------|
| `/ws` | ✅ 需要 | 主 WebSocket（终端连接） |
| `/irc/ws` | ✅ 需要 | IRC WebSocket（聊天） |
| `/api/sessions` | ✅ 需要 | Session 管理 |
| `/api/workspaces` | ✅ 需要 | Workspace 管理 |
| `/api/clipboard` | ✅ 需要 | 剪贴板同步 |
| `/api/webauthn/*` | ❌ 公开 | 认证流程 |
| `/api/weather` | ❌ 公开 | 天气信息 |
| `/`, `/js/*`, `/css/*` | ❌ 公开 | 静态资源 |

### Token 传递方式

API 请求支持多种方式传递 token：

1. **Query 参数**：`/api/sessions?token=xxx`
2. **Authorization Header**：`Authorization: Bearer xxx`
3. **Cookie**：`gotty_auth_token=xxx`

WebSocket 连接使用 query 参数：

```javascript
// 主 WebSocket
ws://host/ws?token=xxx

// IRC WebSocket
ws://host/irc/ws?token=xxx
```

## 启用认证

```bash
# 启用认证（写入权限自动启用）
./gotty -A

# 等同于
./gotty -A -w

# 如果想禁用写入权限（只读模式）
./gotty -A --permit-write=false

# 自定义配置
./gotty -A \
  --webauthn-display-name "My Server" \
  --webauthn-session-ttl 168
```

## 默认行为

| 模式 | 写入权限 | 说明 |
|------|----------|------|
| `./gotty` | ❌ 禁用 | 默认只读 |
| `./gotty -w` | ✅ 启用 | 显式启用写入 |
| `./gotty -A` | ✅ 启用 | 认证模式自动启用写入 |
| `./gotty -A --permit-write=false` | ❌ 禁用 | 认证 + 只读 |

## 配置选项

| 选项 | 默认值 | 说明 |
|------|--------|------|
| `--auth` / `-A` | false | 启用 WebAuthn 认证 |
| `--webauthn-display-name` | GoTTY | 显示名称 |
| `--webauthn-data-dir` | ~/.config/gotty/webauthn | 数据存储目录 |
| `--webauthn-session-ttl` | 168 | Session 有效期（小时），默认 7 天 |
| `--webauthn-register-token` | 空 | 注册新 Passkey 所需的令牌 |
| `--webauthn-allow-register` | false | 是否允许注册新 Passkey |

## 使用流程

### 首次访问（注册）

1. 访问 GoTTY 地址
2. 页面显示 "No Passkey registered. Click to register."
3. 点击 "Register Passkey"
4. 浏览器弹出 Passkey 注册界面
5. 使用 TouchID / Windows Hello / 安全密钥完成注册
6. 注册成功后显示 "Passkey registered successfully!"

### 后续访问（登录）

1. 访问 GoTTY 地址
2. 如果有缓存的 token（7 天内登录过），自动跳过认证
3. 否则显示 "Authenticate" 按钮
4. 点击后使用 Passkey 认证
5. 认证成功后进入终端

## 安全性

### Passkey 存储位置

Passkey 存储在**用户设备**上，服务器只存储凭证 ID 和公钥：

| 认证器 | 存储位置 | 同步范围 |
|--------|----------|----------|
| iCloud Keychain | Apple 设备 | 同一 Apple ID |
| Google Password Manager | Chrome 浏览器 | 同一 Google 账号 |
| Windows Hello | Windows 设备 | 不同步 |
| KeePassXC | 本地数据库 | 手动同步 |
| YubiKey | 硬件密钥 | 随身携带 |

### 访问控制

**只有拥有 Passkey 的设备才能访问：**

- ✅ 注册 Passkey 的设备
- ✅ 同步了 Passkey 的其他设备（iCloud/Google 同步）
- ❌ 没有对应 Passkey 的设备

**攻击者无法访问**，即使知道服务器地址：
1. 点击 "Authenticate"
2. 浏览器要求选择 Passkey
3. 设备上没有注册的 Passkey → 认证失败

### Session 机制

- **Token 存储**：浏览器 localStorage
- **有效期**：默认 7 天（可配置）
- **服务端验证**：每次请求都验证 token

```
浏览器 localStorage (gotty_auth_token)
        ↓
前端 authFetch() 自动添加 token
        ↓
服务端 AuthMiddleware 验证
        ↓
auth_sessions.json (服务端 session 存储)
```

## 多设备支持

### 场景 1: 同一生态系统

使用 iCloud Keychain 或 Google Password Manager：
- 所有同步设备自动拥有 Passkey
- 无需额外配置

### 场景 2: 跨生态系统

例如：Mac + Windows + Linux：
1. 在 Mac 上注册 Passkey（存到 iCloud Keychain）
2. Windows 需要单独注册：
   - 设置 `--webauthn-register-token "your-secret-token"`
   - 在 Windows 上访问并使用 token 注册新 Passkey

### 场景 3: 硬件密钥

YubiKey 等硬件密钥：
- 在任意设备插入密钥即可使用
- 最安全的方案

## 文件结构

```
~/.config/gotty/webauthn/
├── webauthn_user.json    # 用户凭证（公钥）
└── auth_sessions.json    # 活跃 session
```

## 故障排除

### 问题：注册后刷新页面仍要求注册

**检查**：
```bash
cat ~/.config/gotty/webauthn/webauthn_user.json
```

### 问题：认证后显示 "Token is invalid or expired"

**解决**：
```bash
rm ~/.config/gotty/webauthn/auth_sessions.json
```

### 问题：浏览器提示 "No passkey available"

**解决**：
1. 使用已注册 Passkey 的设备
2. 或使用 `--webauthn-register-token` 在新设备上注册

### 问题：API 请求返回 401 Unauthorized

**检查**：
1. 浏览器控制台是否有 `gotty_auth_token`
2. Network 面板中请求是否包含 `?token=xxx` 参数

## 重置认证

```bash
# 删除所有凭证和 session，重新开始
rm -rf ~/.config/gotty/webauthn
```

**注意**：此操作会删除所有已注册的 Passkey 关联，需要重新注册。
