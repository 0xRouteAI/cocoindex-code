package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"cocoindex-go/pkg/indexer"
	"cocoindex-go/pkg/mcp"
	"cocoindex-go/pkg/provider"
	"cocoindex-go/pkg/store"
	"cocoindex-go/pkg/watcher"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	apiBase := os.Getenv("OPENAI_API_BASE")
	model := os.Getenv("EMBEDDING_MODEL")
	if model == "" {
		model = "text-embedding-3-small"
	}

	if apiKey == "" {
		fmt.Fprintf(os.Stderr, "错误: 请设置 OPENAI_API_KEY 环境变量\n")
		os.Exit(1)
	}

	prov := provider.NewCloudProvider(apiKey, apiBase, model)

	// 使用 XDG 标准或 HOME 目录存储数据库
	dbPath := os.Getenv("COCOINDEX_DB_PATH")
	if dbPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "获取用户目录失败: %v\n", err)
			os.Exit(1)
		}
		dataDir := filepath.Join(homeDir, ".cocoindex")
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "创建数据目录失败: %v\n", err)
			os.Exit(1)
		}
		dbPath = filepath.Join(dataDir, "coco_index.db")
	}

	st, err := store.NewStore(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化数据库失败: %v\n", err)
		os.Exit(1)
	}

	idx := indexer.NewIndexer(prov, st, 10)

	// 修复：显式检查 Watcher 初始化错误
	fw, err := watcher.NewFileWatcher(idx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化文件监控器失败: %v\n", err)
		os.Exit(1)
	}

	projectPath, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "获取当前路径失败: %v\n", err)
		os.Exit(1)
	}

	if err := fw.Watch(projectPath); err != nil {
		fmt.Fprintf(os.Stderr, "启动监控失败: %v\n", err)
		os.Exit(1)
	}

	server := mcp.NewMcpServer(idx, st, prov)
	
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Fprintf(os.Stderr, "\n正在关闭服务器...\n")
		os.Exit(0)
	}()

	fmt.Fprintf(os.Stderr, "CocoIndex Go MCP 启动成功!\n")
	fmt.Fprintf(os.Stderr, "使用模型: %s\n", model)
	fmt.Fprintf(os.Stderr, "监控路径: %s\n", projectPath)
	
	if err := server.Serve(); err != nil {
		fmt.Fprintf(os.Stderr, "服务器运行出错: %v\n", err)
	}
}
