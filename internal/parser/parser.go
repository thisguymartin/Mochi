package parser

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

// Task represents a single unit of work parsed from a task file.
type Task struct {
	Title       string // Short, single-line title from the bullet point
	Description string // Full, multi-line description of the task
	Slug        string // Branch-safe identifier, e.g. "add-user-auth"
	Model       string // Optional per-task model override
}

var (
	modelAnnotation = regexp.MustCompile(`\[model:([^\]]+)\]`)
	titleAnnotation = regexp.MustCompile(`\[title:([^\]]+)\]`)
	bulletPattern   = regexp.MustCompile(`^[\s]*[-*]\s+`)
)

// ParseFile reads a markdown task file and returning all tasks found under a
// "## Tasks" section. Lines starting with "#" or blank lines are appended to
// the current task's description if one is active.
// If no tasks are found under "## Tasks", it falls back to reading the entire
// file as a single large task.
func ParseFile(path string) ([]Task, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("cannot open task file %q: %w", path, err)
	}
	defer f.Close()

	var tasks []Task
	var currentTask *Task
	inTasksSection := false

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Detect section headers
		if strings.HasPrefix(trimmed, "## ") {
			// If we were parsing a task, save it before moving on
			if currentTask != nil {
				tasks = append(tasks, *currentTask)
				currentTask = nil
			}
			inTasksSection = strings.EqualFold(strings.TrimPrefix(trimmed, "## "), "tasks")
			continue
		}

		if !inTasksSection {
			continue
		}

		// Check if it's a new bullet point (start of a new task)
		if bulletPattern.MatchString(line) {
			if currentTask != nil {
				tasks = append(tasks, *currentTask)
			}

			title := strings.TrimSpace(bulletPattern.ReplaceAllString(line, ""))
			model := ""
			explicitTitle := ""

			// Extract optional [model:...] annotation from the title
			if m := modelAnnotation.FindStringSubmatch(title); m != nil {
				model = strings.TrimSpace(m[1])
				title = strings.TrimSpace(modelAnnotation.ReplaceAllString(title, ""))
			}

			// Extract optional [title:...] annotation
			if t := titleAnnotation.FindStringSubmatch(title); t != nil {
				explicitTitle = strings.TrimSpace(t[1])
				title = strings.TrimSpace(titleAnnotation.ReplaceAllString(title, ""))
			}

			if explicitTitle != "" {
				title = explicitTitle
			}

			// Initialize the new task
			currentTask = &Task{
				Title:       title,
				Description: "",
				Slug:        toSlug(title),
				Model:       model,
			}
			continue
		}

		// If we are currently parsing a task and this isn't a new bullet point,
		// append it to the task's description layout.
		if currentTask != nil {
			if currentTask.Description != "" {
				currentTask.Description += "\n"
			}
			currentTask.Description += line
		}
	}

	// Append the very last task if there is one
	if currentTask != nil {
		tasks = append(tasks, *currentTask)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading task file: %w", err)
	}

	// Fallback mechanism: If no tasks were found under a `## Tasks` section
	// (or the section was missing entirely), read the entire file as a single task.
	if len(tasks) == 0 {
		_, err := f.Seek(0, io.SeekStart)
		if err != nil {
			return nil, fmt.Errorf("error seeking to start of file for fallback parse: %w", err)
		}

		contentBytes, err := io.ReadAll(f)
		if err != nil {
			return nil, fmt.Errorf("error reading entire file for fallback parse: %w", err)
		}

		content := string(contentBytes)
		if strings.TrimSpace(content) == "" {
			return nil, fmt.Errorf("task file %q is empty", path)
		}

		fileName := filepath.Base(path)
		title := strings.TrimSuffix(fileName, filepath.Ext(fileName)) // e.g., "PRD.md" -> "PRD"
		model := ""
		explicitTitle := ""

		// Parse out any model annotation found anywhere in the entire text
		if m := modelAnnotation.FindStringSubmatch(content); m != nil {
			model = strings.TrimSpace(m[1])
			content = strings.TrimSpace(modelAnnotation.ReplaceAllString(content, ""))
		}

		// Parse out any title annotation found anywhere in the entire text
		if t := titleAnnotation.FindStringSubmatch(content); t != nil {
			explicitTitle = strings.TrimSpace(t[1])
			content = strings.TrimSpace(titleAnnotation.ReplaceAllString(content, ""))
		}

		if explicitTitle != "" {
			title = explicitTitle
		}

		tasks = append(tasks, Task{
			Title:       title,
			Description: strings.TrimSpace(content),
			Slug:        toSlug(title),
			Model:       model,
		})
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

	res := strings.TrimRight(b.String(), "-")
	if len(res) > 100 {
		// Try to cut at exactly 100 and trim any trailing dash
		res = strings.TrimRight(res[:100], "-")
	}
	return res
}
