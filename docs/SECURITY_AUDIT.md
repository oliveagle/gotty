# 安全审计报告

## 端点认证状态

### HTTP API 端点

| 端点 | 认证 | 保护方式 | 说明 |
|------|------|----------|------|
| `/` | ❌ 公开 | - | 首页，需要显示认证界面 |
| `/js/*` | ❌ 公开 | - | 静态 JS 资源 |
| `/css/*` | ❌ 公开 | - | 静态 CSS 资源 |
| `/favicon.png` | ❌ 公开 | - | 图标 |
| `/auth_token.js` | ❌ 公开 | - | 认证配置（不含敏感信息） |
| `/config.js` | ❌ 公开 | - | 终端配置 |
| `/api/sessions` | ✅ 需要 | `authMiddleware.Wrap()` | Session 列表 |
| `/api/sessions/reorder` | ✅ 需要 | `authMiddleware.Wrap()` | Session 重排序 |
| `/api/sessions/*` | ✅ 需要 | `authMiddleware.Wrap()` | Session CRUD |
| `/api/workspaces` | ✅ 需要 | `authMiddleware.Wrap()` | Workspace 列表 |
| `/api/workspaces/*` | ✅ 需要 | `authMiddleware.Wrap()` | Workspace CRUD |
| `/api/clipboard` | ✅ 需要 | `authMiddleware.Wrap()` | 剪贴板同步 |
| `/api/weather` | ✅ 需要 | `authMiddleware.Wrap()` | 天气 API |
| `/weather-preview.html` | ✅ 需要 | `authMiddleware.Wrap()` | 调试页面 |
| `/api/webauthn/status` | ❌ 公开 | - | 认证状态（不含敏感信息） |
| `/api/webauthn/register/begin` | ❌ 公开 | - | 注册流程 |
| `/api/webauthn/register/finish` | ❌ 公开 | - | 注册流程 |
| `/api/webauthn/login/begin` | ❌ 公开 | - | 登录流程 |
| `/api/webauthn/login/finish` | ❌ 公开 | - | 登录流程 |
| `/api/webauthn/validate` | ❌ 公开 | - | Token 验证 |
| `/irc/` | ✅ 需要 | `authMiddleware.Wrap()` | IRC 页面 |
| `/irc/ws` | ✅ 需要 | `authMiddleware.WrapWS()` | IRC WebSocket |

### WebSocket 端点

| 端点 | 认证 | 保护方式 | 说明 |
|------|------|----------|------|
| `/ws` | ✅ 需要 | handler 内验证 | 主终端连接，通过 init 消息传递 token |

## 认证流程

### 1. WebAuthn 注册流程（公开）

```
用户访问 → /api/webauthn/status → /api/webauthn/register/begin → 浏览器 Passkey → /api/webauthn/register/finish
```

### 2. WebAuthn 登录流程（公开）

```
用户访问 → /api/webauthn/status → /api/webauthn/login/begin → 浏览器 Passkey → /api/webauthn/login/finish → 返回 token
```

### 3. 受保护 API 调用

```
浏览器 localStorage (token) → authFetch() 添加 token → authMiddleware 验证 → 处理请求
```

### 4. WebSocket 连接

```
浏览器 localStorage (token) → connectTerminal() → WebTTY init 消息带 token → processWSConn() 验证
```

## Token 传递方式

### HTTP API

支持三种方式：

1. **Query 参数**：`/api/sessions?token=xxx`
2. **Authorization Header**：`Authorization: Bearer xxx`
3. **Cookie**：`gotty_auth_token=xxx`

### WebSocket

- `/ws`：通过 WebTTY init 消息的 `AuthToken` 字段
- `/irc/ws`：通过 URL query 参数 `?token=xxx`

## 安全检查清单

- [x] 所有 API 端点都有认证保护（除了 WebAuthn 流程）
- [x] WebSocket 连接需要认证
- [x] Token 从 localStorage 正确读取
- [x] Token 在服务端验证
- [x] 静态资源不需要认证
- [x] 认证流程端点不需要认证（否则循环依赖）

## 公开端点合理性分析

| 端点 | 公开原因 |
|------|----------|
| `/` | 用户需要先看到认证界面 |
| `/auth_token.js` | 返回认证配置，不含敏感信息 |
| `/config.js` | 返回终端类型配置 |
| `/api/webauthn/*` | 认证流程必需，否则无法登录 |
| `/js/*`, `/css/*` | 静态资源，不包含敏感数据 |

## 风险评估

### 低风险

- **静态资源公开**：不包含敏感信息
- **WebAuthn 端点公开**：符合 WebAuthn 协议要求

### 需要监控

- **Token 有效期**：默认 7 天，可配置
- **Session 文件权限**：`0600`，仅所有者可读写

## 漏洞分析

### 高风险

#### 1. 参数注入漏洞

**位置**: `backend/localcommand/factory.go:40-48`

```go
if params != nil && len(params["arg"]) > 0 {
    argv = append(argv, params["arg"]...)  // 用户输入直接追加
}
```

**问题**: 当 `PermitArguments` 为 true 时，用户可通过 URL 参数注入命令行参数。

**攻击示例**:
```
http://localhost:8080/?arg=--help
http://localhost:8080/?arg=-e&arg=malicious_command
```

**状态**: 待修复

---

### 中等风险

#### 2. Token 时序攻击

**位置**: `server/webauthn.go:236-241`

```go
return m.registerToken == token  // 直接字符串比较
```

**问题**: 使用非常量时间比较，可能通过时序分析推断 Token。

**状态**: 待修复

#### 3. WebSocket 消息无大小限制

**位置**: `webtty/webtty.go`

**问题**: 没有限制单个消息的最大大小，攻击者可发送大量数据导致内存耗尽。

**状态**: 待修复

#### 4. 会话元数据文件权限过宽

**位置**: `server/session_manager.go:79-80`

```go
os.WriteFile(metadataFile, []byte("[]"), 0644)  // 权限 0644
```

**问题**: 其他用户可读取会话元数据。

**状态**: 待修复

#### 5. 错误信息泄露内部细节

**位置**: `server/webauthn_handlers.go:138-139`

```go
jsonError(w, "Failed to parse credential: "+err.Error(), http.StatusBadRequest)
```

**问题**: 错误详情直接返回给客户端，可能泄露内部实现细节。

**状态**: 待修复

---

### 低风险

#### 6. 会话 ID 熵值

**位置**: `server/session_manager.go:261`

```go
id := randomstring.Generate(8)  // 8 字符约 48 位熵
```

**问题**: 对于高安全场景可能不足，但对于终端共享场景可接受。

**状态**: 暂不修复（符合预期使用场景）

---

## 加固记录

### 2026-03-09 安全加固 (第一轮)

| 漏洞 | 修复方案 | 状态 |
|------|----------|------|
| 参数注入 | 1. `PermitArguments` 默认值改为 false<br>2. 添加 `validateArg()` 和 `sanitizeArgs()` 过滤危险字符 | ✅ 已修复 |
| Token 时序攻击 | 使用 `crypto/subtle.ConstantTimeCompare` | ✅ 已修复 |
| WebSocket 消息大小 | 添加 `maxMessageSize` 限制 (默认 1MB) | ✅ 已修复 |
| 会话元数据权限 | 目录权限 `0700`，文件权限 `0600` | ✅ 已修复 |
| 错误信息泄露 | 返回通用错误消息，详情仅记录日志 | ✅ 已修复 |

### 2026-03-09 安全加固 (第二轮)

| 漏洞 | 严重程度 | 修复方案 | 状态 |
|------|----------|----------|------|
| IRC CSWSH | 严重 | 实现 Origin 验证，仅允许同源和 localhost | ✅ 已修复 |
| XSS (前端) | 高 | 对所有用户输入使用 `escapeHtml()` 转义 | ✅ 已修复 |
| 缺少安全头 | 中 | 添加 CSP, X-Frame-Options 等安全头 | ✅ 已修复 |

### 修复详情

#### 1. 参数注入修复

**文件**: `backend/localcommand/factory.go`

- 添加 `validateArg()` 验证单个参数
- 添加 `sanitizeArgs()` 批量验证
- 阻止危险前缀 (`-e`, `-c`, `--execute`, `--command`, `--`)
- 阻止危险字符 (`;`, `|`, `&`, `$`, `` ` ``, `(`, `)`, `<`, `>`, `\n`, `\r`)

#### 2. Token 时序攻击修复

**文件**: `server/webauthn.go`

```go
// 使用常量时间比较
return subtle.ConstantTimeCompare([]byte(m.registerToken), []byte(token)) == 1
```

#### 3. WebSocket 消息大小限制

**文件**: `webtty/webtty.go`, `webtty/option.go`

- 默认限制 1MB
- 可通过 `WithMaxMessageSize()` 配置

#### 4. 会话元数据权限修复

**文件**: `server/session_manager.go`

- 目录权限: `0755` → `0700`
- 文件权限: `0644` → `0600`

#### 5. 错误信息泄露修复

**文件**: `server/webauthn_handlers.go`

- 移除 `err.Error()` 返回给客户端
- 错误详情仅记录到服务端日志

#### 6. IRC CSWSH 修复 (严重)

**文件**: `irc/handler.go`

原代码允许所有 Origin，存在跨站 WebSocket 劫持风险：

```go
// 危险代码
CheckOrigin: func(r *http.Request) bool {
    return true // 允许所有来源
},
```

修复后验证 Origin：

```go
CheckOrigin: func(r *http.Request) bool {
    origin := r.Header.Get("Origin")
    // 允许无 Origin 请求（CLI 工具）
    if origin == "" { return true }
    // 允许同源请求
    host := r.Host
    if strings.HasPrefix(origin, "http://"+host) { return true }
    // 允许 localhost 开发
    if strings.HasPrefix(origin, "http://localhost") { return true }
    // 拒绝其他来源
    return false
},
```

#### 7. XSS 修复 (高)

**文件**: `resources/index.html`

修复以下 XSS 漏洞：

- IRC 频道名称未转义 → `this.escapeHtml(channel)`
- IRC 昵称未转义 → `this.escapeHtml(nick)`
- Workspace 名称/图标未转义 → `this.escapeHtml(ws.name)`, `this.escapeHtml(ws.icon)`
- 天气数据未转义 → `this.escapeHtml(weatherInfo.city)` 等

#### 8. 安全头添加 (中)

**文件**: `server/middleware.go`

```go
w.Header().Set("X-Content-Type-Options", "nosniff")
w.Header().Set("X-Frame-Options", "SAMEORIGIN")
w.Header().Set("X-XSS-Protection", "1; mode=block")
w.Header().Set("Content-Security-Policy", "default-src 'self'; ...")
w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
w.Header().Set("Permissions-Policy", "clipboard-read=(), clipboard-write=(self)")
```

### 2026-03-09 安全加固 (第三轮)

| 漏洞 | 严重程度 | 修复方案 | 状态 |
|------|----------|----------|------|
| zellij 会话名称注入 | 中高 | 添加 `validateSessionName()` 白名单验证 | ✅ 已修复 |
| zellij 参数注入 | 中高 | 添加 `sanitizeArgs()` 过滤危险字符 | ✅ 已修复 |
| zellij tab 名称注入 | 中 | 添加 `validateTabName()` 过滤危险字符 | ✅ 已修复 |

#### 9. zellij 会话名称验证

**文件**: `backend/zellijcommand/zellij_command.go`

```go
// 白名单正则：只允许字母、数字、下划线、连字符、点
var sessionNameRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,63}$`)

func validateSessionName(name string) error {
    if len(name) > MaxSessionNameLength {
        return fmt.Errorf("session name too long")
    }
    if !sessionNameRegex.MatchString(name) {
        return fmt.Errorf("invalid session name")
    }
    return nil
}
```

#### 10. zellij 参数验证

**文件**: `backend/zellijcommand/factory.go`

- 添加 `validateArg()` 验证单个参数
- 添加 `sanitizeArgs()` 批量验证
- 阻止危险字符 (`;`, `|`, `&`, `$`, `` ` ``, `(`, `)`, `<`, `>`, `\n`, `\r`)

### 2026-03-09 安全加固 (第四轮)

| 漏洞 | 严重程度 | 修复方案 | 状态 |
|------|----------|----------|------|
| 剪贴板无大小限制 | 中 | 添加 `MaxClipboardSize` 限制 (1MB) | ✅ 已修复 |
| 加密使用固定 IV | 高 | 改用 AES-256-GCM 认证加密 | ✅ 已修复 |
| VerifyToken 时序攻击 | 中 | 使用 `crypto/subtle.ConstantTimeCompare` | ✅ 已修复 |

#### 11. 剪贴板大小限制

**文件**: `server/clipboard.go`

```go
const MaxClipboardSize = 1 * 1024 * 1024 // 1MB

func (cm *ClipboardManager) GetClipboardContent() string {
    // ...
    if len(content) > MaxClipboardSize {
        content = content[:MaxClipboardSize]
    }
    // ...
}
```

#### 12. 加密实现修复

**文件**: `server/crypto.go`

原代码使用固定 IV，存在严重安全风险：

```go
// 危险代码
iv := make([]byte, aes.BlockSize)
for i := range iv {
    iv[i] = byte(i)  // 固定 IV！
}
```

修复后使用 AES-256-GCM：

```go
// 使用 GCM 模式进行认证加密
gcm, err := cipher.NewGCM(block)
nonce := make([]byte, gcm.NonceSize())
io.ReadFull(rand.Reader, nonce)  // 随机 nonce
ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
```

---

## 剩余风险

### Token 通过 URL 传递 (中风险) - 已缓解

Token 支持多种传递方式：
- ✅ Authorization header (推荐)
- ✅ Cookie
- ⚠️ URL query parameter (仍有日志风险)

建议生产环境使用 Authorization header 或 HttpOnly Cookie。

---

## 更新日志

- 2026-03-09: 第四轮安全加固 - 修复剪贴板大小限制、加密实现、VerifyToken 时序攻击
- 2026-03-09: 第三轮安全加固 - 修复 zellij 会话名称验证、参数验证
- 2026-03-09: 第二轮安全加固 - 修复 IRC CSWSH、XSS、添加安全头
- 2026-03-09: 添加漏洞分析章节，记录安全审计发现
- 2026-03-09: 添加 `/api/weather`, `/weather-preview.html`, `/irc/` 的认证保护
- 2026-03-09: 修复 `connectTerminal()` 未从 localStorage 获取 token 的问题
