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
