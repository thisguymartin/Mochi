package source

import (
	"strings"
	"testing"
)

func TestToSlug_BasicConversion(t *testing.T) {
	got := ToSlug("Add user auth!")
	if got != "add-user-auth" {
		t.Errorf("ToSlug(\"Add user auth!\") = %q; want %q", got, "add-user-auth")
	}
}

func TestToSlug_AlreadyClean(t *testing.T) {
	got := ToSlug("fix-navbar")
	if got != "fix-navbar" {
		t.Errorf("ToSlug(\"fix-navbar\") = %q; want %q", got, "fix-navbar")
	}
}

func TestToSlug_LeadingTrailingSpecialChars(t *testing.T) {
	got := ToSlug("!!!hello world???")
	if got != "hello-world" {
		t.Errorf("ToSlug(\"!!!hello world???\") = %q; want %q", got, "hello-world")
	}
}

func TestToSlug_Digits(t *testing.T) {
	got := ToSlug("Task 123 done")
	if got != "task-123-done" {
		t.Errorf("ToSlug(\"Task 123 done\") = %q; want %q", got, "task-123-done")
	}
}

func TestToSlug_LongString(t *testing.T) {
	long := strings.Repeat("a", 120)
	got := ToSlug(long)
	if len(got) > 100 {
		t.Errorf("ToSlug long string length = %d; want <= 100", len(got))
	}
	if strings.HasSuffix(got, "-") {
		t.Errorf("ToSlug long string ends with dash: %q", got)
	}
}

func TestToSlug_Empty(t *testing.T) {
	got := ToSlug("")
	if got != "" {
		t.Errorf("ToSlug(\"\") = %q; want %q", got, "")
	}
}

func TestToSlug_ConsecutiveSpaces(t *testing.T) {
	got := ToSlug("hello    world")
	if got != "hello-world" {
		t.Errorf("ToSlug(\"hello    world\") = %q; want %q", got, "hello-world")
	}
}
