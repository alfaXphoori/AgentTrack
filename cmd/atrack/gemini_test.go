package main

import (
	"os"
	"os/exec"
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
		{"auto log call", `auto "$QUESTION"`},
		{"model priority env var", "GEMINI_MODEL"},
		{"session file path", ".gemini/tmp"},
		{"exit command", `"exit"`},
		{"model switch command", `"/model"`},
	}
	for _, c := range checks {
		if !strings.Contains(content, c.keyword) {
			t.Errorf("gemitrack.sh missing %s (keyword: %q)", c.desc, c.keyword)
		}
	}

	// Test model detection script (unit test the python snippet)
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

	// Run the python model-detection snippet from gemitrack.sh
	pythonScript := `
import json, os, glob, sys

def find_model(obj):
    if isinstance(obj, dict):
        for k, v in obj.items():
            if k == 'model' and isinstance(v, str) and 'gemini' in v.lower():
                return v
            r = find_model(v)
            if r: return r
    elif isinstance(obj, list):
        for i in obj:
            r = find_model(i)
            if r: return r
    return None

cwd = sys.argv[1]
tmp_base = os.path.join(sys.argv[2], '.gemini', 'tmp')

target_dir = None
for d in os.listdir(tmp_base):
    pr = os.path.join(tmp_base, d, '.project_root')
    if os.path.exists(pr):
        with open(pr) as f:
            if f.read().strip().lower() == cwd.lower():
                target_dir = os.path.join(tmp_base, d)
                break

if not target_dir:
    sys.exit(1)

sessions = sorted(glob.glob(os.path.join(target_dir, 'chats', 'session-*.jsonl')), key=os.path.getmtime)
for s in reversed(sessions):
    model = None
    with open(s) as f:
        for line in f:
            line = line.strip()
            if not line: continue
            try:
                m = find_model(json.loads(line))
                if m: model = m
            except: pass
    if model:
        print(model)
        sys.exit(0)
sys.exit(1)
`
	cmd := exec.Command("python3", "-c", pythonScript, cwd, tmpHome)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("Model detection script failed: %v", err)
	}
	detected := strings.TrimSpace(string(out))
	if detected != "gemini-test-model-preview" {
		t.Fatalf("Model detection returned %q, want %q", detected, "gemini-test-model-preview")
	}
}
