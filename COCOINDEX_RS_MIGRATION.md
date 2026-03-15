# CocoIndex-Code Rust 重构方案 (全云端 & MCP 适配版)

## 1. 项目愿景 (Vision)
本项目旨在将 `cocoindex-code` 重构为高性能、工业级的 Rust 二进制程序。
- **远程依赖**：将 `cocoindex` 核心引擎作为远程 Git 依赖引用，本地无需源码。
- **全云端化**：移除所有本地模型依赖（Torch, Candle, ONNX），仅保留云端 OpenAI 兼容 API。
- **语法感知**：继承原版 `ops_text` 模块，原生支持 20+ 种编程语言的语法解析。
- **极致分发**：编译为单一静态链接二进制文件（约 8-12MB），无环境依赖，安装即用。

---

## 2. 技术栈选型 (Tech Stack)

| 模块 | Rust 选型 | 说明 |
| :--- | :--- | :--- |
| **异步运行时** | `tokio` | 高性能异步 IO 核心 |
| **命令行解析** | `clap` | 业界标准的 CLI 框架 (替代 Go 版 Cobra) |
| **云端 API** | `cocoindex::llm::openai` | 原生适配 OpenAI 格式，支持自定义 BaseURL |
| **语法分块** | `cocoindex_ops_text` | 基于 Tree-sitter 的递归分块 (1000 字符 + 200 重叠) |
| **本地存储** | `rusqlite` + `sqlite-vec` | 本地 SQLite 向量存储，支持 VSS 搜索 |
| **文件监控** | `notify` | 跨平台实时文件变动监听 (替代 Go 版 fsnotify) |
| **MCP 协议** | `mcp-sdk-rs` | 官方/社区 Rust MCP SDK |

---

## 3. 核心功能实现路径 (对标 Go 版)

### 3.1 远程依赖配置 (`Cargo.toml`)
无需本地源码，直接通过 Git 引用核心库。
```toml
[dependencies]
cocoindex = { git = "https://github.com/cocoindex-io/cocoindex.git", dir = "rust/cocoindex" }
cocoindex_ops_text = { git = "https://github.com/cocoindex-io/cocoindex.git", dir = "rust/ops_text" }
tokio = { version = "1.0", features = ["full"] }
serde = { version = "1.0", features = ["derive"] }
notify = "6.1"
```

### 3.2 语法感知分块 (Perception & Chunking)
直接复用原版最精华的 `RecursiveSplitter`，无需重写。
- **逻辑**：解析语法树 -> 识别函数/类 -> 递归切分 (Max 1000) -> 语义重叠 (200)。
- **支持语言**：原生支持 Go, Python, Rust, TS, C++, Java, PHP, Ruby 等。

### 3.3 性能优化与成本控制 (Indexer)
- **内容去重**：在索引前计算文件 MD5 哈希，比对数据库。如果哈希未变，跳过 API 请求。
- **并发流水线**：利用 `tokio::spawn` 开启 Worker Pool，并发请求云端 Embedding。
- **事务处理**：使用 SQLite 事务进行批量写入，保证数据一致性。

### 3.4 实时监控与增量更新 (Watcher)
- **递归监听**：自动监听新创建的子目录。
- **防抖 (Debounce)**：使用 `tokio::time::sleep` 配合 `Select` 逻辑，实现 300ms 防抖，避免高频保存触发 API 滥用。

### 3.5 搜索算法优化 (Store)
- **Top-K 堆排序**：使用 `BinaryHeap` 实现最小堆检索，在万级数据量下保持微秒级响应。
- **数学计算**：在 Rust 层执行高度优化的余弦相似度计算，支持加载 `sqlite-vec` 硬件加速。

---

## 4. 为什么 Rust 方案比 Go 版更强？

1. **分发更稳**：Rust 编译出的静态二进制文件对底层 C 库（如 SQLite）的链接处理比 Go 更加健壮，不会出现 Libc 版本冲突。
2. **内存极省**：由于没有 GC（垃圾回收），作为后台 MCP 服务运行速度更快，且内存占用通常不到 Go 的一半。
3. **算法原生**：直接使用原版 Rust 实现的算法，搜索质量与官方 Python 版 **100% 对齐**。
4. **代码量极少**：因为绝大部分逻辑都在库里，你只需要编写约 **300-500 行** 业务代码即可完成重构。

---

## 5. 如何作为 MCP 服务器使用 (Client Config)

修改你的 `claude_desktop_config.json` 或在 `claude` CLI 中指定：

```json
{
  "mcpServers": {
    "coco-rs": {
      "command": "/path/to/coco-rs-mcp",
      "env": {
        "OPENAI_API_KEY": "sk-xxxx",
        "OPENAI_API_BASE": "https://api.yourproxy.com/v1",
        "EMBEDDING_MODEL": "text-embedding-3-small"
      }
    }
  }
}
```

---

## 6. 总结
本方案实现了“**轻量化、全云端、全感知、全实时**”的重构目标。它不仅是一个插件，更是一个工业级的代码检索引擎。它让你的 AI 助手（Claude/Gemini）能够以最快的速度、最小的开销，读懂你最庞大的代码库。
