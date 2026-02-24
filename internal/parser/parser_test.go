package parser

import (
	"os"
	"path/filepath"
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
		{
			"This is an incredibly long and extremely verbose task description that just keeps going on and on and on and frankly should be truncated before it becomes a branch name because git will probably complain if it gets too long",
			"this-is-an-incredibly-long-and-extremely-verbose-task-description-that-just-keeps-going-on-and-on-an",
		},
	}
	for _, tc := range cases {
		got := toSlug(tc.input)
		if got != tc.want {
			t.Errorf("toSlug(%q) = %q; want %q", tc.input, got, tc.want)
		}
	}
}

func writeTempFile(t *testing.T, name, content string) string {
	t.Helper()
	if name == "" {
		name = "mochi-parser-test-*.md"
	}
	f, err := os.CreateTemp("", name)
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
	path := writeTempFile(t, "", `# Project

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
	if tasks[0].Title != "Add user authentication" {
		t.Errorf("tasks[0].Title = %q; want %q", tasks[0].Title, "Add user authentication")
	}
	if tasks[0].Slug != "add-user-authentication" {
		t.Errorf("tasks[0].Slug = %q; want %q", tasks[0].Slug, "add-user-authentication")
	}
	if tasks[1].Title != "Fix mobile navbar bug" {
		t.Errorf("tasks[1].Title = %q; want %q", tasks[1].Title, "Fix mobile navbar bug")
	}
}

func TestParseFile_ModelAnnotation(t *testing.T) {
	path := writeTempFile(t, "", `## Tasks
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
	if tasks[0].Title != "Add auth" {
		t.Errorf("tasks[0].Title = %q; want %q", tasks[0].Title, "Add auth")
	}
	if tasks[1].Model != "gemini-2.0-flash" {
		t.Errorf("tasks[1].Model = %q; want %q", tasks[1].Model, "gemini-2.0-flash")
	}
}

func TestParseFile_TitleAnnotation(t *testing.T) {
	path := writeTempFile(t, "", `## Tasks
- I really don't like this title [title:Shorter Title]
- Another task [model:gemini-2.0-flash] [title:Custom Model Title]
`)
	tasks, err := ParseFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("got %d tasks; want 2", len(tasks))
	}
	if tasks[0].Title != "Shorter Title" {
		t.Errorf("tasks[0].Title = %q; want %q", tasks[0].Title, "Shorter Title")
	}
	if tasks[0].Slug != "shorter-title" {
		t.Errorf("tasks[0].Slug = %q; want %q", tasks[0].Slug, "shorter-title")
	}

	if tasks[1].Title != "Custom Model Title" {
		t.Errorf("tasks[1].Title = %q; want %q", tasks[1].Title, "Custom Model Title")
	}
	if tasks[1].Model != "gemini-2.0-flash" {
		t.Errorf("tasks[1].Model = %q; want %q", tasks[1].Model, "gemini-2.0-flash")
	}
}

func TestParseFile_MultilineDescription(t *testing.T) {
	path := writeTempFile(t, "", `## Tasks
- Task One
This is line one of the description.
And line two!
- Task Two
Single line desc for two.
`)
	tasks, err := ParseFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("got %d tasks; want 2", len(tasks))
	}
	if tasks[0].Title != "Task One" {
		t.Errorf("tasks[0].Title = %q; want Task One", tasks[0].Title)
	}
	wantDesc1 := "This is line one of the description.\nAnd line two!"
	if tasks[0].Description != wantDesc1 {
		t.Errorf("tasks[0].Description = %q; want %q", tasks[0].Description, wantDesc1)
	}

	if tasks[1].Title != "Task Two" {
		t.Errorf("tasks[1].Title = %q; want Task Two", tasks[1].Title)
	}
	if tasks[1].Description != "Single line desc for two." {
		t.Errorf("tasks[1].Description = %q; want single line desc", tasks[1].Description)
	}
}

func TestParseFile_FallbackSingleFile(t *testing.T) {
	content := `# Project Overview
This is just a regular markdown file.
It has no tasks section.
But it does request a model! [model:gemini-2.5-pro]
`
	path := writeTempFile(t, "PRD-testing-*.md", content)

	tasks, err := ParseFile(path)
	if err != nil {
		t.Fatalf("expected fallback to succeed, got error: %v", err)
	}

	if len(tasks) != 1 {
		t.Fatalf("got %d tasks; want 1", len(tasks))
	}

	baseName := filepath.Base(path)
	expectedTitle := baseName[:len(baseName)-len(filepath.Ext(baseName))]

	if tasks[0].Title != expectedTitle {
		t.Errorf("tasks[0].Title = %q; want %q", tasks[0].Title, expectedTitle)
	}

	expectedDesc := `# Project Overview
This is just a regular markdown file.
It has no tasks section.
But it does request a model!`
	if tasks[0].Description != expectedDesc {
		t.Errorf("tasks[0].Description = %q; want %q", tasks[0].Description, expectedDesc)
	}
	if tasks[0].Model != "gemini-2.5-pro" {
		t.Errorf("tasks[0].Model = %q; want gemini-2.5-pro", tasks[0].Model)
	}
}

func TestParseFile_IgnoresLinesOutsideSection(t *testing.T) {
	path := writeTempFile(t, "", `## Overview
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
	if tasks[0].Title != "Valid task" {
		t.Errorf("tasks[0].Title = %q; want %q", tasks[0].Title, "Valid task")
	}
}

func TestParseFile_IgnoresBlankLinesAndComments(t *testing.T) {
	path := writeTempFile(t, "", `## Tasks

# This is a comment
- Real task one
This task has a description
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

	wantDesc := "This task has a description\n# Another comment\n"
	if tasks[0].Description != wantDesc {
		t.Errorf("tasks[0].Description = %q; want %q", tasks[0].Description, wantDesc)
	}
}
