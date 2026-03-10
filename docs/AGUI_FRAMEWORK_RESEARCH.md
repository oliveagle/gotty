# CopilotKit & AG-UI 框架调研报告

> **调研时间**: 2026-03-10
> **项目地址**: https://github.com/CopilotKit/CopilotKit
> **官方文档**: https://docs.copilotkit.ai

---

## 目录

- [项目概述](#项目概述)
- [核心概念](#核心概念)
- [AG-UI 协议](#ag-ui-协议)
- [架构设计](#架构设计)
- [核心组件](#核心组件)
- [交互流程](#交互流程)
- [实现参考](#实现参考)

---

## 项目概述

### 什么是 CopilotKit

CopilotKit 是一个**全栈 SDK**，用于构建"代理原生应用程序"（Agent-Native Applications）。它的核心理念是：

> "Build agent-native apps with Generative UI, shared state, and human-in-the-loop workflows."

### 解决的问题

传统的 AI 应用开发面临以下挑战：
1. **UI 与 Agent 交互复杂** - 如何让 AI 代理动态控制 UI
2. **状态同步困难** - Agent 和前端如何实时共享状态
3. **人在回路缺失** - 如何在 AI 执行过程中请求用户输入
4. **流式通信复杂** - 如何处理实时的 Agent-UI 事件流

### 主要功能

| 功能 | 描述 |
|------|------|
| **聊天 UI** | 开箱即用的 React 聊天界面，支持消息流、工具调用 |
| **后端工具渲染** | 代理返回 UI，客户端直接渲染 |
| **生成式 UI** | 代理动态生成和更新 UI 组件 |
| **共享状态** | 代理与 UI 实时读写同步状态 |
| **人在回路** | 代理暂停并请求用户输入/确认 |

---

## 核心概念

### 1. AG-UI 协议

AG-UI (Agent-Graphical User Interface) 是 CopilotKit 背后的核心协议。

**定义**: 基于事件的 Server-Sent Events (SSE) 协议，连接前端与 AI 代理。

**核心思想**:
- 前端订阅 SSE 事件流
- 后端通过事件推送消息、工具调用、UI 更新等
- 双向通信（前端可通过 REST 发送消息）

### 2. Agent (智能代理)

Agent 是核心执行单元，具有以下特性：

```typescript
// 简化概念
interface Agent {
  id: string;
  name: string;
  // 工具注册
  tools: Tool[];
  // 状态管理
  state: Record<string, any>;
  // 执行逻辑
  run(): AsyncIterable<AgentEvent>;
}
```

### 3. Tool (工具)

Agent 可调用的功能单元：

```typescript
interface FrontendTool<T = any> {
  name: string;
  description?: string;
  parameters?: ZodType<T>;      // 参数 schema
  handler?: (args: T) => Promise<any>;  // 前端执行
  followUp?: boolean;           // 是否需要追问
  available?: boolean;          // 可用性开关
}
```

### 4. Shared State (共享状态)

Agent 和前端可以同时读写的状态：

```typescript
// 概念示意
interface SharedState {
  get(key: string): any;
  set(key: string, value: any): void;
  subscribe(callback: (state) => void): () => void;
}
```

### 5. Human-in-the-Loop (人在回路)

Agent 执行过程中可以暂停，请求用户输入：

```typescript
// 概念示意
interface HumanInTheLoopRequest {
  type: "request_input" | "confirm";
  message: string;
  schema?: ZodType;  // 输入验证
}
```

---

## AG-UI 协议

### 协议概述

AG-UI 是一个**事件驱动的 SSE 协议**，用于 Agent 与前端的实时通信。

### 通信架构

```
┌─────────────┐                   ┌─────────────┐
│   Frontend  │                   │   Backend   │
│  (React)    │                   │   (Agent)   │
└──────┬──────┘                   └──────┬──────┘
       │                                   │
       │  1. POST /api/chat               │
       ├──────────────────────────────────>│
       │  { messages: [...] }              │
       │                                   │
       │  2. SSE Stream (text/event-stream)│
       │<──────────────────────────────────┤
       │  event: message                   │
       │  data: { content: "..." }        │
       │                                   │
       │  event: tool_call                 │
       │  data: { name: "...", args: {} } │
       │                                   │
       │  event: state_update              │
       │  data: { key: "...", value: ... }│
       │                                   │
       │  3. (可选) POST /api/tool_response│
       ├──────────────────────────────────>│
       │  { result: "..." }                │
       │                                   │
```

### SSE 事件类型

基于源码分析，AG-UI 定义了以下事件类型：

| 事件类型 | 描述 | 方向 |
|---------|------|------|
| `message` | 文本消息 | Backend → Frontend |
| `tool_call` | Agent 调用工具 | Backend → Frontend |
| `tool_result` | 工具执行结果 | Frontend → Backend |
| `state_update` | 状态更新 | Backend → Frontend |
| `ui_component` | 生成 UI 组件 | Backend → Frontend |
| `human_in_the_loop` | 请求用户输入 | Backend → Frontend |
| `human_response` | 用户输入结果 | Frontend → Backend |

### 消息格式示例

#### 1. Message 事件

```typescript
// SSE 格式
event: message
data: {
  "id": "msg_123",
  "role": "assistant",
  "content": "Hello, how can I help you?",
  "timestamp": 1234567890
}
```

#### 2. Tool Call 事件

```typescript
event: tool_call
data: {
  "id": "call_123",
  "name": "get_weather",
  "args": {
    "location": "San Francisco",
    "unit": "celsius"
  }
}
```

#### 3. State Update 事件

```typescript
event: state_update
data: {
  "key": "user.preferences",
  "value": {
    "theme": "dark",
    "notifications": true
  },
  "operation": "set"  // "set" | "delete" | "merge"
}
```

#### 4. Human-in-the-Loop 事件

```typescript
event: human_in_the_loop
data: {
  "id": "hitl_123",
  "type": "confirm",  // "confirm" | "input" | "select"
  "message": "Are you sure you want to delete this file?",
  "schema": {
    "type": "object",
    "properties": {
      "confirm": { "type": "boolean" }
    }
  }
}
```

---

## 架构设计

### 仓库结构

```
CopilotKit/
├── packages/v2/
│   ├── core/              # 核心逻辑
│   │   ├── src/
│   │   │   ├── core/
│   │   │   │   ├── core.ts           # 主类
│   │   │   │   ├── state-manager.ts  # 状态管理
│   │   │   │   ├── context-store.ts  # 上下文管理
│   │   │   │   ├── agent-registry.ts # Agent 注册
│   │   │   │   └── suggestion-engine.ts
│   │   │   ├── agent.ts
│   │   │   ├── intelligence-agent.ts
│   │   │   └── types.ts
│   │
│   ├── react/             # React 绑定
│   │   ├── src/
│   │   │   ├── hooks/
│   │   │   │   ├── use-agent.ts
│   │   │   │   ├── use-component.ts
│   │   │   │   ├── use-frontend-tool.ts
│   │   │   │   ├── use-human-in-the-loop.ts
│   │   │   │   └── ...
│   │   │   ├── a2ui/
│   │   │   │   └── A2UIMessageRenderer.tsx
│   │   │   ├── components/
│   │   │   ├── providers/
│   │   │   └── types/
│   │
│   ├── runtime/           # 后端运行时
│   │   ├── src/
│   │   │   ├── middleware-sse-parser.ts  # SSE 解析中间件
│   │   │   ├── runtime.ts
│   │   │   ├── handler.ts
│   │   │   ├── express.ts
│   │   │   └── endpoints/
│   │
│   ├── agent/
│   ├── angular/
│   ├── shared/
│   └── voice/
│
└── examples/v2/
    ├── react/
    ├── node-express/
    ├── next-pages-router/
    └── ...
```

### 核心分层架构

```
┌─────────────────────────────────────────────────────────┐
│                     Frontend Layer                        │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐   │
│  │  React UI   │  │  Components │  │   Hooks     │   │
│  └─────────────┘  └─────────────┘  └─────────────┘   │
├─────────────────────────────────────────────────────────┤
│                   AG-UI Protocol Layer                    │
│         (SSE Events + REST API)                           │
├─────────────────────────────────────────────────────────┤
│                   Core Logic Layer                         │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐  │
│  │   Agent      │ │   State      │ │  Context     │  │
│  │  Registry    │ │  Manager     │ │   Store      │  │
│  └──────────────┘ └──────────────┘ └──────────────┘  │
├─────────────────────────────────────────────────────────┤
│                   Runtime Layer                            │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐  │
│  │  SSE Parser  │ │   Handlers   │ │  Endpoints   │  │
│  └──────────────┘ └──────────────┘ └──────────────┘  │
├─────────────────────────────────────────────────────────┤
│                   Agent Execution Layer                   │
│  (LangGraph, CrewAI, custom agents, etc.)                │
└─────────────────────────────────────────────────────────┘
```

---

## 核心组件

### 1. CopilotKitCore (core.ts)

核心管理器，协调整个系统。

```typescript
// 简化示意
class CopilotKitCore {
  // 子系统
  agentRegistry: AgentRegistry;
  stateManager: StateManager;
  contextStore: ContextStore;
  suggestionEngine: SuggestionEngine;

  // 订阅者
  subscribers: Set<CoreSubscriber>;

  // 生命周期
  constructor();
  registerAgent(agent: AbstractAgent);
  unregisterAgent(agentId: string);
  addContext(context: Context);
  setState(key: string, value: any);
  notifySubscribers(event: CoreEvent);
}
```

### 2. StateManager (state-manager.ts)

管理 Agent 和前端共享的状态。

```typescript
class StateManager {
  private _state: Record<string, any> = {};

  get state(): Readonly<Record<string, any>>;

  setState(key: string, value: any): void;
  getState(key: string): any;
  deleteState(key: string): void;
  mergeState(key: string, value: object): void;

  private async notifySubscribers(): Promise<void>;
}
```

### 3. ContextStore (context-store.ts)

管理 Agent 可用的上下文信息。

```typescript
class ContextStore {
  private _context: Record<string, Context> = {};

  get context(): Readonly<Record<string, Context>>;

  addContext({ description, value }: Context): string;
  removeContext(id: string): void;

  private async notifySubscribers(): Promise<void>;
}
```

### 4. React Hooks

#### useAgent()

```typescript
// packages/v2/react/src/hooks/use-agent.tsx

function useAgent<TState = any, TActions = any>(
  options?: UseAgentOptions
): UseAgentResult<TState, TActions> {
  // 返回:
  // - state: 当前状态
  // - setState: 更新状态
  // - send: 发送消息
  // - run: 运行 agent
  // - actions: 代理动作
}
```

#### useFrontendTool()

```typescript
function useFrontendTool<T = any>(
  tool: FrontendTool<T>
): void {
  // 注册前端工具，Agent 可以调用
}
```

#### useHumanInTheLoop()

```typescript
function useHumanInTheLoop(): HumanInTheLoopResult {
  // 处理人在回路请求
  // - pendingRequests: 待处理请求
  // - respond: 响应用户输入
}
```

### 5. SSE Middleware (middleware-sse-parser.ts)

解析 SSE 流的中间件。

```typescript
// 概念示意
function createSSEParserMiddleware() {
  return (req, res, next) => {
    // 解析 text/event-stream
    // 提取 event 和 data 字段
    // 分发给对应的处理器
  };
}
```

---

## 交互流程

### 1. 初始化流程

```
┌─────────┐                    ┌──────────────┐                    ┌─────────┐
│  User   │                    │   React      │                    │ Backend │
└────┬────┘                    └──────┬───────┘                    └────┬────┘
     │                                  │                                   │
     │  1. 打开页面                    │                                   │
     ├─────────────────────────────────>│                                   │
     │                                  │                                   │
     │                                  │  2. <CopilotKitProvider>         │
     │                                  │     初始化                         │
     │                                  │  <AgentProvider agentId="...">   │
     │                                  │                                   │
     │                                  │  3. GET /api/agents              │
     │                                  ├──────────────────────────────────>│
     │                                  │                                   │
     │                                  │  4. 200 OK { agents: [...] }    │
     │                                  │<──────────────────────────────────┤
     │                                  │                                   │
     │                                  │  5. 建立 SSE 连接                 │
     │                                  │     GET /api/events               │
     │                                  ├──────────────────────────────────>│
     │                                  │                                   │
     │                                  │  6. 保持 SSE 连接开启             │
     │                                  │<──────────────────────────────────┤
     │                                  │     (等待事件)                     │
```

### 2. 发送消息流程

```
┌─────────┐                    ┌──────────────┐                    ┌─────────┐
│  User   │                    │   React      │                    │ Backend │
└────┬────┘                    └──────┬───────┘                    └────┬────┘
     │                                  │                                   │
     │  1. 输入消息                    │                                   │
     ├─────────────────────────────────>│                                   │
     │  "What's the weather?"          │                                   │
     │                                  │                                   │
     │                                  │  2. POST /api/chat               │
     │                                  │     { messages: [...] }          │
     │                                  ├──────────────────────────────────>│
     │                                  │                                   │
     │                                  │                                   │  3. Agent 处理
     │                                  │                                   │  4. 生成响应
     │                                  │                                   │
     │                                  │  5. SSE: event: message          │
     │                                  │<──────────────────────────────────┤
     │                                  │     data: { content: "..." }     │
     │                                  │                                   │
     │                                  │  6. 更新 UI                       │
     │  7. 显示消息                     │                                   │
     │<─────────────────────────────────┤                                   │
```

### 3. 工具调用流程

```
┌─────────┐                    ┌──────────────┐                    ┌─────────┐
│  User   │                    │   React      │                    │ Backend │
└────┬────┘                    └──────┬───────┘                    └────┬────┘
     │                                  │                                   │
     │                                  │  1. SSE: event: tool_call        │
     │                                  │<──────────────────────────────────┤
     │                                  │     name: "get_weather"           │
     │                                  │                                   │
     │                                  │  2. 查找已注册的前端工具          │
     │                                  │     useFrontendTool()             │
     │                                  │                                   │
     │                                  │  3. 执行 handler()                │
     │                                  │                                   │
     │  4. (可选) 用户交互              │                                   │
     │<─────────────────────────────────┤                                   │
     │  5. 用户输入                     │                                   │
     ├─────────────────────────────────>│                                   │
     │                                  │                                   │
     │                                  │  6. handler 返回结果              │
     │                                  │                                   │
     │                                  │  7. POST /api/tool_result        │
     │                                  ├──────────────────────────────────>│
     │                                  │     { result: "72°F" }            │
     │                                  │                                   │
     │                                  │                                   │  8. Agent 继续处理
     │                                  │                                   │
     │                                  │  9. SSE: event: message          │
     │                                  │<──────────────────────────────────┤
     │                                  │                                   │
     │  10. 显示最终结果                │                                   │
     │<─────────────────────────────────┤                                   │
```

### 4. 人在回路流程

```
┌─────────┐                    ┌──────────────┐                    ┌─────────┐
│  User   │                    │   React      │                    │ Backend │
└────┬────┘                    └──────┬───────┘                    └────┬────┘
     │                                  │                                   │
     │                                  │  1. SSE: event: human_in_the_loop│
     │                                  │<──────────────────────────────────┤
     │                                  │     type: "confirm"               │
     │                                  │     message: "Delete file?"       │
     │                                  │                                   │
     │                                  │  2. 显示对话框/模态框             │
     │  3. 显示确认对话框               │                                   │
     │<─────────────────────────────────┤                                   │
     │                                  │                                   │
     │  4. 用户点击 "确认"              │                                   │
     ├─────────────────────────────────>│                                   │
     │                                  │                                   │
     │                                  │  5. POST /api/human_response     │
     │                                  ├──────────────────────────────────>│
     │                                  │     { confirm: true }             │
     │                                  │                                   │
     │                                  │                                   │  6. Agent 恢复执行
     │                                  │                                   │
     │                                  │  7. SSE: event: message          │
     │                                  │<──────────────────────────────────┤
     │                                  │                                   │
     │  8. 显示执行结果                 │                                   │
     │<─────────────────────────────────┤                                   │
```

### 5. 状态同步流程

```
┌─────────┐                    ┌──────────────┐                    ┌─────────┐
│  User   │                    │   React      │                    │ Backend │
└────┬────┘                    └──────┬───────┘                    └────┬────┘
     │                                  │                                   │
     │  1. 修改设置                    │                                   │
     ├─────────────────────────────────>│                                   │
     │  theme: "dark"                  │                                   │
     │                                  │                                   │
     │                                  │  2. setState("theme", "dark")    │
     │                                  │                                   │
     │                                  │  3. POST /api/state              │
     │                                  ├──────────────────────────────────>│
     │                                  │     { key: "theme", value: "dark"}│
     │                                  │                                   │
     │                                  │                                   │  4. 更新后端状态
     │                                  │                                   │
     │                                  │  5. SSE: event: state_update     │
     │                                  │<──────────────────────────────────┤
     │                                  │     (广播给所有连接的客户端)       │
     │                                  │                                   │
     │                                  │  6. 所有客户端更新本地 state      │
     │  7. UI 反映新主题                │                                   │
     │<─────────────────────────────────┤                                   │
```

---

## 实现参考

### 前端集成示例

#### 1. 基础设置

```tsx
// app.tsx
import { CopilotKitProvider } from "@copilotkit/react";
import { AgentProvider } from "@copilotkit/react";

function App() {
  return (
    <CopilotKitProvider
      runtimeUrl="/api/copilotkit"
      publicApiKey="..."  // 可选
    >
      <AgentProvider agentId="my-agent">
        <MyApp />
      </AgentProvider>
    </CopilotKitProvider>
  );
}
```

#### 2. 使用 useAgent

```tsx
// chat.tsx
import { useAgent } from "@copilotkit/react";

function Chat() {
  const { state, setState, send, run } = useAgent({
    agentId: "my-agent",
    initialState: {
      theme: "light",
      messages: []
    }
  });

  const handleSendMessage = async (text: string) => {
    await send({
      role: "user",
      content: text
    });
  };

  return (
    <div>
      <div>Theme: {state.theme}</div>
      <button onClick={() => setState("theme", "dark")}>
        Dark Mode
      </button>
      <MessageList messages={state.messages} />
      <MessageInput onSend={handleSendMessage} />
    </div>
  );
}
```

#### 3. 注册前端工具

```tsx
// tools.tsx
import { useFrontendTool } from "@copilotkit/react";
import { z } from "zod";

function WeatherTool() {
  useFrontendTool({
    name: "get_weather",
    description: "Get current weather",
    parameters: z.object({
      location: z.string().describe("City name"),
      unit: z.enum(["celsius", "fahrenheit"]).optional()
    }),
    handler: async ({ location, unit }) => {
      const response = await fetch(`/api/weather?location=${location}`);
      return response.json();
    },
    followUp: true  // 执行后让 Agent 继续
  });

  return null;
}
```

#### 4. 人在回路处理

```tsx
// human-in-the-loop.tsx
import { useHumanInTheLoop } from "@copilotkit/react";

function HumanInTheLoopModal() {
  const { pendingRequests, respond, reject } = useHumanInTheLoop();

  if (pendingRequests.length === 0) {
    return null;
  }

  const request = pendingRequests[0];

  return (
    <div className="modal">
      <h3>{request.message}</h3>

      {request.type === "confirm" && (
        <>
          <button onClick={() => respond(request.id, { confirm: true })}>
            Confirm
          </button>
          <button onClick={() => respond(request.id, { confirm: false })}>
            Cancel
          </button>
        </>
      )}

      {request.type === "input" && (
        <form onSubmit={(e) => {
          e.preventDefault();
          const value = (e.target as any).input.value;
          respond(request.id, { value });
        }}>
          <input name="input" />
          <button type="submit">Submit</button>
        </form>
      )}
    </div>
  );
}
```

### 后端集成示例 (Express)

```typescript
// backend.ts
import express from "express";
import { CopilotRuntime, copilotkitMiddleware } from "@copilotkit/runtime";

const app = express();

// 1. 添加中间件
app.use(copilotkitMiddleware());

// 2. 创建 Agent
const agent = {
  id: "my-agent",
  name: "My Assistant",

  async *run({ messages, state, context }) {
    // 发送消息
    yield {
      type: "message",
      content: "Let me check that for you..."
    };

    // 更新状态
    yield {
      type: "state_update",
      key: "is_processing",
      value: true
    };

    // 调用工具 (前端执行)
    yield {
      type: "tool_call",
      name: "get_weather",
      args: { location: "San Francisco" }
    };

    // 请求用户确认
    yield {
      type: "human_in_the_loop",
      request: {
        type: "confirm",
        message: "Continue with this action?"
      }
    };

    // 发送最终消息
    yield {
      type: "message",
      content: "Done!"
    };
  }
};

// 3. 配置运行时
const runtime = new CopilotRuntime({
  agents: [agent]
});

// 4. 挂载端点
app.use("/api/copilotkit", runtime.express());

app.listen(3000);
```

---

## 关键技术点总结

### 1. SSE vs WebSocket

CopilotKit 选择 **SSE (Server-Sent Events)** 而非 WebSocket：

| 特性 | SSE | WebSocket |
|------|-----|-----------|
| 方向 | 服务器 → 客户端 (单向) | 双向 |
| 协议 | HTTP/HTTPS | 自定义协议 |
| 自动重连 | 原生支持 | 需要手动实现 |
| 二进制 | 仅文本 | 支持二进制 |
| 防火墙 | 友好 | 可能被阻止 |

**CopilotKit 的方案**:
- SSE 用于服务器推送事件
- REST API 用于客户端发送消息
- 结合两者的优点

### 2. 状态管理设计

```typescript
// 共享状态的核心思想
interface ReactiveState {
  // 读
  get(key: string): any;

  // 写 (触发事件)
  set(key: string, value: any): void;

  // 订阅变化
  subscribe(callback: (state) => void): () => void;
}
```

### 3. 工具注册机制

- **前端工具**: 通过 `useFrontendTool()` 注册，Agent 可调用
- **后端工具**: 在 Agent 端定义，后端执行
- **混合模式**: 前端工具执行后可触发 Agent 继续

### 4. 扩展性

CopilotKit 支持多种代理框架集成：
- LangGraph
- CrewAI
- Custom Agents
- OpenAI Assistants API

---

## 参考资源

| 资源 | 链接 |
|------|------|
| GitHub 仓库 | https://github.com/CopilotKit/CopilotKit |
| 官方文档 | https://docs.copilotkit.ai |
| AG-UI 介绍 | https://docs.copilotkit.ai/ag-ui/introduction |
| NPM 包 | @copilotkit/react, @copilotkit/runtime, @copilotkit/core |

---

*文档结束*
