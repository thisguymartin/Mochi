package orchestrator

import (
	"fmt"
	"strings"
	"testing"

	"github.com/thisguymartin/ai-forge/internal/config"
)

func TestCheckDependencies_AllPresent(t *testing.T) {
	origLookPath := lookPath
	defer func() { lookPath = origLookPath }()

	lookPath = func(name string) (string, error) {
		return "/usr/bin/" + name, nil
	}

	cfg := config.Config{Model: "claude-opus-4-6"}
	if err := checkDependencies(cfg); err != nil {
		t.Errorf("checkDependencies all present: unexpected error: %v", err)
	}
}

func TestCheckDependencies_GitMissing(t *testing.T) {
	origLookPath := lookPath
	defer func() { lookPath = origLookPath }()

	lookPath = func(name string) (string, error) {
		if name == "git" {
			return "", fmt.Errorf("not found")
		}
		return "/usr/bin/" + name, nil
	}

	cfg := config.Config{Model: "claude-opus-4-6"}
	err := checkDependencies(cfg)
	if err == nil {
		t.Fatal("expected error when git is missing, got nil")
	}
	if !strings.Contains(err.Error(), "git") {
		t.Errorf("error should mention git: %v", err)
	}
}

func TestCheckDependencies_GhNeededAndMissing(t *testing.T) {
	origLookPath := lookPath
	defer func() { lookPath = origLookPath }()

	lookPath = func(name string) (string, error) {
		if name == "gh" {
			return "", fmt.Errorf("not found")
		}
		return "/usr/bin/" + name, nil
	}

	cfg := config.Config{Model: "claude-opus-4-6", CreatePRs: true}
	err := checkDependencies(cfg)
	if err == nil {
		t.Fatal("expected error when gh is needed but missing, got nil")
	}
	if !strings.Contains(err.Error(), "gh") {
		t.Errorf("error should mention gh: %v", err)
	}
}

func TestCheckDependencies_GhNotNeededAndMissing(t *testing.T) {
	origLookPath := lookPath
	defer func() { lookPath = origLookPath }()

	lookPath = func(name string) (string, error) {
		if name == "gh" {
			return "", fmt.Errorf("not found")
		}
		return "/usr/bin/" + name, nil
	}

	cfg := config.Config{Model: "claude-opus-4-6", CreatePRs: false}
	if err := checkDependencies(cfg); err != nil {
		t.Errorf("checkDependencies gh not needed: unexpected error: %v", err)
	}
}

func TestCheckDependencies_GeminiModel(t *testing.T) {
	origLookPath := lookPath
	defer func() { lookPath = origLookPath }()

	lookPath = func(name string) (string, error) {
		if name == "gemini" {
			return "", fmt.Errorf("not found")
		}
		return "/usr/bin/" + name, nil
	}

	cfg := config.Config{Model: "gemini-2.5-pro"}
	err := checkDependencies(cfg)
	if err == nil {
		t.Fatal("expected error when gemini is missing, got nil")
	}
	if !strings.Contains(err.Error(), "gemini") {
		t.Errorf("error should mention gemini: %v", err)
	}
}
