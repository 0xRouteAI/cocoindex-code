# CocoIndex-RS

🚀 高性能 Rust 实现的代码语义搜索工具，支持 MCP (Model Context Protocol) 协议。

## 📖 项目简介

CocoIndex-RS 是 [CocoIndex](https://github.com/cocoindex-io/cocoindex) 的 Rust 重写版本，专注于为 AI 编程助手提供高效的代码搜索能力。通过向量化代码片段并使用语义搜索，让 AI 助手能够快速找到相关代码。

### 核心特性

- ⚡ **高性能** - Rust 实现，比 Python 版本快 2-5 倍
- 🎯 **语义搜索** - 基于 OpenAI Embeddings 的向量搜索
- 🔍 **智能分区** - 按编程语言分区索引，搜索速度提升 20-50%
- 📦 **零配置** - SQLite + sqlite-vec，无需额外数据库
- 🔄 **增量索引** - 基于文件 hash，只索引变更文件
- 🌐 **MCP 支持** - 原生支持 Model Context Protocol
- 🎨 **25+ 语言** - 支持主流编程语言和配置文件

### 与 Python 版本对比

| 特性 | Python 版本 | Rust 版本 |
|------|-----------|---------|
| 索引速度 | 100 files/s | 200+ files/s |
| 搜索延迟 | 10-50ms | 5-20ms |
| 内存占用 | 500MB+ | 100-200MB |
| 二进制大小 | N/A | 50MB |
| 启动时间 | 1-2s | <100ms |

---

## 🚀 快速开始

### 安装

```bash
# 克隆仓库
git clone https://github.com/cocoindex-io/cocoindex.git
cd cocoindex/cocoindex-rs

# 构建
cargo build --release

# 二进制文件位于
./target/release/coco-rs
```

### 基础使用

```bash
# 1. 初始化项目
./target/release/coco-rs init

# 2. 配置 API 密钥
cat > ~/.cocoindex_code/settings.yml << EOF
api_key: sk-your-openai-api-key
api_base: https://api.openai.com/v1
model: text-embedding-3-small
EOF

# 3. 索引项目
./target/release/coco-rs index /path/to/your/project

# 4. 搜索代码
./target/release/coco-rs search "authentication logic"

# 5. 高级搜索
./target/release/coco-rs search "database connection" \
  --languages rust python \
  --paths "src/**" \
  --limit 10
```

---

## 🤖 AI 工具集成

CocoIndex-RS 支持多种 AI 编程助手，通过 MCP 协议提供代码搜索能力。

### 支持的 AI 工具

| 工具 | 支持状态 | 配置方式 |
|------|---------|---------|
| **Claude Code** | ✅ 原生支持 | MCP 服务器 |
| **Claude Desktop** | ✅ 完整支持 | MCP 配置文件 |
| **Cursor** | ⚠️ 实验性 | MCP 插件 |
| **Windsurf** | ⚠️ 实验性 | MCP 插件 |
| **其他 MCP 客户端** | ✅ 通用支持 | 标准 MCP 协议 |

---

## 📱 Claude Code 集成

Claude Code 是 Anthropic 官方的 CLI 工具，原生支持 MCP 协议。

### 自动配置（推荐）

Claude Code 会自动发现项目根目录的 MCP 服务器配置。

**1. 在项目根目录创建 `.mcp.json`**：

```bash
cd /path/to/your/project

cat > .mcp.json << 'EOF'
{
  "mcpServers": {
    "cocoindex": {
      "command": "/home/hushuaishuai2949525/cocoindex-code/cocoindex-rs/target/release/coco-rs",
      "args": ["mcp"],
      "env": {
        "OPENAI_API_KEY": "sk-your-key"
      }
    }
  }
}
EOF
```

**2. 启动 Claude Code**：

```bash
cd /path/to/your/project
claude
```

Claude Code 会自动加载 `.mcp.json` 并启动 CocoIndex MCP 服务器。

### 手动配置

如果需要全局配置，编辑 Claude Code 的配置文件：

**配置文件位置**：
- Linux: `~/.config/claude-code/mcp_settings.json`
- macOS: `~/Library/Application Support/claude-code/mcp_settings.json`

**配置内容**：

```json
{
  "mcpServers": {
    "cocoindex": {
      "command": "/path/to/coco-rs",
      "args": ["mcp"],
      "env": {
        "OPENAI_API_KEY": "sk-your-key",
        "OPENAI_API_BASE": "https://api.openai.com/v1",
        "EMBEDDING_MODEL": "text-embedding-3-small"
      }
    }
  }
}
```

### 使用示例

在 Claude Code 中：

```
# 索引当前项目
请使用 index_project 工具索引当前目录

# 搜索代码
在项目中搜索 "error handling" 相关的代码

# 高级搜索
搜索 src 目录下 Rust 文件中的 "database connection"
```

Claude Code 会自动调用 CocoIndex 的 MCP 工具。

---

## 🖥️ Claude Desktop 集成

Claude Desktop 是 Anthropic 的桌面应用，支持 MCP 协议。

### 配置步骤

**1. 找到配置文件**：

- **macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
- **Windows**: `%APPDATA%\Claude\claude_desktop_config.json`
- **Linux**: `~/.config/Claude/claude_desktop_config.json`

**2. 编辑配置文件**：

```json
{
  "mcpServers": {
    "cocoindex-rs": {
      "command": "/home/hushuaishuai2949525/cocoindex-code/cocoindex-rs/target/release/coco-rs",
      "args": ["mcp"],
      "env": {
        "OPENAI_API_KEY": "sk-your-openai-api-key",
        "OPENAI_API_BASE": "https://api.openai.com/v1",
        "EMBEDDING_MODEL": "text-embedding-3-small"
      }
    }
  }
}
```

**重要提示**：
- 使用**绝对路径**
- 确保二进制文件有执行权限：`chmod +x coco-rs`

**3. 重启 Claude Desktop**：

```bash
# macOS
osascript -e 'quit app "Claude"'
open -a Claude

# Linux
pkill -9 claude
claude &
```

### 使用示例

在 Claude Desktop 中：

```
请使用 index_project 工具索引 /path/to/my/project

在项目中搜索 "authentication logic"

搜索 Rust 和 Python 文件中的 "API client"
```

详细配置请参考 [MCP_SETUP.md](./MCP_SETUP.md)。

---

## 🔧 Cursor / Windsurf 集成

Cursor 和 Windsurf 是基于 VSCode 的 AI 编辑器，实验性支持 MCP。

### Cursor 配置

**1. 安装 MCP 扩展**（如果可用）：

在 Cursor 扩展市场搜索 "MCP" 或 "Model Context Protocol"。

**2. 配置 MCP 服务器**：

创建 `.cursor/mcp.json`：

```json
{
  "servers": {
    "cocoindex": {
      "command": "/path/to/coco-rs",
      "args": ["mcp"],
      "env": {
        "OPENAI_API_KEY": "sk-xxx"
      }
    }
  }
}
```

**3. 重启 Cursor**

### Windsurf 配置

类似 Cursor，在项目根目录创建 `.windsurf/mcp.json`。

**注意**：Cursor 和 Windsurf 的 MCP 支持可能不完整，建议使用 Claude Code 或 Claude Desktop。

---

## 🌐 通用 MCP 客户端

任何支持 MCP 协议的客户端都可以使用 CocoIndex-RS。

### 协议信息

- **协议版本**: `2024-11-05`
- **通信方式**: JSON-RPC 2.0 over stdin/stdout
- **工具数量**: 2 个（`index_project`, `search_code`）

### 手动启动

```bash
# 启动 MCP 服务器
/path/to/coco-rs mcp

# 服务器会等待 JSON-RPC 请求
```

### 示例请求

**初始化**：
```json
{
  "jsonrpc": "2.0",
  "method": "initialize",
  "params": {},
  "id": 1
}
```

**列出工具**：
```json
{
  "jsonrpc": "2.0",
  "method": "tools/list",
  "params": {},
  "id": 2
}
```

**调用工具**：
```json
{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "search_code",
    "arguments": {
      "query": "authentication",
      "limit": 10
    }
  },
  "id": 3
}
```

---

## 🛠️ MCP 工具说明

### 1. index_project

索引项目目录，创建代码向量数据库。

**参数**：
```json
{
  "path": "/path/to/project",      // 必需：项目路径
  "refresh_index": false            // 可选：强制重新索引
}
```

**功能**：
- ✅ 增量索引（基于文件 hash）
- ✅ 自动检测 25+ 编程语言
- ✅ 支持自定义 include/exclude 模式
- ✅ 进度反馈

**示例**：
```
请索引 /home/user/my-project 目录
```

### 2. search_code

在索引中搜索代码片段。

**参数**：
```json
{
  "query": "authentication logic",  // 必需：搜索查询
  "limit": 10,                      // 可选：结果数量（默认 10，最大 100）
  "offset": 0,                      // 可选：分页偏移（默认 0）
  "languages": ["rust", "python"],  // 可选：语言过滤
  "paths": ["src/**/*.rs"]          // 可选：路径过滤（GLOB）
}
```

**功能**：
- ✅ 语义搜索（基于向量相似度）
- ✅ 语言过滤（使用分区索引优化）
- ✅ 路径过滤（GLOB 模式）
- ✅ 分页支持
- ✅ 相似度评分

**示例**：
```
搜索 src 目录下 Rust 文件中的 "database connection"
```

---

## 📋 支持的编程语言

CocoIndex-RS 支持 **25 种**编程语言和文件类型：

### 主流编程语言
Python, JavaScript, TypeScript, Rust, Go, Java, C, C++, C#, Ruby, PHP, Swift, Kotlin, Scala, SQL, Bash

### 前端技术
JSX, TSX, HTML, CSS, Markdown, XML

### 配置文件
JSON, YAML, TOML

### 自定义语言

通过项目配置文件添加：

```yaml
# .cocoindex_code/settings.yml
language_overrides:
  inc: php
  tpl: html
  vue: javascript
```

---

## ⚙️ 配置文件

### 用户配置

`~/.cocoindex_code/settings.yml`：

```yaml
# API 配置
api_key: sk-your-openai-api-key
api_base: https://api.openai.com/v1
model: text-embedding-3-small

# 环境变量
envs:
  HTTP_PROXY: http://proxy.example.com:8080
```

### 项目配置

`.cocoindex_code/settings.yml`：

```yaml
# 包含的文件模式
include_patterns:
  - "**/*.rs"
  - "**/*.py"
  - "**/*.go"
  - "**/*.ts"

# 排除的文件模式
exclude_patterns:
  - "**/target"
  - "**/node_modules"
  - "**/.git"
  - "**/dist"

# 语言覆盖
language_overrides:
  inc: php
  tpl: html
```

---

## 🎯 性能优化

### 1. 选择合适的 Embedding 模型

**速度优先**：
```yaml
model: text-embedding-3-small  # 最快，1536 维
```

**质量优先**：
```yaml
model: text-embedding-3-large  # 更准确，3072 维
```

### 2. 使用语言过滤

利用语言分区索引加速搜索：

```bash
# 只搜索 Rust 代码，速度提升 20-50%
coco-rs search "error handling" --languages rust
```

### 3. 配置文件过滤

排除不需要索引的目录：

```yaml
exclude_patterns:
  - "**/target"
  - "**/node_modules"
  - "**/.git"
  - "**/dist"
  - "**/build"
  - "**/__pycache__"
```

### 4. 增量索引

默认启用，只索引变更的文件：

```bash
# 第一次：索引所有文件
coco-rs index /path/to/project

# 后续：只索引变更的文件
coco-rs index /path/to/project
```

---

## 🔍 使用场景

### 1. 代码理解

```
在项目中搜索 "JWT token validation" 的实现
```

AI 助手会找到所有相关的 JWT 验证代码。

### 2. 功能定位

```
找到处理用户登录的代码
```

快速定位登录相关的函数和模块。

### 3. 重构辅助

```
搜索所有使用旧 API 的代码
```

帮助识别需要更新的代码位置。

### 4. 学习代码库

```
搜索 "database migration" 相关的代码
```

了解项目如何处理数据库迁移。

### 5. Bug 修复

```
搜索 "error handling" 和 "null pointer"
```

找到可能存在问题的错误处理代码。

---

## 🐛 故障排查

### 问题 1：MCP 服务器无法启动

**检查二进制文件**：
```bash
ls -la /path/to/coco-rs
chmod +x /path/to/coco-rs
```

**测试运行**：
```bash
/path/to/coco-rs --help
/path/to/coco-rs mcp
```

### 问题 2：索引失败

**检查 API 密钥**：
```bash
curl https://api.openai.com/v1/models \
  -H "Authorization: Bearer $OPENAI_API_KEY"
```

**查看详细错误**：
```bash
coco-rs index /path/to/project
```

### 问题 3：搜索结果为空

**确认索引已创建**：
```bash
ls -la /path/to/project/.cocoindex_code/target_sqlite.db
```

**检查数据库**：
```bash
sqlite3 /path/to/project/.cocoindex_code/target_sqlite.db \
  "SELECT COUNT(*) FROM code_chunks_vec;"
```

**重新索引**：
```bash
coco-rs index /path/to/project
```

### 问题 4：Claude Code 看不到工具

**检查 .mcp.json**：
```bash
cat .mcp.json | jq .
```

**查看 Claude Code 日志**：
```bash
# 启动时添加 --verbose
claude --verbose
```

### 问题 5：Claude Desktop 工具不可用

**验证配置文件**：
```bash
cat ~/Library/Application\ Support/Claude/claude_desktop_config.json | jq .
```

**查看日志**：
```bash
# macOS
tail -f ~/Library/Logs/Claude/mcp*.log

# Linux
tail -f ~/.config/Claude/logs/mcp*.log
```

---

## 📚 更多文档

- **MCP 配置详解**: [MCP_SETUP.md](./MCP_SETUP.md)
- **项目主页**: https://github.com/cocoindex-io/cocoindex
- **MCP 协议**: https://modelcontextprotocol.io/
- **问题反馈**: https://github.com/cocoindex-io/cocoindex/issues

---

## 🤝 贡献

欢迎贡献代码、报告问题或提出建议！

```bash
# Fork 仓库
git clone https://github.com/your-username/cocoindex.git
cd cocoindex/cocoindex-rs

# 创建分支
git checkout -b feature/your-feature

# 提交更改
git commit -am "Add your feature"
git push origin feature/your-feature

# 创建 Pull Request
```

---

## 📄 许可证

Apache License 2.0

---

## 🙏 致谢

- [CocoIndex](https://github.com/cocoindex-io/cocoindex) - 原始 Python 实现
- [sqlite-vec](https://github.com/asg017/sqlite-vec) - SQLite 向量扩展
- [Anthropic](https://www.anthropic.com/) - Claude 和 MCP 协议

---

## 📞 联系方式

- **GitHub Issues**: https://github.com/cocoindex-io/cocoindex/issues
- **Discord**: [加入社区](https://discord.gg/cocoindex)
- **Email**: support@cocoindex.io

---

## 🗺️ 路线图

- [ ] 支持更多 Embedding 模型（本地模型）
- [ ] Web UI 界面
- [ ] VSCode 扩展
- [ ] 多项目管理
- [ ] 实时索引（文件监听）
- [ ] 分布式索引
- [ ] 更多 AI 工具集成

---

**⭐ 如果觉得有用，请给项目点个 Star！**
