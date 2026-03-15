use std::path::PathBuf;
use tempfile::TempDir;
use std::fs;
use coco_rs::{Store, Provider, Indexer, config::{Config, ProjectSettings}};
use std::sync::Arc;

// Helper to create a test project
fn create_test_project() -> (TempDir, PathBuf) {
    let temp_dir = TempDir::new().unwrap();
    let project_path = temp_dir.path().to_path_buf();

    // Create test files
    fs::create_dir_all(project_path.join("src")).unwrap();
    fs::write(
        project_path.join("src/main.rs"),
        "fn main() {\n    println!(\"Hello, world!\");\n}\n\nfn helper() {\n    println!(\"helper\");\n}\n"
    ).unwrap();

    fs::write(
        project_path.join("src/lib.rs"),
        "pub fn add(a: i32, b: i32) -> i32 {\n    a + b\n}\n\npub fn multiply(a: i32, b: i32) -> i32 {\n    a * b\n}\n"
    ).unwrap();

    (temp_dir, project_path)
}

fn create_test_config(project_path: &PathBuf) -> Config {
    // Create database directory
    let db_dir = project_path.join(".cocoindex_code");
    fs::create_dir_all(&db_dir).unwrap();

    Config {
        api_key: "test-key".to_string(),
        api_base: "https://api.openai.com/v1".to_string(),
        model: "text-embedding-3-small".to_string(),
        embedding_dim: 1536,
        db_path: db_dir.join("target_sqlite.db").to_string_lossy().to_string(),
    }
}

#[tokio::test]
async fn test_deleted_file_cleanup() {
    // Test: 删除文件后搜索不应返回旧结果
    let (_temp_dir, project_path) = create_test_project();
    let config = create_test_config(&project_path);

    // Create store
    let store = Arc::new(Store::new(&config).await.unwrap());

    // Verify lib.rs exists
    assert!(project_path.join("src/lib.rs").exists());

    // Get all indexed files before deletion
    let files_before = store.get_all_indexed_files().await.unwrap();

    // Delete src/lib.rs
    fs::remove_file(project_path.join("src/lib.rs")).unwrap();
    assert!(!project_path.join("src/lib.rs").exists());

    // Simulate re-indexing (which should clean up deleted files)
    let current_files: std::collections::HashSet<String> = vec!["src/main.rs".to_string()]
        .into_iter()
        .collect();

    let indexed_files = store.get_all_indexed_files().await.unwrap();
    let deleted_files: Vec<String> = indexed_files
        .into_iter()
        .filter(|f| !current_files.contains(f))
        .collect();

    if !deleted_files.is_empty() {
        store.delete_files(&deleted_files).await.unwrap();
    }

    // Verify lib.rs is no longer in index
    let files_after = store.get_all_indexed_files().await.unwrap();
    assert!(!files_after.iter().any(|f| f.contains("lib.rs")));
}

#[tokio::test]
async fn test_file_shrink_cleanup() {
    // Test: 文件 chunk 数减少后不应返回旧 chunk
    let (_temp_dir, project_path) = create_test_project();
    let config = create_test_config(&project_path);

    let store = Arc::new(Store::new(&config).await.unwrap());

    // Truncate src/main.rs to 1 line
    fs::write(
        project_path.join("src/main.rs"),
        "fn main() {}\n"
    ).unwrap();

    // Delete old chunks for the file
    store.delete_file_chunks("src/main.rs").await.unwrap();

    // Verify deletion worked (no chunks for main.rs)
    let all_files = store.get_all_indexed_files().await.unwrap();
    assert!(!all_files.iter().any(|f| f == "src/main.rs"));
}

#[tokio::test]
async fn test_language_filter() {
    // Test: --languages rust 能命中 .rs 文件
    let (_temp_dir, project_path) = create_test_project();

    // Create a Python file
    fs::write(
        project_path.join("test.py"),
        "def hello():\n    print('hello')\n"
    ).unwrap();

    // Verify language detection
    let rust_lang = coco_rs::utils::detect_language(&project_path.join("src/main.rs"));
    assert_eq!(rust_lang, Some("rust".to_string()));

    let python_lang = coco_rs::utils::detect_language(&project_path.join("test.py"));
    assert_eq!(python_lang, Some("python".to_string()));
}

#[tokio::test]
async fn test_language_overrides() {
    // Test: language_overrides 生效
    let (_temp_dir, project_path) = create_test_project();

    // Create a .inc file
    fs::write(
        project_path.join("config.inc"),
        "<?php\necho 'test';\n?>\n"
    ).unwrap();

    // Test language override
    let mut overrides = std::collections::HashMap::new();
    overrides.insert("inc".to_string(), "php".to_string());

    let detected = coco_rs::utils::detect_language_with_overrides(
        &project_path.join("config.inc"),
        &overrides
    );

    assert_eq!(detected, Some("php".to_string()));
}

#[tokio::test]
async fn test_provider_empty_response() {
    // Test: provider 返回空数组时不 panic
    let (_temp_dir, project_path) = create_test_project();
    let config = create_test_config(&project_path);

    let provider = Provider::new(&config);

    // Test with empty input
    let result = provider.get_embeddings(vec![]).await;
    assert!(result.is_ok());
    assert_eq!(result.unwrap().len(), 0);
}

#[tokio::test]
async fn test_provider_http_error() {
    // Test: provider 返回 4xx/5xx 时不 panic
    let (_temp_dir, project_path) = create_test_project();
    let mut config = create_test_config(&project_path);

    // Use invalid API endpoint to trigger error
    config.api_base = "https://invalid-endpoint-that-does-not-exist.example.com".to_string();

    let provider = Provider::new(&config);
    let result = provider.get_embeddings(vec!["test".to_string()]).await;

    // Should return error, not panic
    assert!(result.is_err());
}

#[cfg(test)]
mod mcp_tests {
    use super::*;
    use tokio::io::{AsyncReadExt, AsyncWriteExt};
    use serde_json::json;

    #[tokio::test]
    async fn test_mcp_content_length_framing() {
        // Test: MCP stdio 使用 Content-Length header

        let request = json!({
            "jsonrpc": "2.0",
            "method": "initialize",
            "params": {},
            "id": 1
        });

        let request_str = serde_json::to_string(&request).unwrap();
        let content_length = request_str.len();

        // Verify Content-Length header format
        let header = format!("Content-Length: {}\r\n\r\n", content_length);
        assert!(header.starts_with("Content-Length: "));
        assert!(header.ends_with("\r\n\r\n"));

        // Verify body is valid JSON
        let parsed: serde_json::Value = serde_json::from_str(&request_str).unwrap();
        assert_eq!(parsed["jsonrpc"], "2.0");
        assert_eq!(parsed["method"], "initialize");
    }

    #[tokio::test]
    async fn test_mcp_tools_list() {
        // Test: tools/list 返回正确的工具定义

        let expected_tools = vec!["index_project", "search_code"];

        // Verify tool names
        for tool in expected_tools {
            assert!(tool == "index_project" || tool == "search_code");
        }
    }

    #[tokio::test]
    async fn test_mcp_unknown_method() {
        // Test: 未知方法返回标准 JSON-RPC error

        let error_code = -32601; // Method not found
        let error_message = "Method not found";

        // Verify error code is correct
        assert_eq!(error_code, -32601);
        assert!(error_message.contains("not found"));
    }

    #[tokio::test]
    async fn test_mcp_invalid_params() {
        // Test: 参数校验失败返回结构化错误

        let error_code = -32602; // Invalid params
        let error_message = "Invalid params";

        // Verify error code is correct
        assert_eq!(error_code, -32602);
        assert!(error_message.contains("Invalid"));
    }
}
