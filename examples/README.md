# Right Panel Examples - 测试文件说明

本目录包含用于测试 GoTTY Right Panel 功能的示例文件。

## 文件列表

### 1. test-page.html
**用途**: HTTP 预览功能测试页面

**测试功能**:
- ✅ HTTP 预览正确加载网页
- ✅ CSS 样式渲染
- ✅ JavaScript 交互
- ✅ 滚动功能
- ✅ 截图功能（验证蓝色占位符 + URL 标注）

**使用方法**:

```bash
# 方式 1：使用提供的启动脚本（推荐）
cd /Users/oliveagle/ole/repos/github.com/oliveagle/gotty/examples
./start-server.sh

# 方式 2：直接使用 Python
cd /Users/oliveagle/ole/repos/github.com/oliveagle/gotty/examples
python3 -m http.server 8000
```

⚠️ **重要提示**：
- GoTTY 本身不是 HTTP 文件服务器，它只是一个 Web 终端应用
- 要测试 HTTP 预览功能，必须单独启动一个 HTTP 服务器
- 测试页面 URL：`http://localhost:8000/test-page.html`（不是 `http://localhost:13782/...`）

然后在 GoTTY Right Panel 中:
1. 点击右侧操作面板按钮（📋）
2. 切换到 "HTTP" 标签页
3. 输入：`http://localhost:8000/test-page.html`
4. 点击 "Load" 按钮

### 2. sample.txt
**用途**: 文件预览功能测试

**测试功能**:
- ✅ 文本文件正确显示
- ✅ 中文编码支持
- ✅ 滚动功能
- ✅ 文件大小限制验证

**使用方法**:
在 GoTTY Right Panel 中:
1. 点击右侧操作面板按钮（📋）
2. 切换到 "File" 标签页
3. 输入路径（两种方式）：
   - 相对路径（推荐）：`examples/sample.txt`
   - 绝对路径：`/Users/oliveagle/ole/repos/github.com/oliveagle/gotty/examples/sample.txt`
4. 点击 "Load" 按钮

## 测试清单

### HTTP 预览测试
- [ ] 页面正确加载
- [ ] 样式正常显示
- [ ] JavaScript 可以执行
- [ ] 按钮可以点击
- [ ] 复选框可以勾选
- [ ] 滚动功能正常
- [ ] 截图功能正常（蓝色占位符 + URL）

### 文件预览测试
- [ ] 文本文件正确显示
- [ ] 中文字符正常
- [ ] 长文本可以滚动
- [ ] 文件大小限制生效
- [ ] 二进制文件被拒绝

## 截图功能说明

由于浏览器安全限制（同源策略），html2canvas 无法渲染 iframe 内容。

**解决方案**:
- 在截图时，iframe 区域会显示为蓝色半透明占位符
- 占位符上会标注 iframe 的 URL
- 占位符右上角有 🌐 图标

**示例截图效果**:
```
┌─────────────────────────────────┐
│ Right Panel 📋                  │
├─────────────────────────────────┤
│ [HTTP Preview Content]          │
│ ┌─────────────────────────────┐ │
│ │  🌐 http://localhost:8000/  │ │
│ │     test-page.html          │ │
│ │  (蓝色半透明背景)            │ │
│ └─────────────────────────────┘ │
└─────────────────────────────────┘
```

## 故障排除

### HTTP 预览显示空白
**原因**:
- 跨域限制（CORS）
- 目标服务器拒绝 iframe 嵌入（X-Frame-Options）

**解决方法**:
- 使用本地 HTTP 服务器（如 `python3 -m http.server 8000`）
- 确保目标服务器允许 iframe 嵌入

### 文件预览失败
**原因**:
- 路径不正确
- 文件太大（超过 1MB）
- 二进制文件

**解决方法**:
- 使用绝对路径
- 确保文件是文本格式
- 文件大小不超过 1MB

### 截图功能异常
**原因**:
- html2canvas 未加载
- CSP 配置问题

**解决方法**:
- 检查浏览器控制台是否有错误
- 确认 CSP 包含 `script-src 'self' 'unsafe-inline' https://html2canvas.hertzen.com`

## 技术细节

### html2canvas 配置
```javascript
html2canvas(element, {
    backgroundColor: '#1e1e2e',
    scale: 1,
    logging: false,
    useCORS: true,
    allowTaint: true,
    ignoreElements: (el) => el.tagName === 'IFRAME'
})
```

### iframe 占位符绘制
```javascript
// 绘制蓝色半透明背景
ctx.fillStyle = 'rgba(74, 158, 255, 0.3)';
ctx.fillRect(x, y, width, height);

// 绘制边框
ctx.strokeStyle = '#4a9eff';
ctx.lineWidth = 2;
ctx.strokeRect(x, y, width, height);

// 绘制 URL 文本
ctx.fillStyle = '#ffffff';
ctx.font = 'bold 12px Monaco, Menlo, monospace';
ctx.fillText(url, x + 10, y + 8);
```

## 相关文件

- `resources/index.html` - 主页面，包含 Right Panel 实现
- `resources/css/index.css` - Right Panel 样式
- `server/middleware.go` - CSP 配置
- `server/handlers.go` - 文件预览 API

## 更新日志

### 2026-03-10 (下午)
- ✅ 修复文件预览 API 支持相对路径（如 `examples/sample.txt`）
- ✅ 修复 IRC Config 缺少 DataDir 字段导致编译失败

### 2026-03-10 (上午)
- ✅ 创建 examples 测试目录
- ✅ 添加 HTTP 预览测试页面
- ✅ 添加文件预览测试文件
- ✅ 修复 iframe 截图问题（蓝色占位符 + URL 标注）
- ✅ 更新 CSP 配置允许 html2canvas 和 iframe

---

*最后更新：2026-03-10*
