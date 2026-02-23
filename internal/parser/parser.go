package parser

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
	"unicode"
)

// Task represents a single unit of work parsed from a task file.
type Task struct {
	Description string // Human-readable task description
	Slug        string // Branch-safe identifier, e.g. "add-user-auth"
	Model       string // Optional per-task model override
}

var (
	modelAnnotation = regexp.MustCompile(`\[model:([^\]]+)\]`)
	bulletPattern   = regexp.MustCompile(`^[\s]*[-*]\s+`)
)

// ParseFile reads a markdown task file and returns all tasks found under a
// "## Tasks" section. Lines starting with "#" or blank lines are skipped.
// Tasks may include a model annotation: "- Add auth [model:claude-opus-4-6]"
func ParseFile(path string) ([]Task, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("cannot open task file %q: %w", path, err)
	}
	defer f.Close()

	var tasks []Task
	inTasksSection := false

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Detect section headers — only parse under "## Tasks"
		if strings.HasPrefix(trimmed, "## ") {
			inTasksSection = strings.EqualFold(strings.TrimPrefix(trimmed, "## "), "tasks")
			continue
		}

		if !inTasksSection {
			continue
		}

		// Skip blank lines and comments
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Must be a bullet point to be treated as a task
		if !bulletPattern.MatchString(line) {
			continue
		}

		description := strings.TrimSpace(bulletPattern.ReplaceAllString(line, ""))

		// Extract optional [model:...] annotation
		model := ""
		if m := modelAnnotation.FindStringSubmatch(description); m != nil {
			model = strings.TrimSpace(m[1])
			description = strings.TrimSpace(modelAnnotation.ReplaceAllString(description, ""))
		}

		if description == "" {
			continue
		}

		tasks = append(tasks, Task{
			Description: description,
			Slug:        toSlug(description),
			Model:       model,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading task file: %w", err)
	}

	if len(tasks) == 0 {
		return nil, fmt.Errorf("no tasks found in %q — ensure the file has a '## Tasks' section with bullet points", path)
	}

	return tasks, nil
}

// toSlug converts a human-readable string into a lowercase, hyphen-separated
// branch-safe identifier. e.g. "Add user auth!" -> "add-user-auth"
func toSlug(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	prevDash := false
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			prevDash = false
		} else if !prevDash && b.Len() > 0 {
			b.WriteRune('-')
			prevDash = true
		}
	}
	return strings.TrimRight(b.String(), "-")
}
