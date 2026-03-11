# GoTTY UI 样式规范

> **创建时间**: 2026-03-11
> **最后更新**: 2026-03-11
> **版本**: v1.0

本文档定义了 GoTTY Web 界面的统一样式规范，确保所有组件的视觉一致性和可维护性。

---

## 目录

1. [颜色系统](#1-颜色系统)
2. [布局与尺寸](#2-布局与尺寸)
3. [组件规范](#3-组件规范)
4. [交互与动画](#4-交互与动画)
5. [字体与排版](#5-字体与排版)
6. [辅助功能](#6-辅助功能)

---

## 1. 颜色系统

### 1.1 全局 CSS 变量

所有颜色使用 CSS 变量定义，便于主题切换和统一维护。

#### 工作区主题色

```css
:root {
    --ws-primary: #4a9eff;        /* 主色调 (蓝色) */
    --ws-primary-light: #6ab7ff;  /* 主色调浅色 */
    --ws-primary-dark: #3a8eef;   /* 主色调深色 */
    --ws-bg-gradient: rgba(74, 158, 255, 0.15);  /* 背景渐变 */
    --ws-border: rgba(74, 158, 255, 0.3);        /* 边框色 */
    --ws-glow: rgba(74, 158, 255, 0.4);          /* 发光效果 */
}
```

#### 主题色板

| 主题名 | Primary | Primary Light | Primary Dark |
|--------|---------|---------------|--------------|
| blue | `#4a9eff` | `#6ab7ff` | `#3a8eef` |
| green | `#4ade80` | `#6ee7a0` | `#3ac870` |
| purple | `#a855f7` | `#c084fc` | `#9333ea` |
| orange | `#f59e0b` | `#fbbf24` | `#d97706` |
| red | `#ef4444` | `#f87171` | `#dc2626` |
| cyan | `#06b6d4` | `#22d3ee` | `#0891b2` |
| pink | `#ec4899` | `#f472b6` | `#db2777` |
| gray | `#6b7280` | `#9ca3af` | `#4b5563` |

#### 页面级颜色变量

用于 Landing Page 和 Settings Page 等独立页面：

```css
#landing-page,
#settings-page {
    --page-bg: #1a1a2e;           /* 页面主背景 */
    --page-bg-secondary: #2d2d44; /* 次级背景 */
    --page-border: #3a3a5a;       /* 边框颜色 */
    --page-text: #e0e0e0;         /* 主文字颜色 */
    --page-text-muted: #888;      /* 次级文字 */
    --page-primary: #4a9eff;      /* 主色调 */
    --page-primary-light: #6ab7ff; /* 主色调浅色 */
}
```

#### 中性色板

| 用途 | 颜色值 | 使用场景 |
|------|--------|----------|
| 背景深色 | `#0f0f1a` | 终端容器、文件预览 |
| 背景默认 | `#1a1a2e` | 页面主背景 |
| 背景次级 | `#252538` | 工具栏、面板头 |
| 背景强调 | `#2a2a3e` | 工具栏背景 |
| 背景组件 | `#2d2d2d` | 侧边栏、模态框 |
| 背景悬停 | `#3a3a3a` | 列表项悬停 |
| 边框深色 | `#1a1a1a` | 侧边栏边框 |
| 边框默认 | `#3a3a5a` | 组件边框 |
| 边框强调 | `#444` | 模态框边框 |
| 文字主色 | `#e0e0e0` | 主文字 |
| 文字次级 | `#888` | 说明文字、图标 |
| 文字禁用 | `#555` | 禁用状态 |

### 1.2 状态色

| 状态 | 颜色 | 使用场景 |
|------|------|----------|
| 成功/激活 | `#4ade80` | 活动状态、成功提示 |
| 警告 | `#f59e0b` | 警告提示 |
| 错误/危险 | `#ef4444` | 错误提示、删除操作 |
| 信息 | `#4a9eff` | 信息提示、链接 |

---

## 2. 布局与尺寸

### 2.1 主要布局区域

| 区域 | 宽度 | 说明 |
|------|------|------|
| 侧边栏 | `200px` | 固定宽度，可折叠 |
| 主内容区 | `flex: 1` | 自适应剩余空间 |
| 右面板 | `240px` | 基础宽度，可调整 (200px-450px) |

### 2.2 间距规范

使用统一的间距尺度，基于 4px 倍数：

| 尺寸 | 值 | 使用场景 |
|------|-----|----------|
| xs | `4px` | 极小间距 |
| sm | `6px` | 小组件内部 |
| md | `8px` | 标准组件间距 |
| lg | `10px` | 组件间间距 |
| xl | `12px` | 大组件内部 |
| 2xl | `16px` | 区域间距 |
| 3xl | `20px` | 大区域间距 |
| 4xl | `24px` | 模块间距 |

### 2.3 组件尺寸

#### 按钮尺寸

| 按钮类型 | 宽度 | 高度 | 内边距 | 字体大小 |
|----------|------|------|--------|----------|
| 工具栏按钮 | `28px` | `24-28px` | `4px` | `12-14px` |
| 图标按钮 | `22-32px` | `22-32px` | `4-8px` | `11-18px` |
| 主按钮 | auto | `auto` | `10px 20px` | `14-16px` |
| 返回按钮 | auto | `26px` | `5px 10px` | `12px` |

#### 输入框尺寸

| 类型 | 高度 | 内边距 | 字体大小 |
|------|------|--------|----------|
| 工具栏输入 | `28px` | `6px 10px` | `12px` |
| 标准输入 | auto | `10px 14px` | `14px` |
| 大输入 | auto | `14px 16px` | `16px` |

#### 列表项尺寸

| 列表类型 | 内边距 | 字体大小 | 间距 |
|----------|--------|----------|------|
| Session 列表 | `2px 6px` | `13px` | `1px` |
| 文件列表 | `8px 16px` | `13px` | - |
| IRC 频道 | `2px 6px` | `13px` | `1px` |
| 工作区下拉 | `10px 12px` | `13px` | `1px` |

### 2.4 圆角规范

| 圆角大小 | 值 | 使用场景 |
|----------|-----|----------|
| sm | `3px` | 小按钮、徽章 |
| md | `4px` | 标准按钮、输入框 |
| lg | `6px` | 模态框、下拉菜单 |
| xl | `8px` | 卡片、大按钮 |
| 2xl | `12px` | 大卡片 |
| 3xl | `16px` | 欢迎卡片 |

---

## 3. 组件规范

### 3.1 侧边栏 (Sidebar)

```css
#sidebar {
    width: 200px;
    background: #2d2d2d;
    border-right: 1px solid #1a1a1a;
    display: flex;
    flex-direction: column;
    transition: width 0.2s ease, opacity 0.2s ease, transform 0.2s ease;
}
```

**结构组成**:
- `.sidebar-header` - 头部 (工作区信息、时间显示)
- `#session-list` - Session 列表
- `#irc-channels-section` - IRC 频道区 (可选)
- `.sidebar-toolbar` - 工具栏
- `.sidebar-footer` - 底部信息

### 3.2 右面板 (Right Panel)

```css
.right-panel {
    width: 240px;
    min-width: 200px;
    max-width: 450px;
    background: #1e1e2e;
    border-left: 1px solid #3a3a5a;
    transition: width 0.3s ease;
}
```

**结构组成**:
- `.right-panel-header` - 头部 (标题、操作按钮)
- `.right-panel-tabs` - 标签页切换
- `.right-panel-toolbar` - 工具栏 (`padding: 0px 8px`)
- `.right-panel-content` - 内容区

**高度一致性规则**:
```css
.right-panel-tabs {
    padding: 0 8px;
}

.right-panel-tab {
    padding: 6px 10px;
}

.right-panel-toolbar {
    padding: 0px 8px;  /* 与 tabs 高度一致 */
}
```

### 3.3 按钮类型

#### 工具栏按钮

```css
.btn-toolbar {
    width: 28px;
    height: 28px;
    background: transparent;
    border: 1px solid #555;
    color: #888;
    border-radius: 4px;
    transition: all 0.2s;
}

.btn-toolbar:hover {
    color: var(--ws-primary);
    border-color: var(--ws-primary);
    background: var(--ws-bg-gradient);
}
```

#### 主按钮

```css
.btn-primary {
    background: linear-gradient(135deg, var(--ws-primary), var(--ws-primary-dark));
    color: #fff;
    border: none;
    border-radius: 8px;
    padding: 10px 20px;
    font-size: 14px;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.2s;
}

.btn-primary:hover {
    transform: translateY(-1px);
    box-shadow: 0 4px 12px var(--ws-glow);
}
```

#### 次级按钮

```css
.btn-secondary {
    background: #3a3a5a;
    color: #e0e0e0;
    border: 1px solid #4a4a6a;
    border-radius: 8px;
    padding: 10px 20px;
    font-size: 14px;
    cursor: pointer;
    transition: all 0.2s;
}

.btn-secondary:hover {
    background: #4a4a6a;
    border-color: #5a5a7a;
}
```

### 3.4 输入框

```css
.setting-input,
.file-source-selector,
.right-panel-file-path {
    background: #1a1a2e;
    border: 1px solid #3a3a5a;
    border-radius: 4-8px;
    color: #e0e0e0;
    padding: 6-14px;
    font-size: 12-14px;
    outline: none;
    transition: border-color 0.2s;
}

.setting-input:focus {
    border-color: var(--ws-primary);
}
```

### 3.5 列表项

#### Session 列表项

```css
.session-item {
    padding: 2px 6px;
    margin-bottom: 1px;
    background: #3a3a3a;
    border-radius: 4px;
    cursor: pointer;
    border: 1px solid transparent;
    transition: background 0.15s ease;
}

.session-item:hover {
    background: #404040;
}

.session-item.active {
    background: linear-gradient(135deg, var(--ws-primary), var(--ws-primary-dark));
    border-color: var(--ws-primary-light);
    box-shadow: 0 0 8px var(--ws-glow);
}
```

#### 文件夹项

```css
.session-item.folder {
    background: #3a3a3a;
    border-left: 3px solid transparent;
}

/* 文件夹颜色变体 */
.session-item.folder.folder-color-0 {
    border-left-color: #4a9eff;
    background: linear-gradient(90deg, rgba(74, 158, 255, 0.15) 0%, #3a3a3a 100%);
}
```

### 3.6 下拉菜单

```css
.workspace-dropdown,
.file-source-dropdown {
    background: #2d2d2d;
    border: 1px solid #444;
    border-radius: 6px;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.4);
    z-index: 10000;
    max-height: 300px;
    overflow-y: auto;
}

.dropdown-item {
    padding: 10px 12px;
    cursor: pointer;
    transition: background 0.15s;
    border-bottom: 1px solid #333;
}

.dropdown-item:hover {
    background: #3a3a3a;
}

.dropdown-item.active {
    background: var(--ws-bg-gradient);
}
```

### 3.7 模态框

```css
.confirm-dialog-content,
.workspace-modal-content {
    background: #2d2d2d;
    border: 1px solid #444;
    border-radius: 8px;
    padding: 20px;
    min-width: 280px;
    max-width: 400px;
    box-shadow: 0 4px 20px rgba(0, 0, 0, 0.5);
}
```

---

## 4. 交互与动画

### 4.1 过渡效果

| 属性 | 值 | 使用场景 |
|------|-----|----------|
| 默认过渡 | `0.2s ease` | 大部分交互 |
| 快速过渡 | `0.15s ease` | 列表项悬停 |
| 慢速过渡 | `0.3s ease` | 面板尺寸变化 |
| 渐变过渡 | `1s cubic-bezier(0.4, 0, 0.2, 1)` | 天空背景 |

### 4.2 悬停效果

```css
/* 标准悬停 */
.component:hover {
    background: #404040;
    transition: background 0.15s ease;
}

/* 主题色悬停 */
.component:hover {
    color: var(--ws-primary);
    background: var(--ws-bg-gradient);
}

/* 卡片悬停 */
.action-card:hover {
    transform: translateY(-2px);
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.3);
}
```

### 4.3 激活状态

```css
.component.active {
    background: linear-gradient(135deg, var(--ws-primary), var(--ws-primary-dark));
    border-color: var(--ws-primary-light);
    box-shadow: 0 0 8px var(--ws-glow);
}
```

### 4.4 动画关键帧

```css
/* 淡入 */
@keyframes fadeIn {
    from { opacity: 0; }
    to { opacity: 1; }
}

/* 上滑 */
@keyframes slideUp {
    from { opacity: 0; transform: translateY(20px); }
    to { opacity: 1; transform: translateY(0); }
}

/* 脉冲发光 */
@keyframes pulse-glow {
    0%, 100% {
        transform: scale(1);
        opacity: 1;
        box-shadow: 0 0 4px #4ade80;
    }
    50% {
        transform: scale(1.3);
        opacity: 0.8;
        box-shadow: 0 0 8px #4ade80;
    }
}

/* 打字指示器 */
@keyframes typing-bounce {
    0%, 60%, 100% {
        transform: translateY(0);
        opacity: 0.5;
    }
    30% {
        transform: translateY(-8px);
        opacity: 1;
    }
}
```

---

## 5. 字体与排版

### 5.1 字体族

```css
/* 系统字体栈 */
font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto,
             "Helvetica Neue", Arial, sans-serif;

/* 等宽字体 (代码、终端) */
font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;

/* IRC 聊天字体 */
font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
```

### 5.2 字体大小

| 大小 | 值 | 使用场景 |
|------|-----|----------|
| xs | `10px` | 极小文字、提示 |
| sm | `11-12px` | 次级信息、标签 |
| md | `13-14px` | 正文、列表项 |
| lg | `15-16px` | 标题、强调文字 |
| xl | `18-20px` | 大标题 |
| 2xl | `24px` | 欢迎标题 |
| 3xl | `28px` | 问候语 |
| 4xl | `72px` | 时间显示 |

### 5.3 字体粗细

| 粗细 | 值 | 使用场景 |
|------|-----|----------|
| 细体 | `200` | 时间显示 |
| 常规 | `400` | 正文 |
| 中等 | `500` | 强调文字 |
| 半粗 | `600` | 标题、标签 |
| 粗体 | `700+` | 重要强调 |

### 5.4 行高

| 类型 | 值 | 使用场景 |
|------|-----|----------|
| 紧凑 | `1.1` | 多行文本截断 |
| 标准 | `1.5-1.6` | 正文 |
| 宽松 | `1.25` | 标题 |

### 5.5 字间距

```css
/* 大写字母间距 */
text-transform: uppercase;
letter-spacing: 0.5px;  /* 标准 */
letter-spacing: 1px;    /* 宽松 */

/* 时间显示 */
letter-spacing: -2px;   /* 紧凑 */
letter-spacing: 0.5px;  /* 标准 */
```

---

## 6. 辅助功能

### 6.1 颜色对比度

所有文字与背景的对比度必须符合 WCAG AA 标准：

| 文字大小 | 最小对比度 |
|----------|------------|
| 正常文字 (<18px) | 4.5:1 |
| 大文字 (≥18px) | 3:1 |

### 6.2 焦点状态

所有可交互元素必须有清晰的焦点状态：

```css
button:focus,
input:focus,
select:focus {
    outline: 2px solid var(--ws-primary);
    outline-offset: 2px;
}
```

### 6.3 过渡动画时长

动画时长应适中，避免过快或过慢：

- 最短：`0.15s` (列表项悬停)
- 标准：`0.2s` (按钮、组件交互)
- 最长：`0.3s` (面板尺寸变化)

### 6.4 无障碍提示

- 所有图标按钮必须有 `title` 属性提供文本提示
- 颜色不是唯一的视觉提示手段
- 状态变化应有文字或图标辅助说明

---

## 附录 A: 快速参考卡

### 颜色速查

```
Primary:        var(--ws-primary)
Primary Light:  var(--ws-primary-light)
Primary Dark:   var(--ws-primary-dark)
Background:     var(--ws-bg-gradient)
Border:         var(--ws-border)
Glow:           var(--ws-glow)
```

### 间距速查

```
xs:  4px    |  sm: 6px   |  md: 8px
lg:  10px   |  xl: 12px  |  2xl: 16px
3xl: 20px   |  4xl: 24px
```

### 圆角速查

```
sm:  3px    |  md: 4px   |  lg: 6px
xl:  8px    |  2xl: 12px |  3xl: 16px
```

### 字体速查

```
xs:  10px    |  sm: 11-12px  |  md: 13-14px
lg:  15-16px |  xl: 18-20px  |  2xl: 24px
3xl: 28px    |  4xl: 72px
```

---

## 更新日志

| 版本 | 日期 | 更新内容 |
|------|------|----------|
| v1.0 | 2026-03-11 | 初始版本，整合现有 CSS 规范 |

---

*本规范应随 UI 发展持续更新，所有新组件设计应遵循此规范。*
