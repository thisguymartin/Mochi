package github

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestBuildPRBody_NoLog(t *testing.T) {
	opts := PROptions{
		Slug:     "add-auth",
		Branch:   "feature/add-auth",
		Task:     "Add user authentication",
		LogPath:  "/nonexistent/path.log",
		RepoRoot: "/tmp",
	}
	body := buildPRBody(opts)
	if !strings.Contains(body, "Add user authentication") {
		t.Errorf("PR body does not contain task title")
	}
	if !strings.Contains(body, "MOCHI") {
		t.Errorf("PR body does not contain 'MOCHI' footer")
	}
	if strings.Contains(body, "## Agent Log") {
		t.Errorf("PR body should not contain Agent Log section when log is missing")
	}
}

func TestBuildPRBody_WithLog(t *testing.T) {
	f, err := os.CreateTemp("", "mochi-log-*.log")
	if err != nil {
		t.Fatalf("cannot create temp log file: %v", err)
	}
	f.WriteString("agent output line 1\nagent output line 2\n")
	f.Close()
	defer os.Remove(f.Name())

	opts := PROptions{
		Slug:     "fix-bug",
		Branch:   "feature/fix-bug",
		Task:     "Fix the bug",
		LogPath:  f.Name(),
		RepoRoot: "/tmp",
	}
	body := buildPRBody(opts)
	if !strings.Contains(body, "## Agent Log") {
		t.Errorf("PR body should contain Agent Log section when log file exists")
	}
	if !strings.Contains(body, "agent output line 1") {
		t.Errorf("PR body should contain log content")
	}
}

func TestReadLogSummary_MissingFile(t *testing.T) {
	result := readLogSummary("/nonexistent/path/to/log.log")
	if result != "" {
		t.Errorf("expected empty string for missing file, got %q", result)
	}
}

func TestReadLogSummary_FewLines(t *testing.T) {
	f, err := os.CreateTemp("", "mochi-log-*.log")
	if err != nil {
		t.Fatalf("cannot create temp file: %v", err)
	}
	f.WriteString("line 1\nline 2\nline 3\n")
	f.Close()
	defer os.Remove(f.Name())

	result := readLogSummary(f.Name())
	if !strings.Contains(result, "line 1") {
		t.Errorf("expected line 1 in result, got %q", result)
	}
	if !strings.Contains(result, "line 3") {
		t.Errorf("expected line 3 in result, got %q", result)
	}
}

func TestReadLogSummary_TruncatesAt20(t *testing.T) {
	f, err := os.CreateTemp("", "mochi-log-*.log")
	if err != nil {
		t.Fatalf("cannot create temp file: %v", err)
	}
	for i := 1; i <= 25; i++ {
		fmt.Fprintf(f, "line %d\n", i)
	}
	f.Close()
	defer os.Remove(f.Name())

	result := readLogSummary(f.Name())
	lines := strings.Split(strings.TrimSpace(result), "\n")
	if len(lines) != 20 {
		t.Errorf("expected 20 lines, got %d", len(lines))
	}
	// Lines 1-5 should be excluded (only last 20 lines = lines 6-25)
	if strings.Contains(result, "line 5\n") {
		t.Errorf("result should not contain early lines when file has >20 lines")
	}
	if !strings.Contains(result, "line 25") {
		t.Errorf("result should contain last line (line 25)")
	}
}
