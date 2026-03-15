# CocoIndex-RS Official UX Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make `coco-rs` behave like official `cocoindex-code`: one global config file, no required project init, automatic project root discovery, automatic indexing for small projects, manual indexing for large projects.

**Architecture:** Keep the current Rust binary and MCP server structure, but move configuration and root resolution to a global-first model. Add a lightweight project probe and an MCP-side auto-index policy so `mcp` becomes usable immediately when launched inside a project without requiring `init`.

**Tech Stack:** Rust, Tokio, clap, rusqlite/sqlite-vec, serde/serde_yaml, existing MCP stdio transport

---

### Task 1: Expand Global Settings Model

**Files:**
- Modify: `cocoindex-rs/src/config/user.rs`
- Modify: `cocoindex-rs/src/config/mod.rs`
- Test: `cocoindex-rs/tests/integration_tests.rs`

**Step 1: Write the failing test**

Add tests that deserialize a user settings YAML containing:
- `api_key`
- `api_base`
- `model`
- `embedding_dim`
- `auto_index_small_projects`
- `refresh_on_search`
- `small_project_max_files`
- `small_project_max_bytes`
- `root_markers`
- `excluded_patterns`
- `extra_extensions`

Expected behavior:
- all fields deserialize
- defaults are applied for optional automation settings
- missing `embedding_dim` is treated as invalid for MCP startup

**Step 2: Run test to verify it fails**

Run: `cargo test integration_tests::test_user_settings_deserialize -- --nocapture`

Expected:
- FAIL because fields or validation do not exist yet

**Step 3: Write minimal implementation**

Update `UserSettings` to include:
- `embedding_dim: usize`
- `auto_index_small_projects: bool`
- `refresh_on_search: bool`
- `small_project_max_files: usize`
- `small_project_max_bytes: u64`
- `root_markers: Vec<String>`
- `excluded_patterns: Vec<String>`
- `extra_extensions: Vec<String>`

Add sensible defaults:
- `auto_index_small_projects = true`
- `refresh_on_search = true`
- `small_project_max_files = 300`
- `small_project_max_bytes = 10 * 1024 * 1024`
- `root_markers = [".cocoindex_code", ".git"]`

Keep `embedding_dim` required in effective runtime config.

**Step 4: Run test to verify it passes**

Run: `cargo test integration_tests::test_user_settings_deserialize -- --nocapture`

Expected:
- PASS

**Step 5: Commit**

```bash
git add cocoindex-rs/src/config/user.rs cocoindex-rs/src/config/mod.rs cocoindex-rs/tests/integration_tests.rs
git commit -m "feat: expand global cocoindex settings"
```

### Task 2: Add Project Root Discovery Utility

**Files:**
- Create: `cocoindex-rs/src/utils/project_root.rs`
- Modify: `cocoindex-rs/src/utils/mod.rs`
- Test: `cocoindex-rs/tests/integration_tests.rs`

**Step 1: Write the failing test**

Add tests for:
- nested directory under a repo with `.git`
- nested directory under a repo with `.cocoindex_code`
- directory with neither marker

Expected behavior:
- returns nearest ancestor with configured marker
- falls back to cwd if no marker exists

**Step 2: Run test to verify it fails**

Run: `cargo test integration_tests::test_discover_project_root -- --nocapture`

Expected:
- FAIL because utility does not exist yet

**Step 3: Write minimal implementation**

Implement `discover_project_root(start: &Path, markers: &[String]) -> PathBuf`:
- canonicalize input when possible
- walk parent chain upward
- for each parent, check whether any marker path exists
- return first matching parent
- otherwise return start path

Export the utility from `utils/mod.rs`.

**Step 4: Run test to verify it passes**

Run: `cargo test integration_tests::test_discover_project_root -- --nocapture`

Expected:
- PASS

**Step 5: Commit**

```bash
git add cocoindex-rs/src/utils/project_root.rs cocoindex-rs/src/utils/mod.rs cocoindex-rs/tests/integration_tests.rs
git commit -m "feat: add project root discovery"
```

### Task 3: Add Project Size Probe

**Files:**
- Create: `cocoindex-rs/src/utils/project_probe.rs`
- Modify: `cocoindex-rs/src/utils/mod.rs`
- Modify: `cocoindex-rs/src/config/project.rs`
- Test: `cocoindex-rs/tests/integration_tests.rs`

**Step 1: Write the failing test**

Add tests that create:
- a small temporary project
- a larger temporary project exceeding configured thresholds

Expected behavior:
- probe returns indexed file count
- probe returns total bytes
- probe classifies project as small/large using thresholds

**Step 2: Run test to verify it fails**

Run: `cargo test integration_tests::test_project_probe -- --nocapture`

Expected:
- FAIL because no probe exists yet

**Step 3: Write minimal implementation**

Implement a probe that:
- loads include/exclude rules from `ProjectSettings` plus global `excluded_patterns`
- includes any `extra_extensions` from global settings
- scans candidate files once
- returns:
  - `file_count`
  - `total_bytes`
  - `is_small_project`

Do not trigger embeddings or indexing here.

**Step 4: Run test to verify it passes**

Run: `cargo test integration_tests::test_project_probe -- --nocapture`

Expected:
- PASS

**Step 5: Commit**

```bash
git add cocoindex-rs/src/utils/project_probe.rs cocoindex-rs/src/utils/mod.rs cocoindex-rs/src/config/project.rs cocoindex-rs/tests/integration_tests.rs
git commit -m "feat: add project size probe"
```

### Task 4: Remove `init` From Normal Runtime Flow

**Files:**
- Modify: `cocoindex-rs/src/main.rs`
- Modify: `cocoindex-rs/src/config/project.rs`
- Test: `cocoindex-rs/tests/integration_tests.rs`

**Step 1: Write the failing test**

Add tests for:
- running search/status logic in a project with no `.cocoindex_code/settings.yml`
- runtime should still work using defaults

Expected behavior:
- no `init` required
- project defaults are applied implicitly

**Step 2: Run test to verify it fails**

Run: `cargo test integration_tests::test_no_init_required -- --nocapture`

Expected:
- FAIL because code path still assumes init-oriented workflow or commands

**Step 3: Write minimal implementation**

Adjust CLI and runtime behavior:
- keep `init` only as optional advanced command or remove it from docs
- make `index`, `status`, and `mcp` work with implicit project defaults
- ensure project-level settings file remains optional

**Step 4: Run test to verify it passes**

Run: `cargo test integration_tests::test_no_init_required -- --nocapture`

Expected:
- PASS

**Step 5: Commit**

```bash
git add cocoindex-rs/src/main.rs cocoindex-rs/src/config/project.rs cocoindex-rs/tests/integration_tests.rs
git commit -m "refactor: make project init optional"
```

### Task 5: Auto-Discover Root For CLI Commands

**Files:**
- Modify: `cocoindex-rs/src/main.rs`
- Test: `cocoindex-rs/tests/integration_tests.rs`

**Step 1: Write the failing test**

Add tests that run search/status/index from nested subdirectories under a project root.

Expected behavior:
- root is discovered automatically
- DB path resolves to `<project_root>/.cocoindex_code/target_sqlite.db`

**Step 2: Run test to verify it fails**

Run: `cargo test integration_tests::test_cli_uses_discovered_root -- --nocapture`

Expected:
- FAIL because commands still use `current_dir()` directly

**Step 3: Write minimal implementation**

Update `main.rs`:
- resolve `cwd`
- load `UserSettings`
- compute `project_root = discover_project_root(cwd, settings.root_markers)`
- use `project_root` for `db_path`, `status`, `index`, and MCP startup

**Step 4: Run test to verify it passes**

Run: `cargo test integration_tests::test_cli_uses_discovered_root -- --nocapture`

Expected:
- PASS

**Step 5: Commit**

```bash
git add cocoindex-rs/src/main.rs cocoindex-rs/tests/integration_tests.rs
git commit -m "feat: auto-discover project root for cli"
```

### Task 6: Rename MCP Tools To Official-Style Interface

**Files:**
- Modify: `cocoindex-rs/src/mcp/mod.rs`
- Test: `cocoindex-rs/tests/integration_tests.rs`

**Step 1: Write the failing test**

Add tests for:
- `tools/list` returns `search`, `index`, `status`
- legacy names are absent or intentionally aliased

Expected behavior:
- names align with official UX

**Step 2: Run test to verify it fails**

Run: `cargo test integration_tests::mcp_tests::test_mcp_tools_list -- --nocapture`

Expected:
- FAIL because current names are `index_project` and `search_code`

**Step 3: Write minimal implementation**

Update MCP tool definitions and dispatch:
- `search`
- `index`
- `status`

Optionally keep old names as hidden aliases for compatibility, but do not advertise them.

**Step 4: Run test to verify it passes**

Run: `cargo test integration_tests::mcp_tests::test_mcp_tools_list -- --nocapture`

Expected:
- PASS

**Step 5: Commit**

```bash
git add cocoindex-rs/src/mcp/mod.rs cocoindex-rs/tests/integration_tests.rs
git commit -m "feat: align mcp tool names with official ux"
```

### Task 7: Add Small-Project Auto-Index On First Search

**Files:**
- Modify: `cocoindex-rs/src/mcp/mod.rs`
- Modify: `cocoindex-rs/src/indexer/mod.rs`
- Modify: `cocoindex-rs/src/store/mod.rs`
- Modify: `cocoindex-rs/src/main.rs`
- Test: `cocoindex-rs/tests/integration_tests.rs`

**Step 1: Write the failing test**

Add a real integration test:
- create a small project with no DB
- issue MCP `search`
- assert indexing happens automatically
- assert search returns results

**Step 2: Run test to verify it fails**

Run: `cargo test integration_tests::mcp_tests::test_auto_index_small_project -- --nocapture`

Expected:
- FAIL because search currently assumes preexisting index

**Step 3: Write minimal implementation**

Implement MCP search flow:
- discover root
- if DB missing:
  - run project probe
  - if small and `auto_index_small_projects` true, build index before search
- then run embeddings and search

Keep search flow synchronous for now; do not add background task complexity.

**Step 4: Run test to verify it passes**

Run: `cargo test integration_tests::mcp_tests::test_auto_index_small_project -- --nocapture`

Expected:
- PASS

**Step 5: Commit**

```bash
git add cocoindex-rs/src/mcp/mod.rs cocoindex-rs/src/indexer/mod.rs cocoindex-rs/src/store/mod.rs cocoindex-rs/src/main.rs cocoindex-rs/tests/integration_tests.rs
git commit -m "feat: auto-index small projects on first search"
```

### Task 8: Add Large-Project Manual Index Gate

**Files:**
- Modify: `cocoindex-rs/src/mcp/mod.rs`
- Modify: `cocoindex-rs/src/utils/project_probe.rs`
- Test: `cocoindex-rs/tests/integration_tests.rs`

**Step 1: Write the failing test**

Add a test:
- create a project exceeding thresholds
- ensure no DB exists
- issue MCP `search`
- assert server returns a manual-index instruction instead of starting indexing

**Step 2: Run test to verify it fails**

Run: `cargo test integration_tests::mcp_tests::test_large_project_requires_manual_index -- --nocapture`

Expected:
- FAIL because current code does not distinguish project size

**Step 3: Write minimal implementation**

If no DB exists and project is large:
- do not auto-index
- return a text response including:
  - file count
  - total bytes
  - threshold values
  - exact manual command: `coco-rs index`

**Step 4: Run test to verify it passes**

Run: `cargo test integration_tests::mcp_tests::test_large_project_requires_manual_index -- --nocapture`

Expected:
- PASS

**Step 5: Commit**

```bash
git add cocoindex-rs/src/mcp/mod.rs cocoindex-rs/src/utils/project_probe.rs cocoindex-rs/tests/integration_tests.rs
git commit -m "feat: require manual indexing for large projects"
```

### Task 9: Add Search-Time Incremental Refresh

**Files:**
- Modify: `cocoindex-rs/src/mcp/mod.rs`
- Modify: `cocoindex-rs/src/indexer/mod.rs`
- Test: `cocoindex-rs/tests/integration_tests.rs`

**Step 1: Write the failing test**

Add a test:
- index a small project
- modify a file
- issue MCP `search` with refresh enabled
- assert updated content is returned

**Step 2: Run test to verify it fails**

Run: `cargo test integration_tests::mcp_tests::test_search_refreshes_index -- --nocapture`

Expected:
- FAIL because search does not currently auto-refresh based on config

**Step 3: Write minimal implementation**

Before search:
- if `refresh_index` request arg is true, refresh
- otherwise if arg omitted and `refresh_on_search` is true, refresh
- use incremental indexing, not full refresh

**Step 4: Run test to verify it passes**

Run: `cargo test integration_tests::mcp_tests::test_search_refreshes_index -- --nocapture`

Expected:
- PASS

**Step 5: Commit**

```bash
git add cocoindex-rs/src/mcp/mod.rs cocoindex-rs/src/indexer/mod.rs cocoindex-rs/tests/integration_tests.rs
git commit -m "feat: refresh index automatically during search"
```

### Task 10: Add MCP `status` Tool

**Files:**
- Modify: `cocoindex-rs/src/mcp/mod.rs`
- Modify: `cocoindex-rs/src/store/mod.rs`
- Test: `cocoindex-rs/tests/integration_tests.rs`

**Step 1: Write the failing test**

Add test coverage for MCP `status`:
- no index present
- index present

Expected returned fields:
- `project_root`
- `db_path`
- `indexed`
- `model`
- `embedding_dim`

Optional:
- `file_count`
- `total_bytes`

**Step 2: Run test to verify it fails**

Run: `cargo test integration_tests::mcp_tests::test_mcp_status -- --nocapture`

Expected:
- FAIL because no status tool exists

**Step 3: Write minimal implementation**

Add `status` tool and response payload based on:
- discovered project root
- config
- whether DB exists
- basic probe metadata if cheap to compute

**Step 4: Run test to verify it passes**

Run: `cargo test integration_tests::mcp_tests::test_mcp_status -- --nocapture`

Expected:
- PASS

**Step 5: Commit**

```bash
git add cocoindex-rs/src/mcp/mod.rs cocoindex-rs/src/store/mod.rs cocoindex-rs/tests/integration_tests.rs
git commit -m "feat: add mcp status tool"
```

### Task 11: Enforce Global Config Validation At Startup

**Files:**
- Modify: `cocoindex-rs/src/main.rs`
- Modify: `cocoindex-rs/src/config/user.rs`
- Test: `cocoindex-rs/tests/integration_tests.rs`

**Step 1: Write the failing test**

Add tests for missing or invalid:
- `api_key`
- `api_base`
- `model`
- `embedding_dim`

Expected behavior:
- startup fails with precise message

**Step 2: Run test to verify it fails**

Run: `cargo test integration_tests::test_config_validation -- --nocapture`

Expected:
- FAIL because validation is not centralized yet

**Step 3: Write minimal implementation**

Add a validation function invoked by CLI/MCP startup:
- reject empty strings
- reject `embedding_dim == 0`
- produce actionable errors

**Step 4: Run test to verify it passes**

Run: `cargo test integration_tests::test_config_validation -- --nocapture`

Expected:
- PASS

**Step 5: Commit**

```bash
git add cocoindex-rs/src/main.rs cocoindex-rs/src/config/user.rs cocoindex-rs/tests/integration_tests.rs
git commit -m "feat: validate global api configuration"
```

### Task 12: Update README To Match Official UX

**Files:**
- Modify: `cocoindex-rs/README.md`

**Step 1: Write the failing doc checklist**

Create a manual verification checklist in the PR or notes:
- README no longer says project init is required
- README shows one global settings file
- README explains small vs large project behavior
- README documents Claude Code, Codex CLI, Gemini CLI command registration

**Step 2: Run doc review to verify current README fails**

Run:
```bash
rg -n "init|settings.yml|mcp|index" cocoindex-rs/README.md
```

Expected:
- README still reflects old project-init-first workflow

**Step 3: Write minimal documentation update**

Rewrite usage sections:
- installation
- global config
- MCP registration
- automatic indexing behavior
- manual indexing for large projects

**Step 4: Run doc review to verify it passes**

Run:
```bash
rg -n "global|small|large|Gemini|Codex|Claude" cocoindex-rs/README.md
```

Expected:
- updated README reflects new UX

**Step 5: Commit**

```bash
git add cocoindex-rs/README.md
git commit -m "docs: align readme with official-style ux"
```

### Task 13: Final Verification

**Files:**
- Test: `cocoindex-rs/tests/integration_tests.rs`
- Verify: `cocoindex-rs/src/main.rs`
- Verify: `cocoindex-rs/src/mcp/mod.rs`

**Step 1: Run focused test suites**

Run:
```bash
cargo test integration_tests -- --nocapture
```

Expected:
- all real integration tests pass

**Step 2: Run full test suite**

Run:
```bash
cargo test -- --nocapture
```

Expected:
- PASS with no placeholder-only tests

**Step 3: Run release build**

Run:
```bash
cargo build --release
```

Expected:
- PASS

**Step 4: Manual smoke test**

Run:
```bash
./target/release/coco-rs status
./target/release/coco-rs index
```

Expected:
- root discovery and config validation behave as documented

**Step 5: Commit**

```bash
git add -A
git commit -m "chore: finalize official ux flow"
```
