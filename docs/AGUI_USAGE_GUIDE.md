# AG-UI 使用指南

> **文档创建时间**: 2026-03-11
> **基于协议**: CopilotKit AG-UI Protocol
> **官方文档**: https://docs.copilotkit.ai

---

## 快速开始

### 1. 连接 AG-UI

在 GoTTY 右侧面板中：

1. 点击 **AG UI** 标签页
2. 输入 AG-UI 端点地址（例如：`http://localhost:8000/api/agui`）
3. 点击 🔌 连接按钮

**状态指示器**:
- 🟢 **Connected** - 已连接
- 🟡 **Connecting** - 连接中
- 🔴 **Disconnected** - 未连接
- ⚫ **Error** - 连接错误

### 2. 发送消息

在输入框中输入消息，点击 ➤ 发送或按 Enter 键。

### 3. 查看事件

AG-UI 会显示以下类型的事件：

| 事件类型 | 图标 | 说明 |
|----------|------|------|
| `message` | 💬 | AI 助手的文本消息 |
| `tool_call` | 🔧 | AI 调用工具 |
| `state_update` | 📊 | 状态更新 |
| `human_in_the_loop` | 👤 | 请求用户输入/确认 |

---

## 架构说明

### 通信模式

```
┌─────────────┐                   ┌─────────────┐
│   GoTTY     │                   │   AG-UI     │
│   Client    │                   │   Server    │
│  (Frontend) │                   │   (Agent)   │
└──────┬──────┘                   └──────┬──────┘
       │                                   │
       │  1. POST /api/agui/chat          │
       ├──────────────────────────────────>│
       │  { content: "Hello" }             │
       │                                   │
       │  2. SSE Stream (text/event-stream)│
       │<──────────────────────────────────┤
       │  event: message                   │
       │  data: { content: "Hi!" }         │
       │                                   │
       │  event: tool_call                 │
       │  data: { name: "search", args: {}}│
```

### AG-UI 客户端 API

前端已内置 `AGUIClient` 类，可通过 `window.AGUIClient` 访问。

**核心方法**:

```javascript
// 连接到 SSE 端点
await aguiClient.connect(endpoint);

// 断开连接
aguiClient.disconnect();

// 发送消息
await aguiClient.sendMessage("Hello, AI!");

// 发送工具结果
await aguiClient.sendToolResult(callId, { result: "success" });

// 发送人在回路响应
await aguiClient.sendHumanResponse(requestId, { confirm: true });

// 更新共享状态
await aguiClient.setState("theme", "dark");
```

**事件监听**:

```javascript
const aguiClient = new window.AGUIClient({
  endpoint: '/api/agui',
  onEvent: (event) => {
    console.log('AG-UI Event:', event);
    // 根据 event.type 处理不同事件
  },
  onStatusChange: (status) => {
    console.log('Connection status:', status);
  },
  onError: (error) => {
    console.error('AG-UI Error:', error);
  }
});
```

---

## 服务端实现示例

### Go 后端 SSE 端点

```go
package main

import (
    "encoding/json"
    "fmt"
    "net/http"
    "time"
)

// AGUIEvent 定义 AG-UI 事件类型
type AGUIEvent struct {
    Type      string      `json:"type"`
    ID        string      `json:"id,omitempty"`
    Role      string      `json:"role,omitempty"`
    Content   string      `json:"content,omitempty"`
    Timestamp int64       `json:"timestamp"`
    Name      string      `json:"name,omitempty"`
    Args      interface{} `json:"args,omitempty"`
    Result    interface{} `json:"result,omitempty"`
    Key       string      `json:"key,omitempty"`
    Value     interface{} `json:"value,omitempty"`
    Operation string      `json:"operation,omitempty"`
    Message   string      `json:"message,omitempty"`
    RequestType string    `json:"requestType,omitempty"`
}

// handleAGUI 处理 AG-UI SSE 连接
func handleAGUI(w http.ResponseWriter, r *http.Request) {
    // 设置 SSE 响应头
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    w.Header().Set("Access-Control-Allow-Origin", "*")

    flusher, ok := w.(http.Flusher)
    if !ok {
        http.Error(w, "Streaming not supported", http.StatusInternalServerError)
        return
    }

    // 获取认证 token（如果需要）
    token := r.URL.Query().Get("token")
    if !validateToken(token) {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    // 保持连接
    <-r.Context().Done()
}

// sendEvent 发送 SSE 事件
func sendEvent(w http.ResponseWriter, flusher http.Flusher, event AGUIEvent) {
    data, _ := json.Marshal(event)
    fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, data)
    flusher.Flush()
}

// 示例：发送消息
func sendExampleMessages(w http.ResponseWriter, flusher http.Flusher) {
    // 欢迎消息
    sendEvent(w, flusher, AGUIEvent{
        Type:      "message",
        ID:        "msg_001",
        Role:      "assistant",
        Content:   "你好！我是你的 AI 助手，有什么可以帮你的吗？",
        Timestamp: time.Now().UnixMilli(),
    })

    // 工具调用示例
    sendEvent(w, flusher, AGUIEvent{
        Type:      "tool_call",
        ID:        "call_001",
        Name:      "get_weather",
        Args:      map[string]string{"location": "北京"},
        Timestamp: time.Now().UnixMilli(),
    })

    // 状态更新示例
    sendEvent(w, flusher, AGUIEvent{
        Type:      "state_update",
        Key:       "isProcessing",
        Value:     true,
        Operation: "set",
        Timestamp: time.Now().UnixMilli(),
    })

    // 人在回路请求示例
    sendEvent(w, flusher, AGUIEvent{
        Type:        "human_in_the_loop",
        ID:          "hitl_001",
        RequestType: "confirm",
        Message:     "确定要执行此操作吗？",
        Timestamp:   time.Now().UnixMilli(),
    })
}

func validateToken(token string) bool {
    // 实现 token 验证逻辑
    return token != ""
}
```

### Python FastAPI 示例

```python
from fastapi import FastAPI, Request
from fastapi.responses import StreamingResponse
import json
import time
import asyncio

app = FastAPI()

@app.get("/api/agui")
async def agui_endpoint(request: Request):
    """AG-UI SSE 端点"""

    async def event_generator():
        # 欢迎消息
        yield format_sse("message", {
            "id": "msg_001",
            "role": "assistant",
            "content": "你好！我是你的 AI 助手",
            "timestamp": int(time.time() * 1000)
        })

        # 工具调用
        yield format_sse("tool_call", {
            "id": "call_001",
            "name": "get_weather",
            "args": {"location": "北京"},
            "timestamp": int(time.time() * 1000)
        })

        # 保持连接
        while not request.is_disconnected():
            await asyncio.sleep(30)
            yield ": ping\n\n"

    return StreamingResponse(
        event_generator(),
        media_type="text/event-stream",
        headers={
            "Cache-Control": "no-cache",
            "Connection": "keep-alive",
        }
    )

@app.post("/api/agui/chat")
async def agui_chat(request: Request):
    """接收用户消息"""
    body = await request.json()
    content = body.get("content")
    print(f"收到用户消息：{content}")
    return {"status": "ok"}

@app.post("/api/agui/tool_result")
async def agui_tool_result(request: Request):
    """接收工具执行结果"""
    body = await request.json()
    call_id = body.get("callId")
    result = body.get("result")
    print(f"工具 {call_id} 结果：{result}")
    return {"status": "ok"}

@app.post("/api/agui/human_response")
async def agui_human_response(request: Request):
    """接收人在回路响应"""
    body = await request.json()
    request_id = body.get("requestId")
    data = body.get("data")
    print(f"HITL {request_id} 响应：{data}")
    return {"status": "ok"}

@app.post("/api/agui/state")
async def agui_state(request: Request):
    """更新共享状态"""
    body = await request.json()
    key = body.get("key")
    value = body.get("value")
    print(f"状态更新 {key} = {value}")
    return {"status": "ok"}

def format_sse(event_type: str, data: dict) -> str:
    """格式化 SSE 事件"""
    return f"event: {event_type}\ndata: {json.dumps(data)}\n\n"
```

---

## 事件类型详解

### 1. Message 事件

AI 助手发送的文本消息。

```json
{
  "type": "message",
  "id": "msg_001",
  "role": "assistant",
  "content": "你好！有什么可以帮你的？",
  "timestamp": 1710123456789
}
```

### 2. Tool Call 事件

AI 请求前端执行工具。

```json
{
  "type": "tool_call",
  "id": "call_001",
  "name": "get_weather",
  "args": {
    "location": "北京",
    "unit": "celsius"
  },
  "timestamp": 1710123456789
}
```

**前端响应**:
```javascript
await aguiClient.sendToolResult("call_001", {
  "temperature": 25,
  "condition": "晴朗"
});
```

### 3. State Update 事件

更新共享状态。

```json
{
  "type": "state_update",
  "key": "user.preferences",
  "value": {
    "theme": "dark",
    "notifications": true
  },
  "operation": "set"  // "set" | "delete" | "merge"
}
```

### 4. Human-in-the-Loop 事件

请求用户输入或确认。

```json
{
  "type": "human_in_the_loop",
  "id": "hitl_001",
  "requestType": "confirm",  // "confirm" | "input" | "select"
  "message": "确定要删除此文件吗？",
  "schema": {
    "type": "object",
    "properties": {
      "confirm": { "type": "boolean" }
    }
  }
}
```

**前端响应**:
```javascript
await aguiClient.sendHumanResponse("hitl_001", {
  "confirm": true
});
```

---

## 完整示例：天气查询助手

### 后端 (Python)

```python
@app.get("/api/agui")
async def weather_agent(request: Request):
    """天气查询 Agent"""

    async def event_generator():
        # 1. 欢迎消息
        yield format_sse("message", {
            "type": "message",
            "id": "msg_001",
            "role": "assistant",
            "content": "你好！我可以帮你查询天气。请告诉我城市名称。",
            "timestamp": int(time.time() * 1000)
        })

        # 2. 等待用户输入（通过 REST API）
        # 用户消息会通过 POST /api/agui/chat 发送

        # 3. 收到城市后，调用前端工具
        yield format_sse("tool_call", {
            "type": "tool_call",
            "id": "call_weather_001",
            "name": "get_weather",
            "args": {"location": "北京"},
            "timestamp": int(time.time() * 1000)
        })

        # 4. 等待工具结果（通过 REST API）

        # 5. 显示结果
        yield format_sse("message", {
            "type": "message",
            "id": "msg_002",
            "role": "assistant",
            "content": "北京当前天气：晴朗，温度 25°C",
            "timestamp": int(time.time() * 1000)
        })

    return StreamingResponse(event_generator(), media_type="text/event-stream")
```

### 前端工具注册

```javascript
// 在 GoTTY 中注册天气工具
const aguiClient = new window.AGUIClient({
  endpoint: '/api/agui',
  onEvent: async (event) => {
    if (event.type === 'tool_call' && event.name === 'get_weather') {
      // 执行天气查询
      const weather = await fetchWeather(event.args.location);

      // 发送结果回 Agent
      await aguiClient.sendToolResult(event.id, weather);
    }
  }
});

async function fetchWeather(location) {
  // 调用天气 API
  const response = await fetch(`/api/weather?city=${location}`);
  return response.json();
}
```

---

## 调试技巧

### 1. 查看连接状态

```javascript
console.log('AG-UI Status:', aguiClient.getStatus());
// 输出："disconnected" | "connecting" | "connected" | "error"
```

### 2. 查看消息历史

```javascript
const messages = aguiClient.getMessages();
console.log('Message history:', messages);
```

### 3. 清除消息

```javascript
aguiClient.clearMessages();
```

### 4. 监听错误

```javascript
const aguiClient = new window.AGUIClient({
  onError: (error) => {
    console.error('AG-UI Error:', error.message);
  }
});
```

---

## 最佳实践

### 1. 错误处理

```javascript
try {
  await aguiClient.connect(endpoint);
} catch (error) {
  console.error('连接失败:', error);
  // 显示用户友好的错误提示
}
```

### 2. 断线重连

```javascript
let reconnectAttempts = 0;
const maxReconnects = 5;

async function connectWithRetry() {
  try {
    await aguiClient.connect(endpoint);
    reconnectAttempts = 0;
  } catch (error) {
    reconnectAttempts++;
    if (reconnectAttempts < maxReconnects) {
      setTimeout(connectWithRetry, 1000 * reconnectAttempts);
    }
  }
}
```

### 3. 工具注册

```javascript
const tools = {
  get_weather: async (args) => {
    return await fetchWeather(args.location);
  },
  search_web: async (args) => {
    return await searchWeb(args.query);
  }
};

aguiClient.on('tool_call', async (event) => {
  const handler = tools[event.name];
  if (handler) {
    try {
      const result = await handler(event.args);
      await aguiClient.sendToolResult(event.id, result);
    } catch (error) {
      await aguiClient.sendToolResult(event.id, { error: error.message });
    }
  }
});
```

---

## 参考资料

- [CopilotKit 官方文档](https://docs.copilotkit.ai)
- [AG-UI 协议介绍](https://docs.copilotkit.ai/ag-ui/introduction)
- [CopilotKit GitHub](https://github.com/CopilotKit/CopilotKit)
- [SSE (Server-Sent Events) MDN](https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events)

---

*文档结束*
