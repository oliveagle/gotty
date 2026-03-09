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

## 更新日志

- 2026-03-09: 添加 `/api/weather`, `/weather-preview.html`, `/irc/` 的认证保护
- 2026-03-09: 修复 `connectTerminal()` 未从 localStorage 获取 token 的问题
