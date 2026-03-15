# cocoindex-go: Lightweight Syntax-Aware Code Indexer (MCP)
## cocoindex-go: 轻量级语法感知代码索引器 (MCP)

[English](#english) | [中文](#chinese)

---

<a name="english"></a>
## English

`cocoindex-go` is a high-performance, lightweight code indexing and search engine. It implements the **Model Context Protocol (MCP)**, allowing AI assistants to "read" and "understand" your local codebase.

### 🚀 Key Features
- **Zero Local Models**: All embeddings are processed via Cloud APIs.
- **Syntax-Aware Chunking**: Powered by **Tree-sitter**.
- **Multi-Client Support**: Compatible with Claude Code, Gemini CLI, and Claude Desktop.

### 📦 Installation
```bash
cd cocoindex-go
go build -o coco-mcp cmd/main.go
```

### 🖥 Usage as MCP Server (Client Configurations)

#### **1. Claude Desktop**
Add to `claude_desktop_config.json`:
```json
{
  "mcpServers": {
    "cocoindex": {
      "command": "/path/to/coco-mcp",
      "env": {
        "OPENAI_API_KEY": "your-key",
        "EMBEDDING_MODEL": "text-embedding-3-small"
      }
    }
  }
}
```

#### **2. Claude Code (CLI)**
When starting `claude`, use the `--mcp-server` flag:
```bash
claude --mcp-server /path/to/coco-mcp --env OPENAI_API_KEY=your-key
```

#### **3. Gemini CLI / Cursor**
Configure your MCP settings in the client dashboard, pointing to the binary path and providing the environment variables.

---

<a name="chinese"></a>
## 中文

`cocoindex-go` 是一个高性能、轻量级的代码索引与搜索引擎。它实现了 **Model Context Protocol (MCP)** 协议，让 AI 助手能够深度理解你的本地代码库。

### 🚀 核心特性
- **零本地模型**：所有向量化计算均通过云端 API 完成。
- **语法感知分块**：基于 **Tree-sitter** 驱动。
- **多客户端支持**：完美适配 Claude Code, Gemini CLI 和 Claude Desktop。

### 📦 安装指南
```bash
cd cocoindex-go
go build -o coco-mcp cmd/main.go
```

### 🖥 如何作为 MCP 服务器使用 (客户端配置)

#### **1. Claude Desktop (桌面版)**
修改 `claude_desktop_config.json`:
```json
{
  "mcpServers": {
    "cocoindex": {
      "command": "/你的路径/到/coco-mcp",
      "env": {
        "OPENAI_API_KEY": "你的-API-KEY",
        "OPENAI_API_BASE": "https://api.openai.com/v1",
        "EMBEDDING_MODEL": "text-embedding-3-small"
      }
    }
  }
}
```

#### **2. Claude Code (命令行版)**
在启动 `claude` 时通过命令行参数挂载：
```bash
claude --mcp-server /path/to/coco-mcp --env OPENAI_API_KEY=your-key
```

#### **3. Gemini CLI / Cursor**
在这些工具的 MCP 设置面板中，添加一个新的服务器，指向该二进制文件的绝对路径，并配置环境变量即可。

### ⚙️ 环境变量配置
- `OPENAI_API_KEY`: 必填，用于请求 Embedding。
- `OPENAI_API_BASE`: 可选，适配国内转发或自定义 API 地址。
- `EMBEDDING_MODEL`: 可选，默认为 `text-embedding-3-small`，可根据云端支持的模型自行修改。
