# CocoIndex-RS 完整实现方案

## 1. 项目目标

将 Python 版本的 `cocoindex-code` 重构为 Rust 版本，保持核心功能一致，但简化架构：
- ✅ 只支持 OpenAI 兼容 API（不内置本地模型）
- ✅ 单进程架构（无 Daemon）
- ✅ 使用 sqlite-vec 实现高性能向量搜索
- ✅ 完整的配置管理系统
- ✅ MCP 服务器支持

---

## 2. 功能清单

### 2.1 核心功能

| # | 功能 | Python 实现 | Rust 实现状态 | 优先级 |
|---|------|------------|-------------|--------|
| 1 | 文本分块 | `RecursiveSplitter` | ⚠️ API 错误 | P0 |
| 2 | 向量搜索 | sqlite-vec KNN | ❌ 全表扫描 | P0 |
| 3 | Embedding API | OpenAI 兼容 | ✅ 已实现 | P0 |
| 4 | 增量索引 | 基于 MD5 hash | ✅ 已实现 | P0 |
| 5 | 语言检测 | `detect_code_language()` | ❌ 未实现 | P1 |
| 6 | MCP 服务器 | FastMCP | ❌ 空占位 | P1 |
| 10 | 语言/路径过滤 | SQL WHERE | ❌ 未实现 | P2 |

### 2.2 配置管理

| # | 功能 | Python 实现 | Rust 实现状态 | 优先级 |
|---|------|------------|-------------|--------|
| 11 | 用户设置 | `UserSettings` | ❌ 未实现 | P0 |
| 12 | 项目设置 | `ProjectSettings` | ❌ 未实现 | P0 |
| 13 | 文件模式过滤 | include/exclude patterns | ❌ 硬编码 | P0 |
| 14 | 语言覆盖 | `LanguageOverride` | ❌ 未实现 | P2 |

### 2.3 不实现的功能

- ❌ Daemon 架构（单进程足够）
- ❌ Client-Server IPC（不需要）
- ❌ 本地 Embedding 模型（只用 API）
- ❌ sentence-transformers（只用 OpenAI 格式）
- ❌ 项目注册表（单项目场景）

---

## 3. 技术栈

### 3.1 依赖库

```toml
[dependencies]
# 核心引擎（本地路径，避免 800+ 依赖）
cocoindex_ops_text = { path = "../cocoindex/rust/ops_text" }

# 异步运行时
tokio = { version = "1.48", features = ["full"] }
futures = "0.3"

# HTTP 客户端（rustls，避免 openssl）
reqwest = { version = "0.12", default-features = false, features = ["json", "rustls-tls"] }

# 数据库
rusqlite = { version = "0.32", features = ["bundled"] }
# TODO: 添加 sqlite-vec 支持

# 配置管理
serde = { version = "1.0", features = ["derive"] }
serde_json = "1.0"
serde_yaml = "0.9"

# CLI
clap = { version = "4.5", features = ["derive", "env"] }

# MCP
# TODO: 选择合适的 Rust MCP SDK

# 工具
anyhow = "1.0"
tracing = "0.1"
tracing-subscriber = "0.3"
walkdir = "2.5"
md5 = "0.7"
globset = "0.4"
```

### 3.2 项目结构

```
cocoindex-rs/
├── Cargo.toml
├── src/
│   ├── main.rs              # CLI 入口
│   ├── lib.rs               # 库入口
│   ├── config/
│   │   ├── mod.rs           # 配置管理
│   │   ├── user.rs          # 用户设置
│   │   └── project.rs       # 项目设置
│   ├── indexer/
│   │   ├── mod.rs           # 索引器
│   │   ├── chunker.rs       # 分块逻辑
│   │   └── scanner.rs       # 文件扫描
│   ├── store/
│   │   ├── mod.rs           # 数据库
│   │   └── vec_search.rs    # 向量搜索
│   ├── provider/
│   │   └── mod.rs           # OpenAI API 客户端
│   ├── mcp/
│   │   └── mod.rs           # MCP 服务器
│   └── utils/
│       ├── language.rs      # 语言检测
│       └── patterns.rs      # 文件模式匹配
```

---

## 4. 详细实现计划

### 4.1 阶段 1：修复基础功能（P0）

#### 4.1.1 修复 Cargo.toml 依赖

**当前问题**：
- 使用远程 Git 依赖，导致 800+ 包编译
- 使用 `sqlx` 但代码用的是 `rusqlite`

**解决方案**：
```toml
# 使用本地路径依赖
cocoindex_ops_text = { path = "../cocoindex/rust/ops_text" }

# 移除 sqlx，添加 rusqlite
rusqlite = { version = "0.32", features = ["bundled"] }

# 确保所有依赖使用 rustls
reqwest = { version = "0.12", default-features = false, features = ["json", "rustls-tls"] }
```

#### 4.1.2 修复分块 API（功能 1）

**当前问题** (`indexer/mod.rs:5, 20, 46`):
```rust
use cocoindex_ops_text::RecursiveSplitter;  // ❌ 不存在
let splitter = RecursiveSplitter::new(1000, 200);  // ❌ API 错误
let chunks = splitter.split(&content, Some(path));  // ❌ API 错误
```

**正确实现**：
```rust
use cocoindex_ops_text::{
    RecursiveChunker,
    RecursiveChunkConfig,
    RecursiveSplitConfig,
};

// 创建分块器
let chunker = RecursiveChunker::new(RecursiveSplitConfig::default())?;

// 分块
let chunks = chunker.split(&content, RecursiveChunkConfig {
    chunk_size: 2000,
    min_chunk_size: Some(300),
    chunk_overlap: Some(200),
    language: Some(detect_language(path)),
});

// 使用结果
for chunk in chunks {
    let text = &content[chunk.range.start..chunk.range.end];
    let start_line = chunk.start.line;
    let end_line = chunk.end.line;
    // ...
}
```

#### 4.1.3 实现配置系统（功能 11, 12, 13）

**文件结构**：
```
~/.cocoindex_code/
└── settings.yml          # 用户设置

项目根目录/
└── .cocoindex_code/
    ├── settings.yml      # 项目设置
    └── target_sqlite.db  # 索引数据库
```

**配置定义** (`config/mod.rs`):
```rust
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::path::{Path, PathBuf};

// 用户设置（全局）
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UserSettings {
    pub api_key: String,
    pub api_base: String,
    pub model: String,
    #[serde(default)]
    pub envs: HashMap<String, String>,
}

// 项目设置（每个项目）
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProjectSettings {
    pub include_patterns: Vec<String>,
    pub exclude_patterns: Vec<String>,
    #[serde(default)]
    pub language_overrides: HashMap<String, String>,
}

impl UserSettings {
    pub fn load() -> Result<Self> {
        let path = dirs::home_dir()
            .ok_or_else(|| anyhow!("Cannot find home directory"))?
            .join(".cocoindex_code/settings.yml");

        if !path.exists() {
            return Ok(Self::default());
        }

        let content = std::fs::read_to_string(path)?;
        Ok(serde_yaml::from_str(&content)?)
    }

    pub fn save(&self) -> Result<()> {
        let path = dirs::home_dir()
            .ok_or_else(|| anyhow!("Cannot find home directory"))?
            .join(".cocoindex_code/settings.yml");

        std::fs::create_dir_all(path.parent().unwrap())?;
        let content = serde_yaml::to_string(self)?;
        std::fs::write(path, content)?;
        Ok(())
    }
}

impl Default for UserSettings {
    fn default() -> Self {
        Self {
            api_key: std::env::var("OPENAI_API_KEY").unwrap_or_default(),
            api_base: std::env::var("OPENAI_API_BASE")
                .unwrap_or_else(|_| "https://api.openai.com/v1".to_string()),
            model: std::env::var("EMBEDDING_MODEL")
                .unwrap_or_else(|_| "text-embedding-3-small".to_string()),
            envs: HashMap::new(),
        }
    }
}

impl ProjectSettings {
    pub fn load(project_root: &Path) -> Result<Self> {
        let path = project_root.join(".cocoindex_code/settings.yml");

        if !path.exists() {
            return Ok(Self::default());
        }

        let content = std::fs::read_to_string(path)?;
        Ok(serde_yaml::from_str(&content)?)
    }

    pub fn save(&self, project_root: &Path) -> Result<()> {
        let path = project_root.join(".cocoindex_code/settings.yml");
        std::fs::create_dir_all(path.parent().unwrap())?;
        let content = serde_yaml::to_string(self)?;
        std::fs::write(path, content)?;
        Ok(())
    }
}

impl Default for ProjectSettings {
    fn default() -> Self {
        Self {
            include_patterns: vec![
                "**/*.py", "**/*.js", "**/*.ts", "**/*.rs", "**/*.go",
                "**/*.java", "**/*.c", "**/*.cpp", "**/*.h", "**/*.hpp",
            ].into_iter().map(String::from).collect(),
            exclude_patterns: vec![
                "**/.*", "**/__pycache__", "**/node_modules",
                "**/target", "**/dist", "**/build",
            ].into_iter().map(String::from).collect(),
            language_overrides: HashMap::new(),
        }
    }
}
```

**配置示例**：

`~/.cocoindex_code/settings.yml`:
```yaml
api_key: sk-xxx
api_base: https://api.openai.com/v1
model: text-embedding-3-small
envs:
  SOME_ENV: value
```

`项目/.cocoindex_code/settings.yml`:
```yaml
include_patterns:
  - "**/*.rs"
  - "**/*.py"
  - "**/*.go"
exclude_patterns:
  - "**/target"
  - "**/node_modules"
  - "**/.git"
language_overrides:
  inc: php
  tpl: html
```

#### 4.1.4 实现文件模式匹配（功能 13）

**使用 globset** (`utils/patterns.rs`):
```rust
use globset::{Glob, GlobSet, GlobSetBuilder};
use std::path::Path;

pub struct PatternMatcher {
    include: GlobSet,
    exclude: GlobSet,
}

impl PatternMatcher {
    pub fn new(
        include_patterns: &[String],
        exclude_patterns: &[String],
    ) -> Result<Self> {
        let mut include_builder = GlobSetBuilder::new();
        for pattern in include_patterns {
            include_builder.add(Glob::new(pattern)?);
        }

        let mut exclude_builder = GlobSetBuilder::new();
        for pattern in exclude_patterns {
            exclude_builder.add(Glob::new(pattern)?);
        }

        Ok(Self {
            include: include_builder.build()?,
            exclude: exclude_builder.build()?,
        })
    }

    pub fn matches(&self, path: &Path) -> bool {
        let path_str = path.to_string_lossy();

        // 先检查排除规则
        if self.exclude.is_match(&path_str) {
            return false;
        }

        // 再检查包含规则
        self.include.is_match(&path_str)
    }
}
```

---

### 4.2 阶段 2：集成 sqlite-vec（P0）

#### 4.2.1 添加 sqlite-vec 依赖

**方案 A：使用 sqlite-vec Rust 绑定**（推荐）
```toml
[dependencies]
rusqlite = { version = "0.32", features = ["bundled"] }
# TODO: 查找 sqlite-vec 的 Rust 绑定
```

**方案 B：手动加载扩展**
```rust
use rusqlite::Connection;

let conn = Connection::open(db_path)?;
conn.load_extension_enable()?;
conn.load_extension("path/to/vec0.so", None)?;
```

#### 4.2.2 修改数据库 Schema

**当前 Schema** (`store/mod.rs:14-24`):
```sql
CREATE TABLE code_chunks (
    id TEXT PRIMARY KEY,
    file_path TEXT,
    content TEXT,
    start_line INTEGER,
    end_line INTEGER,
    hash TEXT,
    embedding BLOB  -- ❌ 普通 BLOB
);
```

**新 Schema（使用 vec0）**:
```sql
-- 主表
CREATE TABLE IF NOT EXISTS code_chunks_vec (
    id INTEGER PRIMARY KEY,
    file_path TEXT NOT NULL,
    language TEXT NOT NULL,
    content TEXT NOT NULL,
    start_line INTEGER NOT NULL,
    end_line INTEGER NOT NULL,
    embedding BLOB NOT NULL
) USING vec0(
    embedding FLOAT[1536],  -- 根据模型维度调整
    partition_key=language,  -- 按语言分区
    auxiliary_columns=[file_path, content, start_line, end_line]
);

-- 文件哈希表（用于增量索引）
CREATE TABLE IF NOT EXISTS file_hashes (
    file_path TEXT PRIMARY KEY,
    hash TEXT NOT NULL
);
```

#### 4.2.3 实现向量搜索

**当前实现** (`store/mod.rs:75-99`):
```rust
// ❌ 全表扫描
pub fn search(&self, query_embedding: &[f32], limit: usize) -> Result<Vec<SearchResult>> {
    let mut stmt = self.conn.prepare("SELECT * FROM code_chunks")?;
    // 遍历所有行，计算相似度...
}
```

**新实现（使用 vec0 KNN）**:
```rust
pub fn search(
    &self,
    query_embedding: &[f32],
    limit: usize,
    language: Option<&str>,
    paths: Option<&[String]>,
) -> Result<Vec<SearchResult>> {
    let embedding_bytes = unsafe {
        std::slice::from_raw_parts(
            query_embedding.as_ptr() as *const u8,
            query_embedding.len() * 4,
        )
    };

    let results = if let Some(lang) = language {
        // 使用分区索引（最快）
        self.conn.query_map(
            "SELECT file_path, language, content, start_line, end_line, distance
             FROM code_chunks_vec
             WHERE embedding MATCH ? AND k = ? AND language = ?
             ORDER BY distance",
            params![embedding_bytes, limit, lang],
            |row| {
                Ok(SearchResult {
                    file_path: row.get(0)?,
                    language: row.get(1)?,
                    content: row.get(2)?,
                    start_line: row.get(3)?,
                    end_line: row.get(4)?,
                    score: l2_to_cosine(row.get::<_, f32>(5)?),
                })
            },
        )?
    } else {
        // 全局搜索
        self.conn.query_map(
            "SELECT file_path, language, content, start_line, end_line, distance
             FROM code_chunks_vec
             WHERE embedding MATCH ? AND k = ?
             ORDER BY distance",
            params![embedding_bytes, limit],
            |row| { /* ... */ },
        )?
    };

    results.collect()
}

fn l2_to_cosine(l2_distance: f32) -> f32 {
    1.0 - l2_distance * l2_distance / 2.0
}
```

---

### 4.3 阶段 3：添加高级功能（P1）

#### 4.3.1 语言检测（功能 5）

**使用 cocoindex 的语言检测** (`utils/language.rs`):
```rust
use std::path::Path;

pub fn detect_language(path: &Path) -> Option<String> {
    let ext = path.extension()?.to_str()?;

    match ext {
        "py" => Some("python"),
        "js" | "mjs" | "cjs" => Some("javascript"),
        "ts" | "tsx" => Some("typescript"),
        "rs" => Some("rust"),
        "go" => Some("go"),
        "java" => Some("java"),
        "c" | "h" => Some("c"),
        "cpp" | "cc" | "cxx" | "hpp" => Some("cpp"),
        "cs" => Some("csharp"),
        "rb" => Some("ruby"),
        "php" => Some("php"),
        "swift" => Some("swift"),
        "kt" => Some("kotlin"),
        "scala" => Some("scala"),
        "sql" => Some("sql"),
        "sh" | "bash" => Some("bash"),
        "md" => Some("markdown"),
        _ => None,
    }.map(String::from)
}
```

#### 4.3.2 MCP 服务器（功能 6）

**选择 MCP SDK**：
- 调研 Rust MCP SDK 选项
- 或使用 JSON-RPC 手动实现

**MCP Tools 定义**:
```rust
// mcp/mod.rs
use serde::{Deserialize, Serialize};

#[derive(Serialize, Deserialize)]
pub struct SearchRequest {
    pub query: String,
    pub limit: Option<usize>,
    pub offset: Option<usize>,
    pub languages: Option<Vec<String>>,
    pub paths: Option<Vec<String>>,
    pub refresh_index: Option<bool>,
}

#[derive(Serialize, Deserialize)]
pub struct SearchResponse {
    pub success: bool,
    pub results: Vec<SearchResult>,
    pub total_returned: usize,
    pub offset: usize,
    pub message: Option<String>,
}

pub async fn run_mcp_server(
    store: Store,
    provider: Provider,
    config: Config,
) -> Result<()> {
    // TODO: 实现 MCP 服务器
    // 1. 监听 stdio
    // 2. 解析 JSON-RPC 请求
    // 3. 调用相应的工具函数
    // 4. 返回 JSON-RPC 响应
    Ok(())
}
```

#### 4.3.3 语言/路径过滤（功能 10）

**在搜索中添加过滤** (`store/mod.rs`):
```rust
pub fn search(
    &self,
    query_embedding: &[f32],
    limit: usize,
    languages: Option<&[String]>,
    paths: Option<&[String]>,
) -> Result<Vec<SearchResult>> {
    let mut sql = String::from(
        "SELECT file_path, language, content, start_line, end_line, distance
         FROM code_chunks_vec
         WHERE embedding MATCH ? AND k = ?"
    );

    let mut params: Vec<Box<dyn rusqlite::ToSql>> = vec![
        Box::new(embedding_bytes.to_vec()),
        Box::new(limit),
    ];

    // 语言过滤
    if let Some(langs) = languages {
        let placeholders = langs.iter().map(|_| "?").collect::<Vec<_>>().join(",");
        sql.push_str(&format!(" AND language IN ({})", placeholders));
        for lang in langs {
            params.push(Box::new(lang.clone()));
        }
    }

    // 路径过滤（使用 GLOB）
    if let Some(path_patterns) = paths {
        let conditions = path_patterns
            .iter()
            .map(|_| "file_path GLOB ?")
            .collect::<Vec<_>>()
            .join(" OR ");
        sql.push_str(&format!(" AND ({})", conditions));
        for pattern in path_patterns {
            params.push(Box::new(pattern.clone()));
        }
    }

    sql.push_str(" ORDER BY distance");

    // 执行查询...
}
```

---

### 4.4 阶段 4：完善和优化（P2）

#### 4.4.1 语言覆盖（功能 14）

**在语言检测中应用覆盖** (`utils/language.rs`):
```rust
pub fn detect_language_with_overrides(
    path: &Path,
    overrides: &HashMap<String, String>,
) -> Option<String> {
    let ext = path.extension()?.to_str()?;

    // 先检查覆盖规则
    if let Some(lang) = overrides.get(ext) {
        return Some(lang.clone());
    }

    // 再使用默认检测
    detect_language(path)
}
```

#### 4.4.2 CLI 改进

**添加更多命令** (`main.rs`):
```rust
#[derive(Subcommand)]
enum Commands {
    /// Index a project directory
    Index {
        #[arg(value_name = "PATH")]
        path: PathBuf,
    },
    /// Search code in the index
    Search {
        #[arg(value_name = "QUERY")]
        query: String,
        #[arg(long)]
        languages: Option<Vec<String>>,
        #[arg(long)]
        paths: Option<Vec<String>>,
        #[arg(long, default_value = "5")]
        limit: usize,
    },
    /// Start as MCP server
    Mcp,
    /// Show project status
    Status,
    /// Initialize project settings
    Init {
        #[arg(value_name = "PATH")]
        path: Option<PathBuf>,
    },
}
```

---

## 5. 实施步骤

### 第 1 周：基础修复
- [ ] 修复 Cargo.toml 依赖（使用本地路径）
- [ ] 修复分块 API 调用
- [ ] 实现配置系统（UserSettings + ProjectSettings）
- [ ] 实现文件模式匹配
- [ ] 验证基本索引和搜索功能

### 第 2 周：sqlite-vec 集成
- [ ] 调研 sqlite-vec Rust 绑定
- [ ] 修改数据库 Schema
- [ ] 实现 vec0 KNN 搜索
- [ ] 性能测试和对比

### 第 3 周：高级功能
- [ ] 实现语言检测
- [ ] 实现 MCP 服务器
- [ ] 添加语言/路径过滤
- [ ] 完善 CLI 命令

### 第 4 周：测试和文档
- [ ] 端到端测试
- [ ] 性能基准测试
- [ ] 编写用户文档
- [ ] 编写开发文档

---

## 6. 性能目标

| 指标 | Python 原版 | Rust 目标 |
|------|------------|----------|
| 索引速度 | 100 files/s | 200+ files/s |
| 搜索延迟 | 10-50ms | 5-20ms |
| 内存占用 | 500MB+ | 100-200MB |
| 二进制大小 | N/A | 8-12MB |
| 启动时间 | 1-2s | <100ms |

---

## 7. 测试计划

### 7.1 单元测试
- 配置加载/保存
- 文件模式匹配
- 语言检测
- 向量搜索

### 7.2 集成测试
- 完整索引流程
- 增量更新
- 搜索准确性
- MCP 协议

### 7.3 性能测试
- 大型代码库索引（10,000+ 文件）
- 并发搜索
- 内存泄漏检测

---

## 8. 发布计划

### 8.1 Alpha 版本（v0.1.0）
- 基本索引和搜索功能
- 配置系统
- CLI 命令

### 8.2 Beta 版本（v0.2.0）
- sqlite-vec 集成
- MCP 服务器
- 完整功能

### 8.3 正式版本（v1.0.0）
- 性能优化
- 完整文档
- 生产就绪

---

## 9. 参考资料

- Python 原版：`src/cocoindex_code/`
- Go 版本：`cocoindex-go/`
- cocoindex 核心库：`cocoindex/rust/`
- sqlite-vec 文档：https://github.com/asg017/sqlite-vec
- MCP 协议：https://modelcontextprotocol.io/

---

## 10. 附录

### 10.1 配置文件示例

**用户配置** (`~/.cocoindex_code/settings.yml`):
```yaml
api_key: sk-xxx
api_base: https://api.openai.com/v1
model: text-embedding-3-small
envs:
  CUSTOM_VAR: value
```

**项目配置** (`项目/.cocoindex_code/settings.yml`):
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
language_overrides:
  inc: php
  tpl: html
  conf: toml
```

### 10.2 CLI 使用示例

```bash
# 初始化项目
cocoindex-rs init

# 索引项目
cocoindex-rs index /path/to/project

# 搜索代码
cocoindex-rs search "authentication logic"

# 带过滤的搜索
cocoindex-rs search "database connection" \
  --languages rust python \
  --paths "src/**" \
  --limit 10

# 查看状态
cocoindex-rs status

# 启动 MCP 服务器
cocoindex-rs mcp
```

### 10.3 MCP 配置示例

**Claude Desktop** (`claude_desktop_config.json`):
```json
{
  "mcpServers": {
    "cocoindex-rs": {
      "command": "/path/to/cocoindex-rs",
      "args": ["mcp"],
      "env": {
        "OPENAI_API_KEY": "sk-xxx"
      }
    }
  }
}
```
