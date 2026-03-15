#!/bin/bash
# MCP 服务器测试脚本

echo "=== CocoIndex-RS MCP 服务器测试 ==="
echo ""

# 设置环境变量
export OPENAI_API_KEY="test-key"
export OPENAI_API_BASE="https://api.openai.com/v1"
export EMBEDDING_MODEL="text-embedding-3-small"

MCP_SERVER="./cocoindex-rs/target/release/coco-rs"

if [ ! -f "$MCP_SERVER" ]; then
    echo "❌ 错误: 找不到 coco-rs 二进制文件"
    exit 1
fi

echo "✓ 找到 MCP 服务器: $MCP_SERVER"
echo ""

# 测试 1: 初始化
echo "测试 1: 发送 initialize 请求"
echo '{"jsonrpc":"2.0","method":"initialize","params":{},"id":1}' | $MCP_SERVER mcp 2>/dev/null | head -1 | jq .
echo ""

# 测试 2: 列出工具
echo "测试 2: 发送 tools/list 请求"
echo '{"jsonrpc":"2.0","method":"tools/list","params":{},"id":2}' | $MCP_SERVER mcp 2>/dev/null | head -1 | jq '.result.tools[] | {name, description}'
echo ""

# 测试 3: 调用工具（会失败，因为没有真实的 API key）
echo "测试 3: 测试 tools/call 请求格式"
echo '{"jsonrpc":"2.0","method":"tools/call","params":{"name":"search_code","arguments":{"query":"test"}},"id":3}' | $MCP_SERVER mcp 2>/dev/null | head -1 | jq -r '.result.content[0].text' | head -5
echo ""

echo "=== 测试完成 ==="
