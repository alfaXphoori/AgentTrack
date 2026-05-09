package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGemitrack(t *testing.T) {
	cwd, _ := os.Getwd()
	// The scripts directory is at the project root, two levels up from cmd/atrack
	root := filepath.Join(cwd, "..", "..")

	// Check gemitrack.sh exists and is executable
	scriptPath := filepath.Join(root, "scripts", "gemiatrack.sh")
	info, err := os.Stat(scriptPath)
	if err != nil {
		t.Fatalf("gemitrack.sh not found: %v", err)
	}
	if info.Mode()&0111 == 0 {
		t.Fatalf("gemitrack.sh is not executable")
	}

	// Validate script contains required logic
	data, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("Failed to read gemitrack.sh: %v", err)
	}
	content := string(data)

	checks := []struct {
		desc    string
		keyword string
	}{
		{"auto-detect model function", "detect_live_model"},
		{"atrack binary reference", "ATRACK_BIN"},
		{"model priority env var", "GEMINI_MODEL"},
		{"exit command", `"exit"`},
		{"model switch command", `"/model"`},
	}
	for _, c := range checks {
		if !strings.Contains(content, c.keyword) {
			t.Errorf("gemitrack.sh missing %s (keyword: %q)", c.desc, c.keyword)
		}
	}

	// Test model detection logic (Go native)
	tmpHome := t.TempDir()

	// Create a fake gemini session structure with a model field
	sessionDir := filepath.Join(tmpHome, ".gemini", "tmp", "testproject", "chats")
	os.MkdirAll(sessionDir, 0755)

	// Write .project_root
	os.WriteFile(filepath.Join(tmpHome, ".gemini", "tmp", "testproject", ".project_root"), []byte(cwd), 0644)

	// Write a fake session jsonl with model field
	sessionFile := filepath.Join(sessionDir, "session-2026-05-07T10-00-abcd1234.jsonl")
	sessionData := `{"sessionId":"abcd1234","kind":"main"}` + "\n" +
		`{"type":"user","content":"hello","model":"gemini-test-model-preview"}` + "\n"
	os.WriteFile(sessionFile, []byte(sessionData), 0644)

	// Run the Go model-detection snippet
	detected := runDetectGeminiModel(cwd, tmpHome)
	
	if detected != "gemini-test-model-preview" {
		t.Fatalf("Model detection returned %q, want %q", detected, "gemini-test-model-preview")
	}
}
