package orchestrator

import (
	"github.com/thisguymartin/ai-forge/internal/config"
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
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += ", "
		}
		result += p
	}
	return result
}

// statusStr returns "done" or "failed" based on the success flag.
func statusStr(success bool) string {
	if success {
		return "done"
	}
	return "failed"
}

// loopEnabled returns true when the Ralph Loop should run more than once
// or when a reviewer is configured.
func loopEnabled(cfg config.Config) bool {
	return cfg.ReviewerModel != "" || cfg.MaxIterations > 1
}
