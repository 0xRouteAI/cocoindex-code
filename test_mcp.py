#!/usr/bin/env python3
"""MCP 服务器测试脚本"""

import subprocess
import json
import sys
import os

# 设置环境变量
os.environ['OPENAI_API_KEY'] = 'test-key'
os.environ['OPENAI_API_BASE'] = 'https://api.openai.com/v1'
os.environ['EMBEDDING_MODEL'] = 'text-embedding-3-small'

MCP_SERVER = './cocoindex-rs/target/release/coco-rs'

def send_request(request):
    """发送 JSON-RPC 请求到 MCP 服务器"""
    try:
        proc = subprocess.Popen(
            [MCP_SERVER, 'mcp'],
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True
        )

        # 发送请求
        stdout, stderr = proc.communicate(input=json.dumps(request) + '\n', timeout=5)

        if stdout:
            return json.loads(stdout.strip())
        return None
    except subprocess.TimeoutExpired:
        proc.kill()
        return None
    except Exception as e:
        print(f"错误: {e}")
        return None

def main():
    print("=== CocoIndex-RS MCP 服务器测试 ===\n")

    # 测试 1: Initialize
    print("测试 1: Initialize")
    response = send_request({
        "jsonrpc": "2.0",
        "method": "initialize",
        "params": {},
        "id": 1
    })

    if response:
        print(f"✓ 协议版本: {response.get('result', {}).get('protocolVersion')}")
        print(f"✓ 服务器名称: {response.get('result', {}).get('serverInfo', {}).get('name')}")
    else:
        print("✗ 初始化失败")
    print()

    # 测试 2: Tools List
    print("测试 2: 列出工具")
    response = send_request({
        "jsonrpc": "2.0",
        "method": "tools/list",
        "params": {},
        "id": 2
    })

    if response and 'result' in response:
        tools = response['result'].get('tools', [])
        print(f"✓ 找到 {len(tools)} 个工具:")
        for tool in tools:
            print(f"  - {tool['name']}: {tool['description'][:60]}...")
    else:
        print("✗ 获取工具列表失败")
    print()

    # 测试 3: 调用工具（search_code，会失败因为没有索引）
    print("测试 3: 调用 search_code 工具")
    response = send_request({
        "jsonrpc": "2.0",
        "method": "tools/call",
        "params": {
            "name": "search_code",
            "arguments": {
                "query": "test query"
            }
        },
        "id": 3
    })

    if response and 'result' in response:
        content = response['result'].get('content', [{}])[0].get('text', '')
        is_error = response['result'].get('isError', False)
        if is_error:
            print(f"✓ 工具调用返回预期错误: {content[:100]}")
        else:
            print(f"✓ 工具调用成功: {content[:100]}")
    else:
        print("✗ 工具调用失败")
    print()

    print("=== 测试完成 ===")
    print("\n如果所有测试都显示 ✓，说明 MCP 服务器工作正常！")

if __name__ == '__main__':
    main()
