package orchestrator

import (
	"strings"

	"github.com/thisguymartin/ai-forge/internal/source"
)

// filterBySlug returns the single task matching slug, or nil.
func filterBySlug(tasks []source.Task, slug string) []source.Task {
	for _, t := range tasks {
		if t.Slug == slug {
			return []source.Task{t}
		}
	}
	return nil
}

// slugList returns a comma-separated string of task slugs.
func slugList(tasks []source.Task) string {
	parts := make([]string, len(tasks))
	for i, t := range tasks {
		parts[i] = t.Slug
	}
	return strings.Join(parts, ", ")
}

// statusStr returns "done" or "failed" based on the success flag.
func statusStr(success bool) string {
	if success {
		return "done"
	}
	return "failed"
}
