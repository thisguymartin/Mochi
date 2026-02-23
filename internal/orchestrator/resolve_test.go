package orchestrator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/thisguymartin/ai-forge/internal/config"
	"github.com/thisguymartin/ai-forge/internal/source"
)

func TestResolveSource_InputFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.md")
	if err := os.WriteFile(path, []byte("task content"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{InputFile: path}
	src, err := ResolveSource(cfg)
	if err != nil {
		t.Fatalf("ResolveSource error: %v", err)
	}
	if _, ok := src.(*source.FileSource); !ok {
		t.Errorf("expected FileSource, got %T", src)
	}
}

func TestResolveSource_NoInput(t *testing.T) {
	cfg := config.Config{InputFile: ""}
	_, err := ResolveSource(cfg)
	if err == nil {
		t.Fatal("expected error for no input, got nil")
	}
}

func TestResolveSource_DefaultFallback(t *testing.T) {
	// When InputFile is "PRD.md" but doesn't exist, it tries candidates.
	// In a temp dir with none of them, it still returns a FileSource (pointing to PRD.md).
	origDir, _ := os.Getwd()
	dir := t.TempDir()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	cfg := config.Config{InputFile: "PRD.md"}
	src, err := ResolveSource(cfg)
	if err != nil {
		t.Fatalf("ResolveSource error: %v", err)
	}
	fs, ok := src.(*source.FileSource)
	if !ok {
		t.Fatalf("expected FileSource, got %T", src)
	}
	// Should still be PRD.md since no candidates exist
	if fs.Path != "PRD.md" {
		t.Errorf("Path = %q; want %q", fs.Path, "PRD.md")
	}
}
