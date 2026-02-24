package parser

import (
	"os"
	"testing"
)

func TestToSlug(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"Add user auth", "add-user-auth"},
		{"Fix mobile navbar bug!", "fix-mobile-navbar-bug"},
		{"Hello--World", "hello-world"},
		{"trailing dash-", "trailing-dash"},
		{"UPPERCASE LETTERS", "uppercase-letters"},
		{"123 numbers here", "123-numbers-here"},
		{"multiple   spaces", "multiple-spaces"},
		{"punctuation: removed!", "punctuation-removed"},
	}
	for _, tc := range cases {
		got := toSlug(tc.input)
		if got != tc.want {
			t.Errorf("toSlug(%q) = %q; want %q", tc.input, got, tc.want)
		}
	}
}

func writeTempFile(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp("", "mochi-parser-test-*.md")
	if err != nil {
		t.Fatalf("cannot create temp file: %v", err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("cannot write temp file: %v", err)
	}
	f.Close()
	t.Cleanup(func() { os.Remove(f.Name()) })
	return f.Name()
}

func TestParseFile_HappyPath(t *testing.T) {
	path := writeTempFile(t, `# Project

## Tasks
- Add user authentication
- Fix mobile navbar bug
`)
	tasks, err := ParseFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("got %d tasks; want 2", len(tasks))
	}
	if tasks[0].Description != "Add user authentication" {
		t.Errorf("tasks[0].Description = %q; want %q", tasks[0].Description, "Add user authentication")
	}
	if tasks[0].Slug != "add-user-authentication" {
		t.Errorf("tasks[0].Slug = %q; want %q", tasks[0].Slug, "add-user-authentication")
	}
	if tasks[1].Description != "Fix mobile navbar bug" {
		t.Errorf("tasks[1].Description = %q; want %q", tasks[1].Description, "Fix mobile navbar bug")
	}
}

func TestParseFile_ModelAnnotation(t *testing.T) {
	path := writeTempFile(t, `## Tasks
- Add auth [model:claude-opus-4-6]
- Fix bug [model:gemini-2.0-flash]
`)
	tasks, err := ParseFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("got %d tasks; want 2", len(tasks))
	}
	if tasks[0].Model != "claude-opus-4-6" {
		t.Errorf("tasks[0].Model = %q; want %q", tasks[0].Model, "claude-opus-4-6")
	}
	if tasks[0].Description != "Add auth" {
		t.Errorf("tasks[0].Description = %q; want %q", tasks[0].Description, "Add auth")
	}
	if tasks[1].Model != "gemini-2.0-flash" {
		t.Errorf("tasks[1].Model = %q; want %q", tasks[1].Model, "gemini-2.0-flash")
	}
}

func TestParseFile_NoTasksSection(t *testing.T) {
	path := writeTempFile(t, `# Project

## Overview
Just some text with no tasks section.
`)
	_, err := ParseFile(path)
	if err == nil {
		t.Fatal("expected error when no ## Tasks section, got nil")
	}
}

func TestParseFile_NoBullets(t *testing.T) {
	path := writeTempFile(t, `## Tasks
No bullet points here, just prose.
`)
	_, err := ParseFile(path)
	if err == nil {
		t.Fatal("expected error when tasks section has no bullets, got nil")
	}
}

func TestParseFile_IgnoresLinesOutsideSection(t *testing.T) {
	path := writeTempFile(t, `## Overview
- This should be ignored

## Tasks
- Valid task

## Notes
- Also ignored
`)
	tasks, err := ParseFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("got %d tasks; want 1", len(tasks))
	}
	if tasks[0].Description != "Valid task" {
		t.Errorf("tasks[0].Description = %q; want %q", tasks[0].Description, "Valid task")
	}
}

func TestParseFile_IgnoresBlankLinesAndComments(t *testing.T) {
	path := writeTempFile(t, `## Tasks

# This is a comment
- Real task one

# Another comment

- Real task two
`)
	tasks, err := ParseFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("got %d tasks; want 2", len(tasks))
	}
}
