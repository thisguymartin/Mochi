package provider

import "testing"

func TestDetect(t *testing.T) {
	tests := []struct {
		model string
		want  string
	}{
		{model: "gemini-2.5-pro", want: "gemini"},
		{model: "claude-opus-4-6", want: "claude"},
		{model: "gpt-5", want: "codex"},
		{model: "o3", want: "codex"},
		{model: "codex-mini", want: "codex"},
		{model: "unknown-model", want: "claude"},
	}

	for _, tc := range tests {
		if got := Detect(tc.model).Name; got != tc.want {
			t.Errorf("Detect(%q)=%q want %q", tc.model, got, tc.want)
		}
	}
}
