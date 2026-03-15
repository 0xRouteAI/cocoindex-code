#!/bin/bash
# MCP 服务器简单测试脚本（不依赖 jq）

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
echo '{"jsonrpc":"2.0","method":"initialize","params":{},"id":1}' | timeout 2 $MCP_SERVER mcp 2>/dev/null | head -1
echo ""

# 测试 2: 列出工具
echo "测试 2: 发送 tools/list 请求"
echo '{"jsonrpc":"2.0","method":"tools/list","params":{},"id":2}' | timeout 2 $MCP_SERVER mcp 2>/dev/null | head -1
echo ""

# 测试 3: 测试未知方法
echo "测试 3: 发送未知方法请求"
echo '{"jsonrpc":"2.0","method":"unknown_method","params":{},"id":3}' | timeout 2 $MCP_SERVER mcp 2>/dev/null | head -1
echo ""

echo "=== 测试完成 ==="
echo ""
echo "提示: 如果看到 JSON 响应，说明 MCP 服务器工作正常！"
