package common

// SupportedExtensions 定义所有支持的文件扩展名
var SupportedExtensions = map[string]bool{
	".go":    true,
	".py":    true,
	".ts":    true,
	".tsx":   true,
	".js":    true,
	".jsx":   true,
	".mjs":   true,
	".cjs":   true,
	".rs":    true,
	".java":  true,
	".cpp":   true,
	".cc":    true,
	".cxx":   true,
	".c":     true,
	".h":     true,
	".hpp":   true,
	".cs":    true,
	".rb":    true,
	".php":   true,
	".sh":    true,
	".bash":  true,
	".lua":   true,
	".sql":   true,
	".swift": true,
}

// IsSupportedExt 检查文件扩展名是否被支持
func IsSupportedExt(ext string) bool {
	return SupportedExtensions[ext]
}
