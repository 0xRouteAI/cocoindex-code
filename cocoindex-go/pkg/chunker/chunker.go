package chunker

import (
	"path/filepath"
	"strings"

	"github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/bash"
	"github.com/smacker/go-tree-sitter/cpp"
	"github.com/smacker/go-tree-sitter/css"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/smacker/go-tree-sitter/html"
	"github.com/smacker/go-tree-sitter/java"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/lua"
	"github.com/smacker/go-tree-sitter/php"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/smacker/go-tree-sitter/ruby"
	"github.com/smacker/go-tree-sitter/rust"
	"github.com/smacker/go-tree-sitter/sql"
	"github.com/smacker/go-tree-sitter/swift"
	"github.com/smacker/go-tree-sitter/yaml"
)

const (
	MaxChunkSize = 1000
	MinChunkSize = 100
	OverlapSize  = 200
)

type Chunk struct {
	Content   string
	StartLine int
	EndLine   int
}

type Chunker struct {
	language *sitter.Language
	langName string
}

func NewChunker(filename string) (*Chunker, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".go":
		return &Chunker{language: golang.GetLanguage(), langName: "go"}, nil
	case ".py":
		return &Chunker{language: python.GetLanguage(), langName: "python"}, nil
	case ".js", ".mjs", ".cjs":
		return &Chunker{language: javascript.GetLanguage(), langName: "javascript"}, nil
	case ".ts", ".tsx":
		// 虽然 typescript 可能不在 smacker 主库，但通常可以使用 javascript 解析器或通用模式
		return &Chunker{language: javascript.GetLanguage(), langName: "typescript"}, nil
	case ".cpp", ".cc", ".cxx", ".h", ".hpp":
		return &Chunker{language: cpp.GetLanguage(), langName: "cpp"}, nil
	case ".java":
		return &Chunker{language: java.GetLanguage(), langName: "java"}, nil
	case ".rs":
		return &Chunker{language: rust.GetLanguage(), langName: "rust"}, nil
	case ".rb":
		return &Chunker{language: ruby.GetLanguage(), langName: "ruby"}, nil
	case ".php":
		return &Chunker{language: php.GetLanguage(), langName: "php"}, nil
	case ".sh", ".bash":
		return &Chunker{language: bash.GetLanguage(), langName: "bash"}, nil
	case ".lua":
		return &Chunker{language: lua.GetLanguage(), langName: "lua"}, nil
	case ".sql":
		return &Chunker{language: sql.GetLanguage(), langName: "sql"}, nil
	case ".swift":
		return &Chunker{language: swift.GetLanguage(), langName: "swift"}, nil
	case ".html", ".htm":
		return &Chunker{language: html.GetLanguage(), langName: "html"}, nil
	case ".css":
		return &Chunker{language: css.GetLanguage(), langName: "css"}, nil
	case ".yaml", ".yml":
		return &Chunker{language: yaml.GetLanguage(), langName: "yaml"}, nil
	case ".zig", ".dockerfile", ".md", ".json":
		// 这些语言目前通过“通用递归模式”解析，保证最高兼容性
		return &Chunker{language: nil, langName: ext[1:]}, nil
	default:
		return &Chunker{language: nil, langName: "text"}, nil
	}
}

func (c *Chunker) Split(code []byte) ([]Chunk, error) {
	if c.language == nil {
		return c.splitTextOnly(code), nil
	}

	parser := sitter.NewParser()
	parser.SetLanguage(c.language)
	tree := parser.Parse(nil, code)
	root := tree.RootNode()

	var finalChunks []Chunk
	for i := 0; i < int(root.NamedChildCount()); i++ {
		child := root.NamedChild(i)
		nodeText := string(code[child.StartByte():child.EndByte()])
		startLine := int(child.StartPoint().Row) + 1
		endLine := int(child.EndPoint().Row) + 1

		if len(nodeText) > MaxChunkSize {
			subChunks := c.splitWithOverlap(nodeText, startLine)
			finalChunks = append(finalChunks, subChunks...)
		} else if len(nodeText) >= MinChunkSize {
			finalChunks = append(finalChunks, Chunk{
				Content:   nodeText,
				StartLine: startLine,
				EndLine:   endLine,
			})
		}
	}

	if len(finalChunks) == 0 {
		return c.splitTextOnly(code), nil
	}

	return finalChunks, nil
}

func (c *Chunker) splitWithOverlap(text string, baseLine int) []Chunk {
	var chunks []Chunk
	runes := []rune(text)
	textLen := len(runes)

	for i := 0; i < textLen; i += (MaxChunkSize - OverlapSize) {
		end := i + MaxChunkSize
		if end > textLen {
			end = textLen
		}
		chunkContent := string(runes[i:end])
		lines := strings.Count(string(runes[:i]), "\n")
		endLines := strings.Count(chunkContent, "\n")

		chunks = append(chunks, Chunk{
			Content:   chunkContent,
			StartLine: baseLine + lines,
			EndLine:   baseLine + lines + endLines,
		})
		if end == textLen {
			break
		}
	}
	return chunks
}

func (c *Chunker) splitTextOnly(code []byte) []Chunk {
	return c.splitWithOverlap(string(code), 1)
}
