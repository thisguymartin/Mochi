package reviewer

import (
	"testing"
)

func TestParseDecision_Done(t *testing.T) {
	d := parseDecision("DONE")
	if !d.Done {
		t.Error("parseDecision(\"DONE\") should be Done=true")
	}
}

func TestParseDecision_Retry(t *testing.T) {
	d := parseDecision("RETRY: Fix tests")
	if d.Done {
		t.Error("parseDecision RETRY should be Done=false")
	}
	if d.Feedback != "Fix tests" {
		t.Errorf("Feedback = %q; want %q", d.Feedback, "Fix tests")
	}
}

func TestParseDecision_MixedCase(t *testing.T) {
	d := parseDecision("done")
	if !d.Done {
		t.Error("parseDecision(\"done\") should be Done=true (case insensitive)")
	}
}

func TestParseDecision_RetryMixedCase(t *testing.T) {
	d := parseDecision("Retry: needs more work")
	if d.Done {
		t.Error("parseDecision Retry mixed case should be Done=false")
	}
	if d.Feedback != "needs more work" {
		t.Errorf("Feedback = %q; want %q", d.Feedback, "needs more work")
	}
}

func TestParseDecision_NoSignal(t *testing.T) {
	d := parseDecision("I think the code looks fine but could use some improvements")
	if d.Done {
		t.Error("parseDecision with no signal should default to Done=false")
	}
	if d.Feedback == "" {
		t.Error("Feedback should contain the raw output as feedback")
	}
}

func TestParseDecision_MultiLine(t *testing.T) {
	input := "Looking at the code...\nDONE\nSome extra text"
	d := parseDecision(input)
	if !d.Done {
		t.Error("parseDecision multiline with DONE should be Done=true")
	}
}

func TestParseDecision_RetryMultiLine(t *testing.T) {
	input := "Reviewing...\nRETRY: Fix error handling in auth.go\nMore notes"
	d := parseDecision(input)
	if d.Done {
		t.Error("parseDecision multiline RETRY should be Done=false")
	}
	if d.Feedback != "Fix error handling in auth.go" {
		t.Errorf("Feedback = %q; want %q", d.Feedback, "Fix error handling in auth.go")
	}
}
