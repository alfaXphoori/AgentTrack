package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func setupTestDB() string {
	cwd, _ := os.Getwd()
	dbPath := filepath.Join(cwd, "atrack_logs.json")
	bakPath := filepath.Join(cwd, "atrack_logs.json.bak")
	if _, err := os.Stat(dbPath); err == nil {
		data, _ := os.ReadFile(dbPath)
		os.WriteFile(bakPath, data, 0644)
	}
	return bakPath
}

func restoreTestDB(bakPath string) {
	cwd, _ := os.Getwd()
	dbPath := filepath.Join(cwd, "atrack_logs.json")
	if _, err := os.Stat(bakPath); err == nil {
		data, _ := os.ReadFile(bakPath)
		os.WriteFile(dbPath, data, 0644)
		os.Remove(bakPath)
	} else {
		os.Remove(dbPath)
	}
}

func readLogsFile(t *testing.T, path string) []LogEntry {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read logs file: %v", err)
	}

	var logs []LogEntry
	if err := json.Unmarshal(data, &logs); err != nil {
		t.Fatalf("Failed to decode logs: %v", err)
	}
	return logs
}

func writeLogsFile(t *testing.T, path string, logs []LogEntry) {
	t.Helper()
	data, err := json.MarshalIndent(logs, "", "  ")
	if err != nil {
		t.Fatalf("Failed to encode logs: %v", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("Failed to write logs: %v", err)
	}
}

func exportPathFromOutput(output string) string {
	const prefix = "Logs exported successfully to: "
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(line, prefix))
		}
	}
	return ""
}

func TestTracker(t *testing.T) {
	bakPath := setupTestDB()
	defer restoreTestDB(bakPath)

	cwd, _ := os.Getwd()

	cmd := exec.Command("go", "build", "-o", "tracker_test_bin.exe", ".")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build: %v", err)
	}
	defer os.Remove("tracker_test_bin.exe")

	openRouterURL := ""
	runCmd := func(args ...string) (string, error) {
		cmd := exec.Command("./tracker_test_bin.exe", args...)
		env := append(os.Environ(), "ATRACK_HOME="+cwd)
		if openRouterURL != "" {
			env = append(env, "ATRACK_OPENROUTER_MODELS_URL="+openRouterURL)
		}
		cmd.Env = env
		out, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("Command failed: %s %v\nOutput: %s\n", strings.Join(args, " "), err, string(out))
		}
		return string(out), err
	}

	testConfig := `{
		"storage": {
			"log_file_prefix": "atrack_logs",
			"rotation": "none"
		}
	}`
	os.WriteFile(filepath.Join(cwd, "config.json"), []byte(testConfig), 0644)
	defer os.Remove(filepath.Join(cwd, "config.json"))

	if _, err := runCmd("clear"); err != nil {
		t.Fatalf("Failed to clear logs: %v", err)
	}

	dbPath := filepath.Join(cwd, "atrack_logs.json")

	if _, err := runCmd("log", "Test message", "-c", "TestCategory", "-t", "bug,backend"); err != nil {
		t.Fatalf("Log command failed: %v", err)
	}
	if _, err := runCmd("log", "Older message", "-c", "Bugfix", "-t", "bug,go"); err != nil {
		t.Fatalf("Second log command failed: %v", err)
	}
	if _, err := runCmd("auto", "User Question", "AI Answer", "test-model", "10", "20"); err != nil {
		t.Fatalf("Auto command failed: %v", err)
	}
	if _, err := runCmd("config", "set", "pricing.gemini-1.5-flash.input_per_1k", "0.10"); err != nil {
		t.Fatalf("Config pricing input set failed: %v", err)
	}
	if _, err := runCmd("config", "set", "pricing.gemini-1.5-flash.output_per_1k", "0.20"); err != nil {
		t.Fatalf("Config pricing output set failed: %v", err)
	}
	if _, err := runCmd("config", "set", "pricing.test-model.input_per_1k", "0.30"); err != nil {
		t.Fatalf("Config test-model input set failed: %v", err)
	}
	if _, err := runCmd("config", "set", "pricing.test-model.output_per_1k", "0.40"); err != nil {
		t.Fatalf("Config test-model output set failed: %v", err)
	}

	logs := readLogsFile(t, dbPath)
	if len(logs) != 3 {
		t.Fatalf("Expected 3 logs, got %d", len(logs))
	}
	if logs[0].Category != "TestCategory" || len(logs[0].Tags) != 2 {
		t.Fatalf("First log missing category or tags: %+v", logs[0])
	}
	if logs[2].Category != "AutoLog" || logs[2].Question != "User Question" {
		t.Fatalf("Auto log not stored as expected: %+v", logs[2])
	}

	logs[0].Timestamp = "2026-05-02 10:00:00"
	logs[1].Timestamp = "2026-05-01 09:00:00"
	logs[2].Timestamp = "2026-05-06 12:00:00"
	writeLogsFile(t, dbPath, logs)

	out, err := runCmd("list")
	if err != nil {
		t.Fatalf("List command failed: %v", err)
	}
	if !strings.Contains(out, "Test message") || !strings.Contains(out, "Older message") || !strings.Contains(out, "User Question") {
		t.Fatalf("List output missing expected data: %s", out)
	}

	out, err = runCmd("list", "--from", "2026-05-02", "--to", "2026-05-06")
	if err != nil {
		t.Fatalf("List range command failed: %v (%s)", err, out)
	}
	if !strings.Contains(out, "Test message") || !strings.Contains(out, "User Question") || strings.Contains(out, "Older message") {
		t.Fatalf("List range output unexpected: %s", out)
	}

	out, err = runCmd("list", "model", "test-model")
	if err != nil {
		t.Fatalf("List by model failed: %v (%s)", err, out)
	}
	if !strings.Contains(out, "User Question") || strings.Contains(out, "Test message") {
		t.Fatalf("List by model output unexpected: %s", out)
	}

	out, err = runCmd("list", "model", "all")
	if err != nil {
		t.Fatalf("List all models failed: %v (%s)", err, out)
	}
	if !strings.Contains(out, "Model") || !strings.Contains(out, "Count") || !strings.Contains(out, "Tokens") || !strings.Contains(out, "Cost") ||
		!strings.Contains(out, "test-model") || !strings.Contains(out, "gemini-1.5-flash") ||
		strings.Contains(out, "User Question") || strings.Contains(out, "Test message") {
		t.Fatalf("List all models output unexpected: %s", out)
	}

	out, err = runCmd("list", "category", "TestCategory")
	if err != nil {
		t.Fatalf("List by category failed: %v (%s)", err, out)
	}
	if !strings.Contains(out, "Test message") || strings.Contains(out, "Older message") {
		t.Fatalf("List by category output unexpected: %s", out)
	}

	out, err = runCmd("list", "category", "all")
	if err != nil {
		t.Fatalf("List all categories failed: %v (%s)", err, out)
	}
	if !strings.Contains(out, "Category") || !strings.Contains(out, "TestCategory") || !strings.Contains(out, "Bugfix") || !strings.Contains(out, "AutoLog") {
		t.Fatalf("List all categories output unexpected: %s", out)
	}

	out, err = runCmd("search", "message", "--from", "2026-05-02", "--to", "2026-05-06")
	if err != nil {
		t.Fatalf("Search with date range failed: %v (%s)", err, out)
	}
	if !strings.Contains(out, "Test message") || strings.Contains(out, "Older message") {
		t.Fatalf("Search date range output unexpected: %s", out)
	}

	out, err = runCmd("search", "model", "test-model")
	if err != nil {
		t.Fatalf("Search by model failed: %v (%s)", err, out)
	}
	if !strings.Contains(out, "User Question") || strings.Contains(out, "Test message") {
		t.Fatalf("Search by model output unexpected: %s", out)
	}

	out, err = runCmd("search", "tag", "bug")
	if err != nil {
		t.Fatalf("Search by tag failed: %v (%s)", err, out)
	}
	if !strings.Contains(out, "Test message") || !strings.Contains(out, "Older message") || strings.Contains(out, "User Question") {
		t.Fatalf("Search by tag output unexpected: %s", out)
	}

	out, err = runCmd("edit", "0", "Corrected message")
	if err != nil {
		t.Fatalf("Edit command failed: %v (%s)", err, out)
	}
	logs = readLogsFile(t, dbPath)
	if logs[0].Message != "Corrected message" {
		t.Fatalf("Edit command did not update the message: %+v", logs[0])
	}

	out, err = runCmd("config", "get", "pricing.test-model.input_per_1k")
	if err != nil {
		t.Fatalf("Config pricing get failed: %v (%s)", err, out)
	}
	if !strings.Contains(out, "pricing.test-model.input_per_1k = 0.3") {
		t.Fatalf("Pricing config get output unexpected: %s", out)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"data": [
				{"id": "openai/test-model", "pricing": {"prompt": "0.0005", "completion": "0.0007"}},
				{"id": "google/gemini-1.5-flash", "pricing": {"prompt": "0.0001", "completion": "0.0002"}}
			]
		}`))
	}))
	defer server.Close()
	openRouterURL = server.URL

	out, err = runCmd("pricing", "sync", "test-model", "gemini-1.5-flash", "missing-model")
	if err != nil {
		t.Fatalf("Pricing sync failed: %v (%s)", err, out)
	}
	if !strings.Contains(out, "Updated: 1") || !strings.Contains(out, "Unchanged: 1") || !strings.Contains(out, "Missing: 1") {
		t.Fatalf("Pricing sync output unexpected: %s", out)
	}

	out, err = runCmd("config", "get", "pricing.test-model.input_per_1k")
	if err != nil {
		t.Fatalf("Config get after pricing sync failed: %v (%s)", err, out)
	}
	if !strings.Contains(out, "pricing.test-model.input_per_1k = 0.5") {
		t.Fatalf("Synced test-model price not saved: %s", out)
	}

	out, err = runCmd("config", "get", "pricing.gemini-1.5-flash.output_per_1k")
	if err != nil {
		t.Fatalf("Config get after gemini pricing sync failed: %v (%s)", err, out)
	}
	if !strings.Contains(out, "pricing.gemini-1.5-flash.output_per_1k = 0.2") {
		t.Fatalf("Synced gemini price not saved: %s", out)
	}

	out, err = runCmd("pricing", "sync", "test-model")
	if err != nil {
		t.Fatalf("Second pricing sync failed: %v (%s)", err, out)
	}
	if !strings.Contains(out, "No changes needed") {
		t.Fatalf("Second pricing sync should report no changes: %s", out)
	}

	out, err = runCmd("stats", "model")
	if err != nil {
		t.Fatalf("Stats model failed: %v (%s)", err, out)
	}
	if !strings.Contains(out, "test-model") || !strings.Contains(out, "gemini-1.5-flash") || !strings.Contains(out, "Cost") {
		t.Fatalf("Stats model output unexpected: %s", out)
	}

	out, err = runCmd("stats", "cost")
	if err != nil {
		t.Fatalf("Stats cost failed: %v (%s)", err, out)
	}
	if !strings.Contains(out, "Total Estimated Cost") || !strings.Contains(out, "test-model") {
		t.Fatalf("Stats cost output unexpected: %s", out)
	}

	out, err = runCmd("export", "csv")
	if err != nil {
		t.Fatalf("CSV export failed: %v (%s)", err, out)
	}
	csvPath := exportPathFromOutput(out)
	if csvPath == "" {
		t.Fatalf("Could not parse CSV export path from output: %s", out)
	}
	defer os.Remove(csvPath)
	csvData, err := os.ReadFile(csvPath)
	if err != nil {
		t.Fatalf("Failed to read CSV export: %v", err)
	}
	if !strings.Contains(string(csvData), "timestamp,category,message") || !strings.Contains(string(csvData), "Corrected message") {
		t.Fatalf("CSV export content unexpected: %s", string(csvData))
	}

	out, err = runCmd("export", "json")
	if err != nil {
		t.Fatalf("JSON export failed: %v (%s)", err, out)
	}
	jsonPath := exportPathFromOutput(out)
	if jsonPath == "" {
		t.Fatalf("Could not parse JSON export path from output: %s", out)
	}
	defer os.Remove(jsonPath)
	jsonData, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("Failed to read JSON export: %v", err)
	}
	if !strings.Contains(string(jsonData), "\"Corrected message\"") || !strings.Contains(string(jsonData), "\"tags\"") {
		t.Fatalf("JSON export content unexpected: %s", string(jsonData))
	}

	out, err = runCmd()
	if err != nil {
		t.Fatalf("Usage output failed: %v", err)
	}
	if !strings.Contains(out, "Agent Track: The Cross-Platform AI Activity Tracker") ||
		!strings.Contains(out, `atrack log "message"`) ||
		!strings.Contains(out, `atrack list`) ||
		!strings.Contains(out, `atrack dashboard`) ||
		!strings.Contains(out, `atrack stats`) ||
		!strings.Contains(out, `atrack summary`) {
		t.Fatalf("Usage output missing essential commands: %s", out)
	}

	out, err = runCmd("help")
	if err != nil {
		t.Fatalf("Full usage output failed: %v", err)
	}
	if !strings.Contains(out, "Agent Track: Detailed Help") ||
		!strings.Contains(out, `atrack log "message" [-c category] [-t tag1,tag2]`) ||
		!strings.Contains(out, `atrack list category "name"|all`) ||
		!strings.Contains(out, `atrack search model|tag "value"`) ||
		!strings.Contains(out, `atrack edit <index> [field] <value>`) ||
		!strings.Contains(out, `atrack stats | model | cost | today`) ||
		!strings.Contains(out, `atrack pricing sync [all|model]`) ||
		!strings.Contains(out, `atrack export [md|csv|json]`) ||
		!strings.Contains(out, "atrack config [show|get|set|reset]") {
		t.Fatalf("Full usage output missing new commands: %s", out)
	}

	out, err = runCmd("config", "set", "display.max_logs_view", "25")
	if err != nil {
		t.Fatalf("Config set failed: %v (%s)", err, out)
	}

	out, err = runCmd("config", "get", "display.max_logs_view")
	if err != nil {
		t.Fatalf("Config get failed: %v (%s)", err, out)
	}
	if !strings.Contains(out, "display.max_logs_view = 25") {
		t.Fatalf("Config get output unexpected: %s", out)
	}

	out, err = runCmd("config", "reset")
	if err != nil {
		t.Fatalf("Config reset failed: %v (%s)", err, out)
	}

	out, err = runCmd("config", "get", "display.max_logs_view")
	if err != nil {
		t.Fatalf("Config get after reset failed: %v (%s)", err, out)
	}
	if !strings.Contains(out, "display.max_logs_view = 50") {
		t.Fatalf("Config reset did not restore defaults: %s", out)
	}
}
