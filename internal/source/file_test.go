package source

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileSource_Name(t *testing.T) {
	fs := &FileSource{Path: "test.md"}
	if got := fs.Name(); got != "file" {
		t.Errorf("Name() = %q; want %q", got, "file")
	}
}

func TestFileSource_BasicFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "add-user-auth.md")
	if err := os.WriteFile(path, []byte("Implement JWT-based authentication"), 0644); err != nil {
		t.Fatal(err)
	}

	fs := &FileSource{Path: path}
	tasks, err := fs.FetchTasks()
	if err != nil {
		t.Fatalf("FetchTasks() error: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("FetchTasks() returned %d tasks; want 1", len(tasks))
	}

	task := tasks[0]
	if task.Title != "add-user-auth" {
		t.Errorf("Title = %q; want %q", task.Title, "add-user-auth")
	}
	if task.Description != "Implement JWT-based authentication" {
		t.Errorf("Description = %q; want %q", task.Description, "Implement JWT-based authentication")
	}
	if task.Slug != "add-user-auth" {
		t.Errorf("Slug = %q; want %q", task.Slug, "add-user-auth")
	}
}

func TestFileSource_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.md")
	if err := os.WriteFile(path, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	fs := &FileSource{Path: path}
	_, err := fs.FetchTasks()
	if err == nil {
		t.Fatal("expected error for empty file, got nil")
	}
}

func TestFileSource_NonexistentFile(t *testing.T) {
	fs := &FileSource{Path: "/nonexistent/file.md"}
	_, err := fs.FetchTasks()
	if err == nil {
		t.Fatal("expected error for nonexistent file, got nil")
	}
}

func TestFileSource_WhitespaceOnly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "spaces.md")
	if err := os.WriteFile(path, []byte("   \n\n  \t  \n"), 0644); err != nil {
		t.Fatal(err)
	}

	fs := &FileSource{Path: path}
	_, err := fs.FetchTasks()
	if err == nil {
		t.Fatal("expected error for whitespace-only file, got nil")
	}
}
