# CocoIndex-RS MCP 服务器配置指南

本文档介绍如何配置和使用 CocoIndex-RS 的 MCP (Model Context Protocol) 服务器，支持 Claude Desktop 和其他 MCP 客户端。

## 目录

- [快速开始](#快速开始)
- [Claude Desktop 配置](#claude-desktop-配置)
- [环境变量配置](#环境变量配置)
- [使用示例](#使用示例)
- [故障排查](#故障排查)

---

## 快速开始

### 1. 构建 CocoIndex-RS

```bash
cd cocoindex-rs
cargo build --release
```

构建完成后，二进制文件位于：`./target/release/coco-rs`

### 2. 配置 API 密钥

创建用户配置文件：

```bash
mkdir -p ~/.cocoindex_code
cat > ~/.cocoindex_code/settings.yml << EOF
api_key: sk-your-openai-api-key
api_base: https://api.openai.com/v1
model: text-embedding-3-small
EOF
```

### 3. 测试 MCP 服务器

```bash
./target/release/coco-rs mcp
```

服务器启动后会等待 JSON-RPC 输入。按 `Ctrl+C` 退出。

---

## Claude Desktop 配置

### 配置文件位置

**macOS**:
```
~/Library/Application Support/Claude/claude_desktop_config.json
```

**Windows**:
```
%APPDATA%\Claude\claude_desktop_config.json
```

**Linux**:
```
~/.config/Claude/claude_desktop_config.json
```

### 配置示例

编辑 `claude_desktop_config.json`：

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
- 将 `command` 路径替换为你的实际路径
- 使用**绝对路径**，不要使用 `~` 或相对路径
- 确保二进制文件有执行权限：`chmod +x coco-rs`

### 重启 Claude Desktop

配置完成后，**完全退出并重启** Claude Desktop：

**macOS**:
```bash
# 完全退出
osascript -e 'quit app "Claude"'

# 重新启动
open -a Claude
```

**Linux**:
```bash
# 杀死所有 Claude 进程
pkill -9 claude

# 重新启动
claude &
```

---

## 环境变量配置

### 方式 1：在 MCP 配置中设置（推荐）

直接在 `claude_desktop_config.json` 的 `env` 字段中设置：

```json
{
  "mcpServers": {
    "cocoindex-rs": {
      "command": "/path/to/coco-rs",
      "args": ["mcp"],
      "env": {
        "OPENAI_API_KEY": "sk-xxx",
        "OPENAI_API_BASE": "https://api.openai.com/v1",
        "EMBEDDING_MODEL": "text-embedding-3-small"
      }
    }
  }
}
```

### 方式 2：使用用户配置文件

创建 `~/.cocoindex_code/settings.yml`：

```yaml
api_key: sk-your-openai-api-key
api_base: https://api.openai.com/v1
model: text-embedding-3-small
```

这样 MCP 配置可以简化为：

```json
{
  "mcpServers": {
    "cocoindex-rs": {
      "command": "/path/to/coco-rs",
      "args": ["mcp"]
    }
  }
}
```

### 支持的 API 提供商

#### OpenAI
```json
"env": {
  "OPENAI_API_KEY": "sk-xxx",
  "OPENAI_API_BASE": "https://api.openai.com/v1",
  "EMBEDDING_MODEL": "text-embedding-3-small"
}
```

#### Azure OpenAI
```json
"env": {
  "OPENAI_API_KEY": "your-azure-key",
  "OPENAI_API_BASE": "https://your-resource.openai.azure.com/openai/deployments/your-deployment",
  "EMBEDDING_MODEL": "text-embedding-ada-002"
}
```

#### 其他兼容 OpenAI API 的服务
```json
"env": {
  "OPENAI_API_KEY": "your-api-key",
  "OPENAI_API_BASE": "https://your-api-endpoint/v1",
  "EMBEDDING_MODEL": "your-model-name"
}
```

---

## 使用示例

### 在 Claude Desktop 中使用

配置完成后，在 Claude Desktop 中可以看到两个新工具：

#### 1. **index_project** - 索引项目

```
请使用 index_project 工具索引 /path/to/my/project
```

Claude 会调用：
```json
{
  "name": "index_project",
  "arguments": {
    "path": "/path/to/my/project",
    "refresh_index": false
  }
}
```

**参数说明**：
- `path`: 项目目录路径（必需）
- `refresh_index`: 是否强制重新索引（可选，默认 false）

#### 2. **search_code** - 搜索代码

```
在当前项目中搜索 "authentication logic"
```

Claude 会调用：
```json
{
  "name": "search_code",
  "arguments": {
    "query": "authentication logic",
    "limit": 10,
    "offset": 0
  }
}
```

**参数说明**：
- `query`: 搜索查询（必需）
- `limit`: 返回结果数量（可选，默认 10，最大 100）
- `offset`: 分页偏移（可选，默认 0）
- `languages`: 语言过滤（可选，如 `["rust", "python"]`）
- `paths`: 路径过滤（可选，GLOB 模式，如 `["src/**/*.rs"]`）

### 高级搜索示例

#### 按语言过滤
```
搜索 Rust 和 Python 文件中的 "error handling"
```

```json
{
  "query": "error handling",
  "languages": ["rust", "python"]
}
```

#### 按路径过滤
```
在 src 目录下搜索 "database connection"
```

```json
{
  "query": "database connection",
  "paths": ["src/**"]
}
```

#### 组合过滤 + 分页
```
搜索 src 目录下 Rust 文件中的 "API client"，显示第 11-20 条结果
```

```json
{
  "query": "API client",
  "languages": ["rust"],
  "paths": ["src/**/*.rs"],
  "limit": 10,
  "offset": 10
}
```

---

## 其他 MCP 客户端配置

### Gemini CLI (假设支持 MCP)

如果 Gemini CLI 支持 MCP 协议，配置方式类似：

```json
{
  "mcp_servers": {
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

### 通用 MCP 客户端

任何支持 MCP 协议的客户端都可以使用：

```bash
# 启动 MCP 服务器
/path/to/coco-rs mcp

# 服务器通过 stdin/stdout 进行 JSON-RPC 通信
```

**协议版本**: `2024-11-05`

---

## 故障排查

### 1. Claude Desktop 看不到工具

**检查配置文件**：
```bash
# macOS
cat ~/Library/Application\ Support/Claude/claude_desktop_config.json

# Linux
cat ~/.config/Claude/claude_desktop_config.json
```

**验证 JSON 格式**：
```bash
# 使用 jq 验证
cat claude_desktop_config.json | jq .
```

**检查日志**：
- macOS: `~/Library/Logs/Claude/mcp*.log`
- Linux: `~/.config/Claude/logs/mcp*.log`

### 2. 工具调用失败

**测试 API 连接**：
```bash
export OPENAI_API_KEY="sk-xxx"
./target/release/coco-rs init
./target/release/coco-rs index .
./target/release/coco-rs search "test query"
```

**检查 API 密钥**：
```bash
# 测试 OpenAI API
curl https://api.openai.com/v1/models \
  -H "Authorization: Bearer $OPENAI_API_KEY"
```

### 3. 索引失败

**检查项目配置**：
```bash
cd /path/to/your/project
cat .cocoindex_code/settings.yml
```

**手动初始化**：
```bash
./target/release/coco-rs init /path/to/your/project
```

**查看详细错误**：
```bash
# 直接运行索引命令查看错误
./target/release/coco-rs index /path/to/your/project
```

### 4. 搜索结果为空

**确认索引已创建**：
```bash
ls -la /path/to/your/project/.cocoindex_code/
# 应该看到 target_sqlite.db 文件
```

**检查数据库**：
```bash
sqlite3 /path/to/your/project/.cocoindex_code/target_sqlite.db "SELECT COUNT(*) FROM code_chunks_vec;"
```

**重新索引**：
```json
{
  "name": "index_project",
  "arguments": {
    "path": "/path/to/your/project",
    "refresh_index": true
  }
}
```

### 5. 权限问题

**确保二进制可执行**：
```bash
chmod +x /path/to/coco-rs
```

**检查数据库目录权限**：
```bash
ls -la /path/to/your/project/.cocoindex_code/
chmod 755 /path/to/your/project/.cocoindex_code/
```

---

## 性能优化

### 1. 选择合适的 Embedding 模型

**速度优先**：
```yaml
model: text-embedding-3-small  # 最快，1536 维
```

**质量优先**：
```yaml
model: text-embedding-3-large  # 更准确，3072 维
```

### 2. 配置文件过滤

编辑 `.cocoindex_code/settings.yml`：

```yaml
include_patterns:
  - "**/*.rs"
  - "**/*.py"
  - "**/*.go"
  - "**/*.ts"

exclude_patterns:
  - "**/target"
  - "**/node_modules"
  - "**/.git"
  - "**/dist"
  - "**/build"
```

### 3. 使用语言过滤加速搜索

利用语言分区索引：

```json
{
  "query": "authentication",
  "languages": ["rust"]  // 只搜索 Rust 代码，速度提升 20-50%
}
```

---

## 支持的语言

CocoIndex-RS 支持 25 种编程语言：

**主流语言**: Python, JavaScript, TypeScript, Rust, Go, Java, C, C++, C#, Ruby, PHP, Swift, Kotlin, Scala, SQL, Bash

**前端**: JSX, TSX, HTML, CSS, Markdown, XML

**配置**: JSON, YAML, TOML

可以通过 `language_overrides` 添加自定义语言映射。

---

## 更多信息

- **项目主页**: https://github.com/cocoindex-io/cocoindex
- **MCP 协议**: https://modelcontextprotocol.io/
- **问题反馈**: https://github.com/cocoindex-io/cocoindex/issues

---

## 完整配置示例

### Claude Desktop 完整配置

```json
{
  "mcpServers": {
    "cocoindex-rs": {
      "command": "/home/user/cocoindex-code/cocoindex-rs/target/release/coco-rs",
      "args": ["mcp"],
      "env": {
        "OPENAI_API_KEY": "sk-proj-xxx",
        "OPENAI_API_BASE": "https://api.openai.com/v1",
        "EMBEDDING_MODEL": "text-embedding-3-small"
      }
    }
  },
  "globalShortcut": "CommandOrControl+Shift+Space"
}
```

### 用户配置文件

`~/.cocoindex_code/settings.yml`:
```yaml
api_key: sk-proj-xxx
api_base: https://api.openai.com/v1
model: text-embedding-3-small
envs:
  HTTP_PROXY: http://proxy.example.com:8080
```

### 项目配置文件

`/path/to/project/.cocoindex_code/settings.yml`:
```yaml
include_patterns:
  - "**/*.rs"
  - "**/*.py"
  - "**/*.go"
  - "**/*.ts"
  - "**/*.tsx"

exclude_patterns:
  - "**/target"
  - "**/node_modules"
  - "**/.git"
  - "**/dist"
  - "**/build"
  - "**/__pycache__"
  - "**/.pytest_cache"

language_overrides:
  inc: php
  tpl: html
  conf: toml
```

---

## 快速参考

### 常用命令

```bash
# 初始化项目
coco-rs init

# 索引项目
coco-rs index /path/to/project

# 搜索代码
coco-rs search "query"

# 启动 MCP 服务器
coco-rs mcp

# 查看状态
coco-rs status
```

### MCP 工具

| 工具 | 功能 | 必需参数 |
|------|------|---------|
| `index_project` | 索引项目 | `path` |
| `search_code` | 搜索代码 | `query` |

### 环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `OPENAI_API_KEY` | API 密钥 | - |
| `OPENAI_API_BASE` | API 端点 | `https://api.openai.com/v1` |
| `EMBEDDING_MODEL` | 模型名称 | `text-embedding-3-small` |
