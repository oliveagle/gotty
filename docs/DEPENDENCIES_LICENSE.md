# 依赖项许可证清单

> **创建时间**: 2026-03-09
> **最后更新**: 2026-03-09
> **目的**: 记录项目所有依赖项的许可证信息，确认商用许可状态

---

## 总结

**所有依赖项均允许商业使用**。本项目使用的许可证均为宽松型开源许可证，对商业用途无限制。

| 依赖类型 | 数量 | 许可证类型 | 商用状态 |
|----------|------|------------|----------|
| Go 依赖 | 13 | MIT / BSD / Apache-2.0 / MPL-2.0 | ✅ 允许 |
| npm 依赖 | 4 (核心) | MIT / Apache-2.0 | ✅ 允许 |

---

## Go 依赖项

### 直接依赖 (go.mod)

| 依赖项 | 许可证 | 商用 | 注意事项 |
|--------|--------|------|----------|
| `github.com/NYTimes/gziphandler` | Apache-2.0 | ✅ | 保留版权声明和许可证文本 |
| `github.com/creack/pty` | MIT | ✅ | 保留版权声明 |
| `github.com/elazarl/go-bindata-assetfs` | BSD-3-Clause | ✅ | 保留版权声明，二进制分发需包含声明 |
| `github.com/fatih/structs` | BSD-2-Clause | ✅ | 保留版权声明 |
| `github.com/gorilla/websocket` | BSD-2-Clause | ✅ | 保留版权声明 |
| `github.com/pkg/errors` | BSD-2-Clause | ✅ | 保留版权声明 |
| `github.com/urfave/cli/v2` | MIT | ✅ | 保留版权声明 |
| `github.com/yudai/hcl` | MPL-2.0 | ✅ | 修改需开源，保留版权声明 |

### 间接依赖 (Indirect)

| 依赖项 | 许可证 | 商用 | 注意事项 |
|--------|--------|------|----------|
| `github.com/cpuguy83/go-md2man/v2` | MIT | ✅ | 保留版权声明 |
| `github.com/hashicorp/errwrap` | MPL-2.0 | ✅ | 修改需开源 |
| `github.com/hashicorp/go-multierror` | MPL-2.0 | ✅ | 修改需开源 |
| `github.com/russross/blackfriday/v2` | BSD-2-Clause | ✅ | 保留版权声明 |
| `github.com/xrash/smetrics` | MIT | ✅ | 保留版权声明 |

---

## npm 依赖项

### 核心依赖 (package.json)

| 依赖项 | 许可证 | 商用 | 注意事项 |
|--------|--------|------|----------|
| `@rspack/cli` | MIT | ✅ | 保留版权声明 |
| `@rspack/core` | MIT | ✅ | 保留版权声明 |
| `typescript` | Apache-2.0 | ✅ | 保留版权声明和许可证文本 |
| `@xterm/addon-fit` | MIT | ✅ | 保留版权声明 |
| `@xterm/addon-webgl` | MIT | ✅ | 保留版权声明 |
| `@xterm/xterm` | MIT | ✅ | 保留版权声明 |
| `libapps` | MIT | ✅ | 保留版权声明 |

### 主要传递依赖

以下依赖通过上述核心依赖自动安装，同样允许商用：

| 依赖项 | 许可证 | 商用 |
|--------|--------|------|
| `express` | MIT | ✅ |
| `ws` | MIT | ✅ |
| `ajv` | MIT | ✅ |
| `webpack-dev-middleware` | MIT | ✅ |
| `@types/node` | MIT | ✅ |

---

## 许可证类型说明

### MIT License
- **商用**: 允许
- **修改**: 允许
- **分发**: 允许
- **专利授权**: 无明确条款
- **Copyleft**: 无

**要求**: 保留版权声明和许可证文本

### BSD 2-Clause / 3-Clause License
- **商用**: 允许
- **修改**: 允许
- **分发**: 允许
- **专利授权**: 无明确条款
- **Copyleft**: 无

**要求**: 保留版权声明，二进制分发需包含声明

### Apache License 2.0
- **商用**: 允许
- **修改**: 允许
- **分发**: 允许
- **专利授权**: ✅ 明确授予
- **Copyleft**: 无

**要求**: 保留版权声明、许可证文本和 NOTICE 文件（如有）

### Mozilla Public License 2.0 (MPL-2.0)
- **商用**: 允许
- **修改**: 允许
- **分发**: 允许
- **专利授权**: ✅ 明确授予
- **Copyleft**: ⚠️ 弱 Copyleft

**要求**:
- 保留版权声明
- **修改 MPL-2.0 许可的文件必须以相同许可证开源**
- 新文件和链接的库可以使用其他许可证

---

## 商用合规指南

### ✅ 允许的行为

1. **商业部署**: 可以将本项目用于商业服务
2. **修改代码**: 可以修改源代码用于商业目的
3. **分发**: 可以分发编译后的二进制文件
4. **SaaS 使用**: 可以作为服务提供给用户

### ⚠️ 注意事项

1. **保留版权声明**: 所有许可证都要求保留原始版权声明
2. **MPL-2.0 文件**: 如果修改了使用 MPL-2.0 许可证的文件（hashicorp 相关库），需要将修改部分开源
3. **许可证文本**: 分发时应包含本项目和各依赖项的许可证文本

### 📋 建议做法

1. 在产品的"关于"或"许可证"页面列出所有依赖项及其许可证
2. 保留源代码中的版权声明
3. 如果分发二进制文件，附带一份许可证声明文档

---

## 本项目许可证

本项目 (gotty) 使用 **MIT License**：

```
Copyright (c) 2015-2017 Iwasaki Yudai

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
```

---

## 结论

✅ **本项目及所有依赖项均允许商业使用**，无许可证限制。

仅需遵循各许可证的简单要求（保留版权声明等），即可放心用于商业用途。

---

*文档生成时间：2026-03-09*
