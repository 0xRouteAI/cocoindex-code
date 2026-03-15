# CocoIndex-Code Go 语言重构方案 (全云端 & MCP 适配版)

## 1. 项目愿景
本项目旨在将原有的 Python 版 `cocoindex-code` 重构为高性能、轻量级的 Go 语言版本。核心目标是：
- **完全去本地化**：移除所有本地 AI 模型（PyTorch, Sentence-Transformers, GPU 驱动）。
- **云端优先**：适配所有 OpenAI 兼容格式的 Embedding 和 Chat API（支持自定义 BaseURL）。
- **MCP 原生支持**：作为 Model Context Protocol 服务端，无缝外挂至 Claude Desktop、Cursor 等 IDE 环境。
- **极致分发**：编译为单一二进制文件，无环境依赖，安装即用。

---

## 2. 架构对比：为什么选择 Go？

| 维度 | 原 Python 版 | 建议 Rust 版 | **重构 Go 版 (推荐)** |
| :--- | :--- | :--- | :--- |
| **运行时** | 依赖 Python 环境/虚拟环境 | 无依赖 (编译为二进制) | **无依赖 (编译为二进制)** |
| **模型依赖** | 强制依赖本地模型/显存 | 可选本地模型 | **零本地模型，纯 API 调用** |
| **并发能力** | 异步 IO (单线程) | 协程 (极高性能但复杂) | **Goroutines (高性能且极简)** |
| **分发体积** | 几百 MB (含权重) | 几 MB | **约 10-15 MB** |
| **开发难度** | 高 (库多且乱) | 极高 (生命周期/所有权) | **低 (标准库强，开发快)** |

---

## 3. 技术栈选型

- **核心语言**: Go 1.21+
- **MCP 协议库**: `github.com/mark3labs/mcp-go` (成熟的 Go 版 MCP SDK)
- **OpenAI 客户端**: `github.com/sashabaranov/go-openai` (支持自定义 BaseURL)
- **本地向量库**: `github.com/mattn/go-sqlite3` + `sqlite-vec` 扩展
- **代码分块**: `github.com/tmc/langchaingo/textsplitter` (提供递归字符分块)
- **配置管理**: `github.com/spf13/viper` (支持环境变量和 YAML)

---

## 4. 核心模块设计

### 4.1 高性能文件扫描器 (Scanner)
利用 Go 的并行能力，快速遍历项目目录：
- 使用 `errgroup` 限制并发数。
- 自动过滤 `.gitignore` 中的文件（引用 `github.com/go-git/go-git/v5/plumbing/format/gitignore`）。

### 4.2 云端 Embedding 适配层 (Provider)
不再依赖 `litellm`，直接封装 OpenAI 兼容协议：
- 用户只需配置 `OPENAI_API_BASE` 和 `OPENAI_API_KEY`。
- 支持批量请求 Embedding（例如一次发送 100 个分块），最大化网络带宽利用率。

### 4.3 SQLite 向量存储 (Vector Store)
使用 SQLite 的 `sqlite-vec` 扩展：
- **Schema 设计**:
  ```sql
  CREATE TABLE code_chunks (
      id TEXT PRIMARY KEY,
      file_path TEXT,
      content TEXT,
      start_line INTEGER,
      end_line INTEGER,
      embedding F32_VEC(1536) -- 根据模型维度动态调整
  );
  CREATE VIRTUAL TABLE vec_index USING vec0(embedding_col="embedding");
  ```
- **搜索逻辑**: 使用 `vss_search` 进行毫秒级相似度匹配。

---

## 5. MCP Tool 接口定义

外挂到 Claude 后，将暴露以下三个核心工具：

1. `index_project(path string)`: 
   - 触发全量扫描 -> 分块 -> 云端 Embedding -> 写入 SQLite。
2. `search_code(query string, project_id string)`: 
   - 将 query 转化为向量 -> 在本地 SQLite 执行 VSS 搜索 -> 返回前 5-10 个相关代码片段。
3. `get_project_status()`: 
   - 查看当前已索引的项目列表及分块数量。

---

## 6. 查询性能预估

- **问题向量化**: 200ms - 500ms (云端延迟)
- **本地向量检索**: < 10ms (10,000 个分块规模)
- **结果组装**: < 1ms
- **总耗时**: **用户体感延迟约 0.5s 左右**，远快于原版加载本地模型后的检索速度。

---

## 7. 实施路线图

1. **环境准备**: 安装 Go 并下载 `sqlite-vec` 动态库。
2. **初始化项目**: `go mod init cocoindex-mcp`。
3. **实现 OpenAI Client**: 编写一个支持自定义 BaseURL 的封装。
4. **集成 TextSplitter**: 实现代码敏感的递归分块。
5. **构建 SQLite 层**: 完成向量表的创建与查询函数。
6. **接入 MCP SDK**: 包装以上逻辑为 MCP Tools。
7. **测试与发布**: 编写 `claude_desktop_config.json` 进行真机调试。

---

## 8. 总结
本方案通过 **Go + 云端 API + SQLite-vec** 的组合，完美解决了原项目“依赖重、部署难、GPU 要求高”的痛点。它提供了一个极其轻量级且响应迅速的代码搜索引擎，是目前作为 MCP 插件最理想的实现方式。
