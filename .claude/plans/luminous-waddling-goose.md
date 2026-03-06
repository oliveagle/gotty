# Multi-Session Management Page Implementation Plan

## Goal
Add a multi-session management page to Gotty with:
- Left sidebar listing all active sessions
- Right panel showing the Gotty terminal interface

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    Browser (Frontend)                        │
│  ┌─────────────┐  ┌──────────────────────────────────────┐  │
│  │ Session     │  │ Terminal Panel                        │  │
│  │ Sidebar     │  │ (xterm.js + WebSocket)               │  │
│  │             │  │                                       │  │
│  │ - Session 1 │  │                                       │  │
│  │ - Session 2 │  │                                       │  │
│  │ + New       │  │                                       │  │
│  └─────────────┘  └──────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    Gotty Server (Backend)                    │
│  ┌─────────────────┐  ┌─────────────────────────────────┐   │
│  │ Session Manager │  │ WebSocket Handler               │   │
│  │                 │  │ - /sessions (list/create)       │   │
│  │ - Create        │  │ - /ws/{session_id} (terminal)   │   │
│  │ - List          │  │                                 │   │
│  │ - Join          │  │                                 │   │
│  │ - Close         │  │                                 │   │
│  └─────────────────┘  └─────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

## Implementation Steps

### Phase 1: Backend - Session Management

#### 1.1 Create Session Manager (`server/session_manager.go`)
- Store sessions in a map with unique IDs
- Provide methods: `Create()`, `List()`, `Get()`, `Close()`
- Thread-safe with mutex

#### 1.2 Add Session to Server (`server/server.go`)
- Add `SessionManager` to Server struct
- Add session API endpoints:
  - `GET /api/sessions` - list all sessions
  - `POST /api/sessions` - create new session
  - `DELETE /api/sessions/:id` - close session
  - `GET /api/sessions/:id` - get session info

#### 1.3 Modify WebSocket Handler (`server/handlers.go`)
- Add route: `/ws` with optional session ID query param
- If session ID provided: attach to existing session
- If no session ID: create new session and return ID

### Phase 2: Frontend - Session UI

#### 2.1 New Index Page Template (`resources/index.html`)
- Layout: sidebar (25%) + terminal (75%)
- Add session list in sidebar
- Add "New Session" button

#### 2.2 New CSS Styles (`resources/index.css`)
- Sidebar styling (dark theme)
- Session list item styling
- Active session highlight

#### 2.3 New JavaScript (`js/src/session_manager.ts`)
- Fetch session list from API
- Create new session
- Select session to switch terminal

#### 2.4 Modify Main JS (`js/src/main.ts`)
- Load session manager
- Render session UI
- Handle WebSocket with session ID

### Phase 3: WebSocket Protocol Enhancement

#### 3.1 Session Join Message
- Client sends: `{ "Action": "join", "SessionID": "xxx" }`
- Server validates and attaches to session

## Key Files to Modify

| File | Changes |
|------|---------|
| `server/server.go` | Add SessionManager, API routes |
| `server/handlers.go` | Modify WS handler for sessions |
| `server/session_manager.go` | **NEW** - Session tracking |
| `resources/index.html` | Add sidebar layout |
| `resources/index.css` | Add sidebar styles |
| `js/src/main.ts` | Integrate session UI |
| `js/src/session_manager.ts` | **NEW** - Frontend session logic |

## API Design

### List Sessions
```
GET /api/sessions
Response: {
  "sessions": [
    { "id": "abc123", "title": "bash", "created_at": "..." },
    { "id": "def456", "title": "vim", "created_at": "..." }
  ]
}
```

### Create Session
```
POST /api/sessions
Response: { "id": "new123", "title": "bash" }
```

### WebSocket Connection
```
ws://host:port/ws?session_id=abc123
```

## Verification

1. Run `go build` to verify backend compiles
2. Run `npm run build` (or compile TypeScript) to verify frontend
3. Start server and access http://localhost:8080
4. Create new session via button
5. Verify terminal works in right panel
6. Create multiple sessions and switch between them in sidebar

## Technical Notes

- Each session maintains its own PTY process
- Sessions are independent (can join existing or create new)
- Session data stored in memory (no persistence needed for v1)
- Frontend uses fetch API for session management
- WebSocket continues to handle terminal I/O
