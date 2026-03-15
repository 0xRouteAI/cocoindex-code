use serde::{Deserialize, Serialize};
use serde_json::json;
use tokio::io::{self, AsyncBufReadExt, AsyncReadExt, AsyncWriteExt, BufReader};
use crate::{Indexer, Store, Provider};
use std::sync::Arc;
use std::path::Path;

#[derive(Debug, Deserialize, Serialize)]
struct JsonRpcRequest {
    jsonrpc: String,
    method: String,
    params: Option<serde_json::Value>,
    id: Option<serde_json::Value>,
}

#[derive(Debug, Deserialize, Serialize)]
struct JsonRpcResponse {
    jsonrpc: String,
    result: Option<serde_json::Value>,
    error: Option<serde_json::Value>,
    id: Option<serde_json::Value>,
}

pub async fn run(store: Arc<Store>, provider: Arc<Provider>) -> anyhow::Result<()> {
    let stdin = io::stdin();
    let mut stdout = io::stdout();
    let mut reader = BufReader::new(stdin);

    loop {
        // Read Content-Length header
        let mut headers = String::new();
        loop {
            let mut line = String::new();
            if reader.read_line(&mut line).await? == 0 {
                return Ok(()); // EOF
            }

            if line == "\r\n" || line == "\n" {
                break; // End of headers
            }
            headers.push_str(&line);
        }

        // Parse Content-Length
        let content_length = headers
            .lines()
            .find(|line| line.to_lowercase().starts_with("content-length:"))
            .and_then(|line| {
                line.split(':')
                    .nth(1)
                    .and_then(|s| s.trim().parse::<usize>().ok())
            });

        let content_length = match content_length {
            Some(len) => len,
            None => continue, // Invalid request, skip
        };

        // Read body
        let mut body = vec![0u8; content_length];
        reader.read_exact(&mut body).await?;

        let body_str = String::from_utf8_lossy(&body);
        let req: JsonRpcRequest = match serde_json::from_str(&body_str) {
            Ok(r) => r,
            Err(e) => {
                // Send parse error
                let error_response = JsonRpcResponse {
                    jsonrpc: "2.0".to_string(),
                    result: None,
                    error: Some(json!({
                        "code": -32700,
                        "message": format!("Parse error: {}", e)
                    })),
                    id: None,
                };
                write_response(&mut stdout, &error_response).await?;
                continue;
            }
        };

        // Handle request
        let res = handle_request(req, &store, &provider).await;
        write_response(&mut stdout, &res).await?;
    }
}

async fn write_response(
    stdout: &mut io::Stdout,
    response: &JsonRpcResponse,
) -> anyhow::Result<()> {
    let json = serde_json::to_string(response)?;
    let content_length = json.len();

    // Write Content-Length header
    stdout.write_all(format!("Content-Length: {}\r\n\r\n", content_length).as_bytes()).await?;
    // Write body
    stdout.write_all(json.as_bytes()).await?;
    stdout.flush().await?;

    Ok(())
}

async fn handle_request(req: JsonRpcRequest, store: &Store, provider: &Provider) -> JsonRpcResponse {
    let result = match req.method.as_str() {
        "initialize" => {
            Some(json!({
                "protocolVersion": "2024-11-05",
                "capabilities": {
                    "tools": {
                        "listChanged": false
                    }
                },
                "serverInfo": {
                    "name": "cocoindex-rs",
                    "version": "0.1.0"
                }
            }))
        }
        "initialized" => {
            // Notification, no response needed
            return JsonRpcResponse {
                jsonrpc: "2.0".to_string(),
                result: Some(json!({})),
                error: None,
                id: req.id,
            };
        }
        "shutdown" => {
            Some(json!(null))
        }
        "tools/list" => {
            Some(json!({
                "tools": [
                    {
                        "name": "index_project",
                        "description": "Index a project directory for code search. Supports incremental updates based on file hashes.",
                        "inputSchema": {
                            "type": "object",
                            "properties": {
                                "path": {
                                    "type": "string",
                                    "description": "Path to the project directory to index"
                                },
                                "refresh_index": {
                                    "type": "boolean",
                                    "description": "Force re-indexing even if files haven't changed",
                                    "default": false
                                }
                            },
                            "required": ["path"]
                        }
                    },
                    {
                        "name": "search_code",
                        "description": "Search code snippets using semantic similarity. Supports language and path filtering.",
                        "inputSchema": {
                            "type": "object",
                            "properties": {
                                "query": {
                                    "type": "string",
                                    "description": "Natural language search query"
                                },
                                "limit": {
                                    "type": "integer",
                                    "description": "Maximum number of results to return",
                                    "default": 10,
                                    "minimum": 1,
                                    "maximum": 100
                                },
                                "offset": {
                                    "type": "integer",
                                    "description": "Number of results to skip (for pagination)",
                                    "default": 0,
                                    "minimum": 0
                                },
                                "languages": {
                                    "type": "array",
                                    "items": { "type": "string" },
                                    "description": "Filter by programming languages (e.g., ['rust', 'python'])"
                                },
                                "paths": {
                                    "type": "array",
                                    "items": { "type": "string" },
                                    "description": "Filter by file path patterns (GLOB syntax, e.g., ['src/**/*.rs'])"
                                }
                            },
                            "required": ["query"]
                        }
                    }
                ]
            }))
        }
        "tools/call" => {
            let params = req.params.as_ref();
            if params.is_none() {
                return JsonRpcResponse {
                    jsonrpc: "2.0".to_string(),
                    result: None,
                    error: Some(json!({
                        "code": -32602,
                        "message": "Invalid params: missing params"
                    })),
                    id: req.id,
                };
            }

            let params = params.unwrap();
            let name = params.get("name").and_then(|v| v.as_str());
            if name.is_none() {
                return JsonRpcResponse {
                    jsonrpc: "2.0".to_string(),
                    result: None,
                    error: Some(json!({
                        "code": -32602,
                        "message": "Invalid params: missing tool name"
                    })),
                    id: req.id,
                };
            }

            let name = name.unwrap();
            let args = params.get("arguments").cloned().unwrap_or(json!({}));

            match name {
                "index_project" => {
                    let path = args.get("path").and_then(|v| v.as_str()).unwrap_or(".");
                    let refresh = args.get("refresh_index").and_then(|v| v.as_bool()).unwrap_or(false);
                    let path_obj = Path::new(path);

                    if !path_obj.exists() {
                        return JsonRpcResponse {
                            jsonrpc: "2.0".to_string(),
                            result: Some(json!({
                                "isError": true,
                                "content": [{
                                    "type": "text",
                                    "text": format!("Error: Path '{}' does not exist", path)
                                }]
                            })),
                            error: None,
                            id: req.id,
                        };
                    }

                    match Indexer::new(store.clone_internal(), provider.clone_internal(), path_obj) {
                        Ok(indexer) => {
                            match indexer.index_directory_with_refresh(path_obj, refresh).await {
                                Ok(_) => Some(json!({
                                    "content": [{
                                        "type": "text",
                                        "text": format!("✓ Successfully indexed project at '{}'", path)
                                    }]
                                })),
                                Err(e) => Some(json!({
                                    "isError": true,
                                    "content": [{
                                        "type": "text",
                                        "text": format!("Error during indexing: {}", e)
                                    }]
                                })),
                            }
                        }
                        Err(e) => Some(json!({
                            "isError": true,
                            "content": [{
                                "type": "text",
                                "text": format!("Error creating indexer: {}", e)
                            }]
                        })),
                    }
                }
                "search_code" => {
                    let query = args.get("query").and_then(|v| v.as_str()).unwrap_or_default();

                    if query.is_empty() {
                        return JsonRpcResponse {
                            jsonrpc: "2.0".to_string(),
                            result: Some(json!({
                                "isError": true,
                                "content": [{
                                    "type": "text",
                                    "text": "Error: Query cannot be empty"
                                }]
                            })),
                            error: None,
                            id: req.id,
                        };
                    }

                    let limit = args.get("limit").and_then(|v| v.as_u64()).unwrap_or(10) as usize;
                    let offset = args.get("offset").and_then(|v| v.as_u64()).unwrap_or(0) as usize;

                    let languages: Option<Vec<String>> = args.get("languages")
                        .and_then(|v| v.as_array())
                        .map(|arr| arr.iter().filter_map(|v| v.as_str().map(String::from)).collect());

                    let paths: Option<Vec<String>> = args.get("paths")
                        .and_then(|v| v.as_array())
                        .map(|arr| arr.iter().filter_map(|v| v.as_str().map(String::from)).collect());

                    match provider.get_embeddings(vec![query.to_string()]).await {
                        Ok(embeddings) => {
                            let embedding = match embeddings.into_iter().next() {
                                Some(e) => e,
                                None => {
                                    return JsonRpcResponse {
                                        jsonrpc: "2.0".to_string(),
                                        result: Some(json!({
                                            "isError": true,
                                            "content": [{
                                                "type": "text",
                                                "text": "Error: API returned empty embeddings"
                                            }]
                                        })),
                                        error: None,
                                        id: req.id,
                                    };
                                }
                            };

                            match store.search(
                                &embedding,
                                limit,
                                offset,
                                languages.as_deref(),
                                paths.as_deref(),
                            ).await {
                                Ok(results) => {
                                    if results.is_empty() {
                                        Some(json!({
                                            "content": [{
                                                "type": "text",
                                                "text": "No results found. Try a different query or check if the project is indexed."
                                            }]
                                        }))
                                    } else {
                                        let mut output = format!("Found {} result(s):\n\n", results.len());
                                        for (i, r) in results.iter().enumerate() {
                                            output.push_str(&format!(
                                                "{}. {} (Lines {}-{}, Score: {:.3})\n",
                                                i + 1 + offset,
                                                r.file_path,
                                                r.start_line,
                                                r.end_line,
                                                r.score
                                            ));
                                            if let Some(lang) = &r.language {
                                                output.push_str(&format!("   Language: {}\n", lang));
                                            }
                                            output.push_str(&format!("```\n{}\n```\n\n", r.content));
                                        }
                                        Some(json!({
                                            "content": [{
                                                "type": "text",
                                                "text": output
                                            }]
                                        }))
                                    }
                                }
                                Err(e) => Some(json!({
                                    "isError": true,
                                    "content": [{
                                        "type": "text",
                                        "text": format!("Search error: {}", e)
                                    }]
                                })),
                            }
                        }
                        Err(e) => Some(json!({
                            "isError": true,
                            "content": [{
                                "type": "text",
                                "text": format!("Embedding error: {}", e)
                            }]
                        })),
                    }
                }
                _ => {
                    return JsonRpcResponse {
                        jsonrpc: "2.0".to_string(),
                        result: None,
                        error: Some(json!({
                            "code": -32601,
                            "message": format!("Unknown tool: {}", name)
                        })),
                        id: req.id,
                    };
                }
            }
        }
        _ => {
            return JsonRpcResponse {
                jsonrpc: "2.0".to_string(),
                result: None,
                error: Some(json!({
                    "code": -32601,
                    "message": format!("Method not found: {}", req.method)
                })),
                id: req.id,
            };
        }
    };

    JsonRpcResponse {
        jsonrpc: "2.0".to_string(),
        result,
        error: None,
        id: req.id,
    }
}
