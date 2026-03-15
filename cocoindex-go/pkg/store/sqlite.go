package store

import (
	"container/heap"
	"database/sql"
	"encoding/binary"
	"math"

	_ "github.com/mattn/go-sqlite3"
)

type Store struct {
	db *sql.DB
}

type SearchResult struct {
	FilePath  string
	Content   string
	StartLine int
	EndLine   int
	Distance  float32
}

// ResultHeap 实现最大堆，用于保留 Top-K 结果
type ResultHeap []SearchResult

func (h ResultHeap) Len() int           { return len(h) }
func (h ResultHeap) Less(i, j int) bool { return h[i].Distance > h[j].Distance } // 最大堆
func (h ResultHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *ResultHeap) Push(x interface{}) { *h = append(*h, x.(SearchResult)) }
func (h *ResultHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

func NewStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	// 性能优化配置
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA cache_size=-64000", // 64MB cache
		"PRAGMA temp_store=MEMORY",
	}
	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			return nil, err
		}
	}

	// 初始化表结构
	queries := []string{
		`CREATE TABLE IF NOT EXISTS files (
			path TEXT PRIMARY KEY,
			hash TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS chunks (
			id TEXT PRIMARY KEY,
			file_path TEXT,
			content TEXT,
			start_line INTEGER,
			end_line INTEGER,
			embedding BLOB
		);`,
		`CREATE INDEX IF NOT EXISTS idx_chunks_file_path ON chunks(file_path);`,
	}

	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			return nil, err
		}
	}

	return &Store{db: db}, nil
}

func (s *Store) GetFileHash(path string) (string, error) {
	var hash string
	err := s.db.QueryRow("SELECT hash FROM files WHERE path = ?", path).Scan(&hash)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return hash, err
}

func Float32ToByte(f []float32) []byte {
	buf := make([]byte, len(f)*4)
	for i, v := range f {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(v))
	}
	return buf
}

func ByteToFloat32(b []byte) []float32 {
	f := make([]float32, len(b)/4)
	for i := range f {
		f[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:]))
	}
	return f
}

// CosineSimilarity 计算余弦相似度（OpenAI embeddings 已归一化，直接点积即可）
func CosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dotProduct float32
	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
	}
	return dotProduct
}

func (s *Store) SaveChunks(filePath, fileHash string, chunks []ChunkData) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. 更新文件哈希
	if _, err := tx.Exec("INSERT OR REPLACE INTO files (path, hash) VALUES (?, ?)", filePath, fileHash); err != nil {
		return err
	}

	// 2. 清理旧分块
	if _, err := tx.Exec("DELETE FROM chunks WHERE file_path = ?", filePath); err != nil {
		return err
	}

	// 3. 批量插入新分块
	stmt, err := tx.Prepare("INSERT INTO chunks (id, file_path, content, start_line, end_line, embedding) VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, c := range chunks {
		blob := Float32ToByte(c.Embedding)
		if _, err := stmt.Exec(c.ID, filePath, c.Content, c.StartLine, c.EndLine, blob); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) DeleteFile(filePath string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM files WHERE path = ?", filePath); err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM chunks WHERE file_path = ?", filePath); err != nil {
		return err
	}
	return tx.Commit()
}

// Search 优化版：使用最大堆保留 Top-K，避免全量排序
func (s *Store) Search(queryVector []float32, limit int) ([]SearchResult, error) {
	rows, err := s.db.Query("SELECT file_path, content, start_line, end_line, embedding FROM chunks")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	h := &ResultHeap{}
	heap.Init(h)

	for rows.Next() {
		var r SearchResult
		var embBlob []byte
		if err := rows.Scan(&r.FilePath, &r.Content, &r.StartLine, &r.EndLine, &embBlob); err != nil {
			return nil, err
		}

		targetVec := ByteToFloat32(embBlob)
		r.Distance = CosineSimilarity(queryVector, targetVec)

		if h.Len() < limit {
			heap.Push(h, r)
		} else if r.Distance > (*h)[0].Distance {
			heap.Pop(h)
			heap.Push(h, r)
		}
	}

	// 从堆中提取结果并按相似度降序排列
	results := make([]SearchResult, h.Len())
	for i := h.Len() - 1; i >= 0; i-- {
		results[i] = heap.Pop(h).(SearchResult)
	}
	return results, nil
}
