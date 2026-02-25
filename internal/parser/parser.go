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

	// Matches standard markdown bullets: "- ", "* ", "  - ", etc.
	bulletPattern = regexp.MustCompile(`^[\s]*[-*]\s+`)

	// Matches markdown checkboxes: "- [ ] task", "- [x] done task"
	checkboxPattern = regexp.MustCompile(`^[\s]*[-*]\s+\[([ xX])\]\s+`)

	// Matches numbered lists: "1. task", "  2) task"
	numberedPattern = regexp.MustCompile(`^[\s]*\d+[.)]\s+`)

	// Matches task section headers (case-insensitive)
	taskSectionPattern = regexp.MustCompile(`(?i)^##\s+(tasks?|todo|to-?do|action items|work items|checklist|steps)$`)
)

// ParseFile reads a task file and extracts tasks using multi-strategy detection.
//
// Detection order:
//  1. Markdown "## Tasks" section with bullet points (classic mode)
//  2. Markdown checkboxes anywhere in the file (- [ ] / - [x])
//  3. Numbered list items under a recognized task heading
//  4. Bullet points under any recognized task section heading
//  5. Fallback: entire file content as a single task
//
// Supported file formats: any text-based format (.md, .txt, .yaml, .json, etc.)
// The content is passed through to the AI model which handles format-specific parsing.
func ParseFile(path string) ([]Task, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("cannot open task file %q: %w", path, err)
	}
	defer f.Close()

	// Strategy 1: Parse structured task sections
	tasks, err := parseStructuredTasks(f)
	if err != nil {
		return nil, err
	}
	if len(tasks) > 0 {
		return tasks, nil
	}

	// Strategy 2: Scan for checkboxes anywhere in the file
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("error seeking: %w", err)
	}
	tasks, err = parseCheckboxTasks(f)
	if err != nil {
		return nil, err
	}
	if len(tasks) > 0 {
		return tasks, nil
	}

	// Strategy 3: Fallback — entire file as a single task
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("error seeking: %w", err)
	}
	return parseFallbackSingleTask(f, path)
}

// parseStructuredTasks extracts tasks from recognized section headings
// (## Tasks, ## Todo, ## Action Items, ## Steps, etc.) using bullets or numbered lists.
func parseStructuredTasks(r io.Reader) ([]Task, error) {
	var tasks []Task
	var currentTask *Task
	inTasksSection := false

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Detect section headers
		if strings.HasPrefix(trimmed, "## ") {
			if currentTask != nil {
				tasks = append(tasks, *currentTask)
				currentTask = nil
			}
			header := strings.TrimPrefix(trimmed, "## ")
			inTasksSection = taskSectionPattern.MatchString(trimmed) ||
				strings.EqualFold(header, "tasks")
			continue
		}

		if !inTasksSection {
			continue
		}

		// Check for checkbox items: - [ ] task or - [x] done task
		if checkboxPattern.MatchString(line) {
			if currentTask != nil {
				tasks = append(tasks, *currentTask)
			}
			// Skip completed checkboxes
			match := checkboxPattern.FindStringSubmatch(line)
			if match != nil && (match[1] == "x" || match[1] == "X") {
				currentTask = nil
				continue
			}
			title := strings.TrimSpace(checkboxPattern.ReplaceAllString(line, ""))
			currentTask = extractTaskFromLine(title)
			continue
		}

		// Check for bullet points: - task or * task
		if bulletPattern.MatchString(line) {
			if currentTask != nil {
				tasks = append(tasks, *currentTask)
			}
			title := strings.TrimSpace(bulletPattern.ReplaceAllString(line, ""))
			currentTask = extractTaskFromLine(title)
			continue
		}

		// Check for numbered lists: 1. task or 2) task
		if numberedPattern.MatchString(line) {
			if currentTask != nil {
				tasks = append(tasks, *currentTask)
			}
			title := strings.TrimSpace(numberedPattern.ReplaceAllString(line, ""))
			currentTask = extractTaskFromLine(title)
			continue
		}

		// Continuation lines for the current task's description
		if currentTask != nil {
			if currentTask.Description != "" {
				currentTask.Description += "\n"
			}
			currentTask.Description += line
		}
	}

	if currentTask != nil {
		tasks = append(tasks, *currentTask)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading task file: %w", err)
	}

	return tasks, nil
}

// parseCheckboxTasks scans the entire file for markdown checkboxes (- [ ] / - [x])
// regardless of section structure. Completed checkboxes are skipped.
func parseCheckboxTasks(r io.Reader) ([]Task, error) {
	var tasks []Task
	var currentTask *Task

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()

		if checkboxPattern.MatchString(line) {
			if currentTask != nil {
				tasks = append(tasks, *currentTask)
			}
			match := checkboxPattern.FindStringSubmatch(line)
			if match != nil && (match[1] == "x" || match[1] == "X") {
				currentTask = nil
				continue
			}
			title := strings.TrimSpace(checkboxPattern.ReplaceAllString(line, ""))
			currentTask = extractTaskFromLine(title)
			continue
		}

		// Continuation lines
		if currentTask != nil {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
				if currentTask.Description != "" {
					currentTask.Description += "\n"
				}
				currentTask.Description += line
			} else if trimmed == "" {
				// Blank line ends the description for checkbox-mode
				if currentTask != nil {
					tasks = append(tasks, *currentTask)
					currentTask = nil
				}
			}
		}
	}

	if currentTask != nil {
		tasks = append(tasks, *currentTask)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading task file: %w", err)
	}

	return tasks, nil
}

// parseFallbackSingleTask reads the entire file as a single task.
func parseFallbackSingleTask(r io.Reader, path string) ([]Task, error) {
	contentBytes, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("error reading file for fallback parse: %w", err)
	}

	content := string(contentBytes)
	if strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("task file %q is empty", path)
	}

	fileName := filepath.Base(path)
	title := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	model := ""
	explicitTitle := ""

	if m := modelAnnotation.FindStringSubmatch(content); m != nil {
		model = strings.TrimSpace(m[1])
		content = strings.TrimSpace(modelAnnotation.ReplaceAllString(content, ""))
	}

	if t := titleAnnotation.FindStringSubmatch(content); t != nil {
		explicitTitle = strings.TrimSpace(t[1])
		content = strings.TrimSpace(titleAnnotation.ReplaceAllString(content, ""))
	}

	if explicitTitle != "" {
		title = explicitTitle
	}

	return []Task{{
		Title:       title,
		Description: strings.TrimSpace(content),
		Slug:        toSlug(title),
		Model:       model,
	}}, nil
}

// extractTaskFromLine parses a single task line, extracting annotations.
func extractTaskFromLine(title string) *Task {
	model := ""
	explicitTitle := ""

	if m := modelAnnotation.FindStringSubmatch(title); m != nil {
		model = strings.TrimSpace(m[1])
		title = strings.TrimSpace(modelAnnotation.ReplaceAllString(title, ""))
	}

	if t := titleAnnotation.FindStringSubmatch(title); t != nil {
		explicitTitle = strings.TrimSpace(t[1])
		title = strings.TrimSpace(titleAnnotation.ReplaceAllString(title, ""))
	}

	if explicitTitle != "" {
		title = explicitTitle
	}

	return &Task{
		Title:       title,
		Description: "",
		Slug:        toSlug(title),
		Model:       model,
	}
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
