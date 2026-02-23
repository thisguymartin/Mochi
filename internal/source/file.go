package source

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileSource reads any file and treats its entire content as a single task.
type FileSource struct {
	Path string
}

func (s *FileSource) Name() string { return "file" }

func (s *FileSource) FetchTasks() ([]Task, error) {
	data, err := os.ReadFile(s.Path)
	if err != nil {
		return nil, fmt.Errorf("cannot read file %q: %w", s.Path, err)
	}

	content := string(data)
	if strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("file %q is empty", s.Path)
	}

	fileName := filepath.Base(s.Path)
	title := strings.TrimSuffix(fileName, filepath.Ext(fileName))

	return []Task{{
		Title:       title,
		Description: strings.TrimSpace(content),
		Slug:        ToSlug(title),
	}}, nil
}
