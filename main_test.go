package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func setupTestDB() string {
	cwd, _ := os.Getwd()
	dbPath := filepath.Join(cwd, "aikore_logs.json")
	bakPath := filepath.Join(cwd, "aikore_logs.json.bak")
	if _, err := os.Stat(dbPath); err == nil {
		data, _ := os.ReadFile(dbPath)
		os.WriteFile(bakPath, data, 0644)
	}
	return bakPath
}

func restoreTestDB(bakPath string) {
	cwd, _ := os.Getwd()
	dbPath := filepath.Join(cwd, "aikore_logs.json")
	if _, err := os.Stat(bakPath); err == nil {
		data, _ := os.ReadFile(bakPath)
		os.WriteFile(dbPath, data, 0644)
		os.Remove(bakPath)
	} else {
		os.Remove(dbPath)
	}
}

func TestTracker(t *testing.T) {
	bakPath := setupTestDB()
	defer restoreTestDB(bakPath)

	// Build the executable to run tests independently
	cmd := exec.Command("go", "build", "-o", "tracker_test_bin.exe", ".")
	err := cmd.Run()
	if err != nil {
		t.Fatalf("Failed to build: %v", err)
	}
	defer os.Remove("tracker_test_bin.exe")

	runCmd := func(args ...string) (string, error) {
		cmd := exec.Command("./tracker_test_bin.exe", args...)
		out, err := cmd.CombinedOutput()
		return string(out), err
	}

	// Clear logs
	_, err = runCmd("clear")
	if err != nil {
		t.Fatalf("Failed to clear logs: %v", err)
	}

	cwd, _ := os.Getwd()
	dbPath := filepath.Join(cwd, "aikore_logs.json")
	
	// Test log
	runCmd("log", "Test message", "-c", "TestCategory")
	data, _ := os.ReadFile(dbPath)
	var logs []LogEntry
	json.Unmarshal(data, &logs)
	if len(logs) != 1 || logs[0].Message != "Test message" || logs[0].Category != "TestCategory" {
		t.Fatalf("Log command failed")
	}

	// Test auto
	runCmd("auto", "User Question", "AI Answer", "test-model", "10", "20")
	data, _ = os.ReadFile(dbPath)
	json.Unmarshal(data, &logs)
	if len(logs) != 2 || logs[1].Category != "AutoLog" || logs[1].Question != "User Question" {
		t.Fatalf("Auto command failed")
	}

	// Test list
	out, err := runCmd("list")
	if err != nil {
		t.Fatalf("List command failed: %v", err)
	}
	if !strings.Contains(out, "Test message") || !strings.Contains(out, "User Question") {
		t.Fatalf("List output missing expected data: %s", out)
	}
}
