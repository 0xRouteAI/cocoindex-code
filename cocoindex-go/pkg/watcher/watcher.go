package watcher

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"cocoindex-go/pkg/common"
	"cocoindex-go/pkg/indexer"
	"github.com/fsnotify/fsnotify"
)

type FileWatcher struct {
	watcher *fsnotify.Watcher
	indexer *indexer.Indexer
	mu      sync.Mutex
	timers  map[string]*time.Timer
}

func NewFileWatcher(idx *indexer.Indexer) (*FileWatcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	return &FileWatcher{
		watcher: w,
		indexer: idx,
		timers:  make(map[string]*time.Timer),
	}, nil
}

func (fw *FileWatcher) Watch(rootPath string) error {
	err := filepath.WalkDir(rootPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" || d.Name() == "node_modules" || d.Name() == "vendor" {
				return filepath.SkipDir
			}
			return fw.watcher.Add(path)
		}
		return nil
	})
	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case event, ok := <-fw.watcher.Events:
				if !ok {
					return
				}

				if event.Has(fsnotify.Create) {
					info, err := os.Stat(event.Name)
					if err == nil && info.IsDir() {
						_ = fw.watcher.Add(event.Name)
						fmt.Fprintf(os.Stderr, "检测到新目录，已加入监听: %s\n", event.Name)
						continue
					}
				}

				ext := strings.ToLower(filepath.Ext(event.Name))
				if !common.IsSupportedExt(ext) {
					continue
				}

				if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
					fw.mu.Lock()
					if timer, exists := fw.timers[event.Name]; exists {
						timer.Stop()
					}
					fw.timers[event.Name] = time.AfterFunc(300*time.Millisecond, func() {
						fw.mu.Lock()
						delete(fw.timers, event.Name)
						fw.mu.Unlock()

						fmt.Fprintf(os.Stderr, "文件变更，执行增量索引: %s\n", event.Name)
						ctx := context.Background()
						_ = fw.indexer.IndexFile(ctx, event.Name)
					})
					fw.mu.Unlock()
				}

			case err, ok := <-fw.watcher.Errors:
				if !ok {
					return
				}
				fmt.Fprintf(os.Stderr, "监听器错误: %v\n", err)
			}
		}
	}()
	return nil
}
