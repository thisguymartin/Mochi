package source

import (
	"strings"
	"unicode"
)

// Task represents a single unit of work from any source.
type Task struct {
	Title       string // Short title (e.g. filename, issue title)
	Description string // Full context extracted from the source
	Slug        string // Branch-safe identifier
	Model       string // Optional per-task model override
}

// Source is the interface that all task sources must implement.
type Source interface {
	Name() string
	FetchTasks() ([]Task, error)
}

// ToSlug converts a human-readable string into a lowercase, hyphen-separated
// branch-safe identifier. e.g. "Add user auth!" -> "add-user-auth"
func ToSlug(s string) string {
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
		res = strings.TrimRight(res[:100], "-")
	}
	return res
}
