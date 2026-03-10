# GoTTY 项目规范

## 构建说明

**不要手动编译 TypeScript**，使用 make 命令会自动处理：

```bash
make
```

Make 会自动执行：
1. `npm run build` - 编译 TypeScript
2. 复制 bundle 到 resources 目录
3. Go build 编译最终二进制

## 项目结构

```
gotty/
├── js/                    # TypeScript 源码
│   ├── src/
│   │   ├── xterm.ts       # xterm.js 终端实现
│   │   ├── webtty.ts      # WebSocket 通信
│   │   └── main.ts        # 入口
│   └── dist/              # 编译输出
├── resources/
│   ├── index.html         # HTML 模板
│   ├── css/               # 样式文件
│   └── js/                # 编译后的 JS bundle
└── Makefile               # 构建脚本
```

## 关键组件

- **侧边栏宽度**: 固定 210px + 1px border = 211px
- **终端 resize**: 使用 `fitWithSidebarState(isCollapsed)` 直接计算宽度

## 剪贴板功能

### 实现方式

由于 gotty 通常运行在 HTTP 环境（非 HTTPS），`navigator.clipboard` API 不可用。因此使用 `document.execCommand('copy')` 作为 fallback 方法。

**关键发现**（2026-03-09）：
- `navigator.clipboard.writeText()` 在 HTTP 页面上返回 `undefined`
- `document.execCommand('copy')` 在 HTTP 页面上正常工作

### 使用方法

1. **选择文本**：在终端中拖动鼠标选择文本
2. **右键复制**：右键点击选中的文本，自动复制到浏览器剪贴板
3. **粘贴**：在浏览器其他页面使用 `Cmd+V` / `Ctrl+V` 粘贴

### 代码位置

- `js/src/xterm.ts` - `setupClipboardOnSelection()` 右键复制逻辑
- `js/src/xterm.ts` - `copyToClipboardFallback()` 使用 `execCommand('copy')` 实现

### 技术细节

```typescript
// 使用 textarea + execCommand 实现 HTTP 环境下的复制
private copyToClipboardFallback(text: string): boolean {
    const textarea = document.createElement('textarea');
    textarea.value = text;
    textarea.style.position = 'fixed';
    textarea.style.opacity = '0';
    document.body.appendChild(textarea);
    textarea.select();
    const success = document.execCommand('copy');
    document.body.removeChild(textarea);
    return success;
}
```

## 页面交互规范

**所有页面（除了验证页面）必须使用相同的标准和交互逻辑。**

### 页面类型

| 页面 | 说明 | 显示方式 |
|------|------|----------|
| Landing Page | 登录后默认显示，展示欢迎信息和快捷操作 | 右侧区域覆盖显示 |
| Settings Page | 用户设置页面 | 右侧区域覆盖显示 |
| Terminal | 终端会话页面 | 右侧区域显示 |
| IRC Chat | IRC 聊天室 | 右侧区域覆盖显示 |

### 交互规则

1. **页面切换**: 点击任意页面按钮时，隐藏其他所有页面，只显示目标页面
2. **Home 按钮**: sidebar 工具栏的 🏠 按钮返回 Landing Page
3. **Settings 按钮**: 点击 ⚙️ 显示 Settings Page
4. **Session 点击**: 点击 session 进入 Terminal 页面
5. **IRC 点击**: 点击 IRC 频道进入 IRC Chat 页面

### 配色规范

所有页面使用统一的 CSS 变量：

```css
--page-bg: #1a1a2e;           /* 页面背景色 */
--page-bg-secondary: #2d2d44; /* 次级背景色 */
--page-border: #3a3a5a;       /* 边框颜色 */
--page-text: #e0e0e0;         /* 文字颜色 */
--page-text-muted: #888;      /* 次级文字颜色 */
--page-primary: #4a9eff;      /* 主色调 */
--page-primary-light: #6ab7ff; /* 主色调亮色 */
```

### 用户设置持久化

用户设置保存在服务器端：
- 存储位置: `~/.config/gotty/user_settings.json`
- API 端点: `GET/POST /api/user-settings`
- 同时同步到 localStorage 保持向后兼容

