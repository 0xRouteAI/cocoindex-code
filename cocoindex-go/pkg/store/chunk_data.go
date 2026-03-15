package store

// ChunkData 表示要保存的代码块数据
type ChunkData struct {
	ID        string
	Content   string
	StartLine int
	EndLine   int
	Embedding []float32
}
