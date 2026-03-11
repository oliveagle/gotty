# AG-UI 快速测试指南

## 功能概述

已实现的 AG-UI 功能包括：

### 服务端 (Go)
- ✅ SSE 流式端点 (`/api/agui`)
- ✅ 聊天消息接收 (`/api/agui/chat`)
- ✅ 工具结果接收 (`/api/agui/tool_result`)
- ✅ 人在回路响应 (`/api/agui/human_response`)
- ✅ 状态更新 (`/api/agui/state`)
- ✅ WebAuthn 认证集成
- ✅ 演示模式自动发送示例事件

### 前端 (TypeScript)
- ✅ AGUIClient 类 (SSE 连接 + REST API)
- ✅ 连接状态指示器
- ✅ 消息渲染 (message, tool_call, state_update)
- ✅ Human-in-the-Loop UI (确认/输入/选择)
- ✅ 发送消息功能
- ✅ 工具结果发送
- ✅ HITL 响应发送

---

## 快速测试步骤

### 1. 启动 GoTTY

```bash
cd /Users/oliveagle/ole/repos/github.com/oliveagle/gotty-iframe-preview
./gotty --port 8080 zellij
```

### 2. 访问 GoTTY

打开浏览器访问：http://localhost:8080

### 3. 测试 AG-UI 功能

1. **打开右侧面板**
   - 点击右侧面板按钮（右侧边栏）
   - 选择 **AG UI** 标签页

2. **连接到演示端点**
   - 端点 URL：`/api/agui` (使用内置端点)
   - 点击 🔌 连接按钮
   - 状态应变为 🟢 **Connected**

3. **查看演示事件**

   连接后会自动发送以下事件：

   | 时间 | 事件类型 | 内容 |
   |------|----------|------|
   | 0.5s | `message` | 👋 欢迎消息 |
   | 1.3s | `tool_call` | 🔧 调用 get_weather(北京) |
   | 1.8s | `state_update` | 📊 状态：查询天气中... |
   | 2.8s | `message` | 📊 北京天气：晴朗，25°C |

4. **测试 Human-in-the-Loop**

   创建一个 Python 测试脚本发送 HITL 请求：

   ```python
   import http.client
   import json

   # 发送 HITL 请求到 SSE 端点（需要单独的 Agent 服务）
   # 这部分需要一个真实的 Agent 后端
   ```

---

## Python 测试 Agent 示例

创建一个简单的测试 Agent：

```python
# test_agui_agent.py
import asyncio
import json
from datetime import datetime
from aiohttp import web

async def agui_sse_handler(request):
    """AG-UI SSE 端点"""

    async def event_generator():
        # 欢迎消息
        yield format_sse("message", {
            "id": "msg_001",
            "role": "assistant",
            "content": "你好！我是测试 AI 助手。",
            "timestamp": int(datetime.now().timestamp() * 1000)
        })

        await asyncio.sleep(1)

        # 工具调用
        yield format_sse("tool_call", {
            "id": "call_001",
            "name": "get_weather",
            "args": {"location": "上海"},
            "timestamp": int(datetime.now().timestamp() * 1000)
        })

        await asyncio.sleep(1)

        # 人在回路请求
        yield format_sse("human_in_the_loop", {
            "id": "hitl_001",
            "requestType": "confirm",
            "message": "确定要继续吗？",
            "timestamp": int(datetime.now().timestamp() * 1000)
        })

        # 保持连接
        while True:
            await asyncio.sleep(30)
            yield ": ping\n\n"

    return web.Response(
        text=event_generator(),
        content_type='text/event-stream',
        headers={
            'Cache-Control': 'no-cache',
            'Connection': 'keep-alive',
        }
    )

def format_sse(event_type: str, data: dict) -> str:
    return f"event: {event_type}\ndata: {json.dumps(data)}\n\n"

app = web.Application()
app.router.add_get('/api/agui', agui_sse_handler)

if __name__ == '__main__':
    web.run_app(app, port=8000)
```

运行测试 Agent：

```bash
pip install aiohttp
python test_agui_agent.py
```

然后在 GoTTY 中连接：`http://localhost:8000/api/agui`

---

## 预期效果

### 连接成功
- 状态指示器变为绿色
- 显示 "Connected"
- 开始接收消息

### 消息显示
- **user 消息**: 右侧气泡，蓝色背景
- **assistant 消息**: 左侧气泡，绿色背景
- **system 消息**: 灰色背景
- **tool_call**: 带工具图标和参数
- **state_update**: 带状态图标和值
- **human_in_the_loop**: 带确认/取消按钮

### 交互功能
- 点击确认/取消按钮会发送响应
- 输入框可以输入文本并提交
- 下拉选择可以选择选项

---

## 调试技巧

### 浏览器控制台

打开浏览器开发者工具 (F12)，查看：

```javascript
// 检查 AGUIClient 状态
window.aguiClient?.getStatus()

// 查看消息历史
window.aguiClient?.getMessages()

// 手动发送消息
window.aguiClient?.sendMessage("Hello")

// 监听事件
window.aguiClient?.options.onEvent = (event) => {
    console.log('Event:', event)
}
```

### 服务端日志

GoTTY 会输出 AG-UI 相关日志：

```
[AG-UI] New SSE connection from 127.0.0.1:xxxx
[AG-UI] Received chat message: Hello
[AG-UI] Received tool result for call_001: {...}
[AG-UI] Received HITL response for hitl_001: {...}
```

---

## 已知限制

1. **演示模式**: 当前服务端发送的是预设的演示事件
2. **真实 Agent**: 需要连接外部的 AI Agent 服务（如 LangGraph、CopilotKit 等）
3. **工具注册**: 前端工具需要手动注册和处理

---

## 下一步

要集成真实的 AI Agent，参考：

1. [CopilotKit 文档](https://docs.copilotkit.ai)
2. [LangGraph](https://langchain-ai.github.io/langgraph/)
3. [AG-UI 使用指南](./AGUI_USAGE_GUIDE.md)

---

*测试指南创建时间：2026-03-11*
