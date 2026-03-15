package mcp

import (
	"context"
	"fmt"
	"os"

	"cocoindex-go/pkg/indexer"
	"cocoindex-go/pkg/provider"
	"cocoindex-go/pkg/store"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type McpServer struct {
	srv      *server.MCPServer
	indexer  *indexer.Indexer
	store    *store.Store
	provider *provider.CloudProvider
}

func NewMcpServer(idx *indexer.Indexer, s *store.Store, p *provider.CloudProvider) *McpServer {
	srv := server.NewMCPServer("cocoindex-go", "1.0.0")

	ms := &McpServer{
		srv:      srv,
		indexer:  idx,
		store:    s,
		provider: p,
	}

	// 注册 Tool: index_project
	srv.AddTool(mcp.NewTool("index_project",
		mcp.WithDescription("索引当前项目以支持语义搜索"),
		mcp.WithString("path", mcp.Description("项目根目录路径"), mcp.Required()),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.Params.Arguments.(map[string]interface{})
		path, ok := args["path"].(string)
		if !ok {
			return mcp.NewToolResultError("缺少 path 参数"), nil
		}
		err := idx.IndexProject(ctx, path)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("索引失败: %v", err)), nil
		}
		return mcp.NewToolResultText("项目索引完成"), nil
	})

	// 注册 Tool: search_code
	srv.AddTool(mcp.NewTool("search_code",
		mcp.WithDescription("使用自然语言搜索代码"),
		mcp.WithString("query", mcp.Description("搜索查询词"), mcp.Required()),
		mcp.WithNumber("limit", mcp.Description("返回结果数量限制，默认 5，最大 50")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.Params.Arguments.(map[string]interface{})
		query, ok := args["query"].(string)
		if !ok {
			return mcp.NewToolResultError("缺少 query 参数"), nil
		}

		limit := 5
		if limitVal, ok := args["limit"].(float64); ok {
			limit = int(limitVal)
			if limit <= 0 || limit > 50 {
				limit = 5
			}
		}

		vecs, err := p.Embed(ctx, []string{query})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Embedding 失败: %v", err)), nil
		}

		results, err := s.Search(vecs[0], limit)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("搜索数据库失败: %v", err)), nil
		}

		if len(results) == 0 {
			return mcp.NewToolResultText("未找到匹配的代码片段"), nil
		}

		var responseText string
		for _, r := range results {
			responseText += fmt.Sprintf("### 文件: %s (行 %d-%d)\n```\n%s\n```\n\n",
				r.FilePath, r.StartLine, r.EndLine, r.Content)
		}

		fmt.Fprintf(os.Stderr, "执行搜索: %s, 找到 %d 个结果\n", query, len(results))
		return mcp.NewToolResultText(responseText), nil
	})

	return ms
}

func (ms *McpServer) Serve() error {
	return server.ServeStdio(ms.srv)
}
