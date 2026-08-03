package utils

import (
	"testing"
	"time"
)

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
	}

	for _, tt := range tests {
		result := FormatBytes(tt.bytes)
		if result != tt.expected {
			t.Errorf("FormatBytes(%d) = %s; want %s", tt.bytes, result, tt.expected)
		}
	}
}

func TestGetFileIcon(t *testing.T) {
	if icon := GetFileIcon("test.go", false, false); icon != "📄" {
		t.Errorf("Expected 📄 for test.go, got %s", icon)
	}
	if icon := GetFileIcon("folder", true, false); icon != "📁" {
		t.Errorf("Expected 📁 for folder, got %s", icon)
	}
	if icon := GetFileIcon("..", true, false); icon != "⬆️ " {
		t.Errorf("Expected ⬆️  for .., got %s", icon)
	}
	if icon := GetFileIcon("archive.zip", false, false); icon != "📦" {
		t.Errorf("Expected 📦 for zip, got %s", icon)
	}
}

func TestTruncateString(t *testing.T) {
	if s := TruncateString("hello world", 8); s != "hello..." {
		t.Errorf("Expected hello..., got %s", s)
	}
	if s := TruncateString("short", 10); s != "short" {
		t.Errorf("Expected short, got %s", s)
	}
}

func TestFormatTime(t *testing.T) {
	now := time.Now()
	if formatted := FormatTime(now); formatted == "" {
		t.Errorf("Expected non-empty time string")
	}
}
