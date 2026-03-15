package indexer

import (
	"context"
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"cocoindex-go/pkg/chunker"
	"cocoindex-go/pkg/common"
	"cocoindex-go/pkg/provider"
	"cocoindex-go/pkg/store"
)

type Indexer struct {
	provider *provider.CloudProvider
	store    *store.Store
	workers  int
}

func NewIndexer(p *provider.CloudProvider, s *store.Store, workers int) *Indexer {
	if workers <= 0 {
		workers = 4
	}
	return &Indexer{
		provider: p,
		store:    s,
		workers:  workers,
	}
}

func (idx *Indexer) IndexProject(ctx context.Context, rootPath string) error {
	fileChan := make(chan string, 100)
	errChan := make(chan error, 1)

	go func() {
		defer close(fileChan)
		err := filepath.WalkDir(rootPath, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				name := d.Name()
				if name == ".git" || name == "node_modules" || name == "vendor" {
					return filepath.SkipDir
				}
				return nil
			}
			ext := strings.ToLower(filepath.Ext(path))
			if common.IsSupportedExt(ext) {
				fileChan <- path
			}
			return nil
		})
		if err != nil {
			errChan <- err
		}
	}()

	var wg sync.WaitGroup
	for i := 0; i < idx.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range fileChan {
				if err := idx.IndexFile(ctx, path); err != nil {
					fmt.Fprintf(os.Stderr, "索引文件失败 %s: %v\n", path, err)
				}
			}
		}()
	}

	wg.Wait()
	select {
	case err := <-errChan:
		return err
	default:
		return nil
	}
}

func (idx *Indexer) IndexFile(ctx context.Context, path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return idx.store.DeleteFile(path)
		}
		return err
	}

	currentHash := fmt.Sprintf("%x", md5.Sum(content))
	oldHash, _ := idx.store.GetFileHash(path)
	if currentHash == oldHash {
		return nil
	}

	chk, err := chunker.NewChunker(path)
	if err != nil {
		return err
	}

	chunks, err := chk.Split(content)
	if err != nil {
		return err
	}

	if len(chunks) == 0 {
		return idx.store.DeleteFile(path)
	}

	// 批处理：每次最多处理 100 个 chunk
	const batchSize = 100
	allEmbeddings := make([][]float32, 0, len(chunks))

	for i := 0; i < len(chunks); i += batchSize {
		end := i + batchSize
		if end > len(chunks) {
			end = len(chunks)
		}

		texts := make([]string, end-i)
		for j := i; j < end; j++ {
			texts[j-i] = chunks[j].Content
		}

		embeddings, err := idx.provider.Embed(ctx, texts)
		if err != nil {
			return fmt.Errorf("批次 %d-%d embedding 失败: %w", i, end, err)
		}

		if len(embeddings) < len(texts) {
			return fmt.Errorf("批次 %d-%d embedding 返回数量不足", i, end)
		}

		allEmbeddings = append(allEmbeddings, embeddings...)
	}

	dbChunks := make([]store.ChunkData, len(chunks))

	for i, c := range chunks {
		id := fmt.Sprintf("%x", md5.Sum([]byte(path+c.Content)))
		dbChunks[i] = store.ChunkData{
			ID:        id,
			Content:   c.Content,
			StartLine: c.StartLine,
			EndLine:   c.EndLine,
			Embedding: allEmbeddings[i],
		}
	}

	if err := idx.store.SaveChunks(path, currentHash, dbChunks); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "内容更新，已索引: %s (%d 个分块)\n", path, len(chunks))
	return nil
}
