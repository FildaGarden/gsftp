package utils

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// FormatBytes converts raw byte count into human-readable string (B, KB, MB, GB, TB)
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// FormatTime formats modification time into a clean string
func FormatTime(t time.Time) string {
	now := time.Now()
	if t.Year() == now.Year() {
		return t.Format("Jan 02 15:04")
	}
	return t.Format("Jan 02  2006")
}

// GetFileIcon returns an emoji/icon for a given file name and dir status
func GetFileIcon(name string, isDir bool, isSymlink bool) string {
	if isDir {
		if name == ".." {
			return "⬆️ "
		}
		return "📁"
	}
	if isSymlink {
		return "🔗"
	}

	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".go", ".py", ".js", ".ts", ".rs", ".c", ".cpp", ".h", ".html", ".css", ".json", ".yaml", ".yml", ".md":
		return "📄"
	case ".zip", ".tar", ".gz", ".7z", ".bz2", ".xz", ".rar":
		return "📦"
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg", ".ico":
		return "🖼️"
	case ".mp3", ".wav", ".flac", ".ogg", ".aac":
		return "🎵"
	case ".mp4", ".mkv", ".avi", ".mov", ".webm":
		return "🎬"
	case ".pdf", ".doc", ".docx", ".xls", ".xlsx":
		return "📑"
	case ".sh", ".bash", ".zsh":
		return "⚙️"
	default:
		return "📄"
	}
}

// TruncateString truncates a string if it exceeds maxLen
func TruncateString(s string, maxLen int) string {
	if maxLen <= 3 {
		return "..."
	}
	if len(s) > maxLen {
		return s[:maxLen-3] + "..."
	}
	return s
}
