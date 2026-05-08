package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// resetConfig sets config to known defaults and disables OpenRouter network calls.
func resetUnitTestState() {
	config = defaultConfig()
	config.Timezone = "UTC"
	openRouterPricingLoaded = true
	openRouterPricingCache = make(map[string]ModelPrice)
}

// ---------------------------------------------------------------------------
// defaultConfig
// ---------------------------------------------------------------------------

func TestDefaultConfig(t *testing.T) {
	cfg := defaultConfig()

	if cfg.DefaultModel != "gemini-1.5-flash" {
		t.Errorf("DefaultModel = %q, want %q", cfg.DefaultModel, "gemini-1.5-flash")
	}
	if cfg.Timezone != "Asia/Bangkok" {
		t.Errorf("Timezone = %q, want %q", cfg.Timezone, "Asia/Bangkok")
	}
	if cfg.Display.MaxLogsView != 50 {
		t.Errorf("MaxLogsView = %d, want 50", cfg.Display.MaxLogsView)
	}
	if !cfg.Display.ShowWorkspace {
		t.Error("ShowWorkspace should default to true")
	}
	if !cfg.Display.ReverseOrder {
		t.Error("ReverseOrder should default to true")
	}
	if cfg.Storage.Rotation != "monthly" {
		t.Errorf("Storage.Rotation = %q, want monthly", cfg.Storage.Rotation)
	}
	if cfg.Storage.LogFilePrefix != "atrack_logs" {
		t.Errorf("LogFilePrefix = %q, want atrack_logs", cfg.Storage.LogFilePrefix)
	}
	if cfg.TokenEstimation.CharsPerToken != 3.5 {
		t.Errorf("CharsPerToken = %v, want 3.5", cfg.TokenEstimation.CharsPerToken)
	}
	if !cfg.TokenEstimation.Enabled {
		t.Error("TokenEstimation.Enabled should default to true")
	}
	if cfg.Pricing.Models == nil {
		t.Error("Pricing.Models should not be nil")
	}
	if cfg.Pricing.Currency != "USD" {
		t.Errorf("Pricing.Currency = %q, want USD", cfg.Pricing.Currency)
	}
}

// ---------------------------------------------------------------------------
// normalizeTags
// ---------------------------------------------------------------------------

func TestNormalizeTags(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"empty string", "", nil},
		{"single tag", "go", []string{"go"}},
		{"multiple tags", "go,backend,api", []string{"go", "backend", "api"}},
		{"dedup case-insensitive", "Go,go,GO", []string{"Go"}},
		{"whitespace trimmed", " go , backend ", []string{"go", "backend"}},
		{"trailing comma ignored", "go,", []string{"go"}},
		{"only commas", ",,,", nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeTags(tc.input)
			if len(got) != len(tc.want) {
				t.Fatalf("normalizeTags(%q) = %v, want %v", tc.input, got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("normalizeTags(%q)[%d] = %q, want %q", tc.input, i, got[i], tc.want[i])
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// logModel / logCategory
// ---------------------------------------------------------------------------

func TestLogModel(t *testing.T) {
	resetUnitTestState()
	config.DefaultModel = "gemini-1.5-flash"

	if got := logModel(LogEntry{Model: "gpt-4"}); got != "gpt-4" {
		t.Errorf("explicit model: got %q, want gpt-4", got)
	}
	if got := logModel(LogEntry{}); got != "gemini-1.5-flash" {
		t.Errorf("empty model fallback: got %q, want gemini-1.5-flash", got)
	}
}

func TestLogCategory(t *testing.T) {
	if got := logCategory(LogEntry{Category: "Dev"}); got != "Dev" {
		t.Errorf("explicit category: got %q, want Dev", got)
	}
	if got := logCategory(LogEntry{}); got != "General" {
		t.Errorf("empty category fallback: got %q, want General", got)
	}
}

// ---------------------------------------------------------------------------
// logHasTag
// ---------------------------------------------------------------------------

func TestLogHasTag(t *testing.T) {
	log := LogEntry{Tags: []string{"Go", "backend"}}

	tests := []struct {
		tag  string
		want bool
	}{
		{"go", true},        // case-insensitive
		{"GO", true},        // uppercase
		{"back", true},      // partial match (substring)
		{"backend", true},   // exact match
		{"frontend", false}, // absent
		{"", true},          // empty always matches
	}
	for _, tc := range tests {
		if got := logHasTag(log, tc.tag); got != tc.want {
			t.Errorf("logHasTag(tag=%q) = %v, want %v", tc.tag, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// matchesKeyword
// ---------------------------------------------------------------------------

func TestMatchesKeyword(t *testing.T) {
	resetUnitTestState()
	config.DefaultModel = "test-model"

	log := LogEntry{
		Message:  "Fix the bug",
		Question: "What is wrong?",
		Answer:   "Nothing",
		Category: "Dev",
		Model:    "gpt-4",
		Tags:     []string{"backend"},
	}

	tests := []struct {
		keyword string
		want    bool
	}{
		{"", true},
		{"fix", true},
		{"BUG", true},       // case-insensitive
		{"what is", true},   // in question
		{"nothing", true},   // in answer
		{"dev", true},       // in category
		{"gpt-4", true},     // in model
		{"backend", true},   // in tags
		{"notfound", false}, // not in any field
	}
	for _, tc := range tests {
		if got := matchesKeyword(log, tc.keyword); got != tc.want {
			t.Errorf("matchesKeyword(%q) = %v, want %v", tc.keyword, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// matchesDateFilter
// ---------------------------------------------------------------------------

func TestMatchesDateFilter(t *testing.T) {
	resetUnitTestState()

	log := LogEntry{Timestamp: "2026-05-06 12:00:00"}

	t.Run("empty filter matches all", func(t *testing.T) {
		if !matchesDateFilter(log, DateFilter{}) {
			t.Error("empty filter should match")
		}
	})

	t.Run("exact date match", func(t *testing.T) {
		if !matchesDateFilter(log, DateFilter{Exact: "2026-05-06"}) {
			t.Error("exact match should succeed")
		}
	})

	t.Run("exact date mismatch", func(t *testing.T) {
		if matchesDateFilter(log, DateFilter{Exact: "2026-05-07"}) {
			t.Error("wrong exact date should fail")
		}
	})

	t.Run("from: same day passes", func(t *testing.T) {
		from, _ := time.ParseInLocation("2006-01-02", "2026-05-06", time.UTC)
		if !matchesDateFilter(log, DateFilter{From: &from}) {
			t.Error("from on same day should match")
		}
	})

	t.Run("from: later day blocks", func(t *testing.T) {
		from, _ := time.ParseInLocation("2006-01-02", "2026-05-07", time.UTC)
		if matchesDateFilter(log, DateFilter{From: &from}) {
			t.Error("from after log date should not match")
		}
	})

	t.Run("to: same day passes", func(t *testing.T) {
		to, _ := time.ParseInLocation("2006-01-02", "2026-05-06", time.UTC)
		if !matchesDateFilter(log, DateFilter{To: &to}) {
			t.Error("to on same day should match")
		}
	})

	t.Run("to: earlier day blocks", func(t *testing.T) {
		to, _ := time.ParseInLocation("2006-01-02", "2026-05-05", time.UTC)
		if matchesDateFilter(log, DateFilter{To: &to}) {
			t.Error("to before log date should not match")
		}
	})

	t.Run("invalid timestamp returns false", func(t *testing.T) {
		from, _ := time.ParseInLocation("2006-01-02", "2026-05-01", time.UTC)
		bad := LogEntry{Timestamp: "not-a-timestamp"}
		if matchesDateFilter(bad, DateFilter{From: &from}) {
			t.Error("invalid timestamp should not match a date filter")
		}
	})
}

// ---------------------------------------------------------------------------
// filterLogs
// ---------------------------------------------------------------------------

func TestFilterLogs(t *testing.T) {
	resetUnitTestState()

	logs := []LogEntry{
		{Timestamp: "2026-05-06 10:00:00", Category: "Dev", Model: "gpt-4", Tags: []string{"go"}},
		{Timestamp: "2026-05-04 10:00:00", Category: "Test", Model: "gpt-3.5"},
		{Timestamp: "2026-05-06 11:00:00", Category: "Dev", Model: "claude", Tags: []string{"ai"}},
	}

	t.Run("date range", func(t *testing.T) {
		from, _ := time.ParseInLocation("2006-01-02", "2026-05-05", time.UTC)
		to, _ := time.ParseInLocation("2006-01-02", "2026-05-07", time.UTC)
		got := filterLogs(logs, FilterOptions{DateFilter: DateFilter{From: &from, To: &to}})
		if len(got) != 2 {
			t.Errorf("date range: got %d logs, want 2", len(got))
		}
	})

	t.Run("model substring", func(t *testing.T) {
		got := filterLogs(logs, FilterOptions{Model: "gpt"})
		if len(got) != 2 {
			t.Errorf("model filter: got %d logs, want 2", len(got))
		}
	})

	t.Run("category case-insensitive", func(t *testing.T) {
		got := filterLogs(logs, FilterOptions{Category: "dev"})
		if len(got) != 2 {
			t.Errorf("category filter: got %d logs, want 2", len(got))
		}
	})

	t.Run("tag", func(t *testing.T) {
		got := filterLogs(logs, FilterOptions{Tag: "go"})
		if len(got) != 1 {
			t.Errorf("tag filter: got %d logs, want 1", len(got))
		}
	})

	t.Run("keyword in message", func(t *testing.T) {
		logs2 := []LogEntry{
			{Timestamp: "2026-05-06 10:00:00", Message: "fix auth bug"},
			{Timestamp: "2026-05-06 11:00:00", Message: "add feature"},
		}
		got := filterLogs(logs2, FilterOptions{Keyword: "bug"})
		if len(got) != 1 {
			t.Errorf("keyword filter: got %d logs, want 1", len(got))
		}
	})

	t.Run("no filter returns all", func(t *testing.T) {
		got := filterLogs(logs, FilterOptions{})
		if len(got) != len(logs) {
			t.Errorf("no filter: got %d logs, want %d", len(got), len(logs))
		}
	})
}

// ---------------------------------------------------------------------------
// parseOpenRouterRate
// ---------------------------------------------------------------------------

func TestParseOpenRouterRate(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"", 0},
		{"0.0005", 0.5},
		{"0.001", 1.0},
		{"0.0001", 0.1},
		{"abc", 0},
		{"0", 0},
	}
	for _, tc := range tests {
		got := parseOpenRouterRate(tc.input)
		if got != tc.want {
			t.Errorf("parseOpenRouterRate(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// registerOpenRouterAlias
// ---------------------------------------------------------------------------

func TestRegisterOpenRouterAlias(t *testing.T) {
	prices := make(map[string]ModelPrice)
	price := ModelPrice{InputPer1K: 0.5, OutputPer1K: 1.0}

	registerOpenRouterAlias(prices, "openai/gpt-4", price)
	if _, ok := prices["openai/gpt-4"]; !ok {
		t.Error("full id not registered")
	}
	if _, ok := prices["gpt-4"]; !ok {
		t.Error("short alias (after slash) not registered")
	}

	// Empty id should be ignored
	beforeLen := len(prices)
	registerOpenRouterAlias(prices, "", price)
	if len(prices) != beforeLen {
		t.Error("empty id should not add entry")
	}

	// ID without slash
	registerOpenRouterAlias(prices, "claude-3", price)
	if _, ok := prices["claude-3"]; !ok {
		t.Error("id without slash should be registered as-is")
	}
}

// ---------------------------------------------------------------------------
// sameModelPrice
// ---------------------------------------------------------------------------

func TestSameModelPrice(t *testing.T) {
	a := ModelPrice{InputPer1K: 0.5, OutputPer1K: 1.0}
	b := ModelPrice{InputPer1K: 0.5, OutputPer1K: 1.0}
	c := ModelPrice{InputPer1K: 0.6, OutputPer1K: 1.0}
	d := ModelPrice{InputPer1K: 0.5, OutputPer1K: 1.1}

	if !sameModelPrice(a, b) {
		t.Error("identical prices should be same")
	}
	if sameModelPrice(a, c) {
		t.Error("different input price should not be same")
	}
	if sameModelPrice(a, d) {
		t.Error("different output price should not be same")
	}
}

// ---------------------------------------------------------------------------
// parseBool
// ---------------------------------------------------------------------------

func TestParseBool(t *testing.T) {
	trueVals := []string{"true", "1", "yes", "on", "TRUE", "YES", "ON"}
	falseVals := []string{"false", "0", "no", "off", "FALSE", "NO", "OFF"}

	for _, v := range trueVals {
		got, err := parseBool(v)
		if err != nil || !got {
			t.Errorf("parseBool(%q) should be true, got=%v err=%v", v, got, err)
		}
	}
	for _, v := range falseVals {
		got, err := parseBool(v)
		if err != nil || got {
			t.Errorf("parseBool(%q) should be false, got=%v err=%v", v, got, err)
		}
	}
	if _, err := parseBool("invalid"); err == nil {
		t.Error("parseBool(invalid) should return an error")
	}
}

// ---------------------------------------------------------------------------
// parsePricingKey
// ---------------------------------------------------------------------------

func TestParsePricingKey(t *testing.T) {
	t.Run("input_per_1k", func(t *testing.T) {
		model, field, ok := parsePricingKey("pricing.gpt-4.input_per_1k")
		if !ok || model != "gpt-4" || field != "input_per_1k" {
			t.Errorf("got model=%q field=%q ok=%v", model, field, ok)
		}
	})

	t.Run("output_per_1k", func(t *testing.T) {
		model, field, ok := parsePricingKey("pricing.gpt-4.output_per_1k")
		if !ok || model != "gpt-4" || field != "output_per_1k" {
			t.Errorf("got model=%q field=%q ok=%v", model, field, ok)
		}
	})

	t.Run("non-pricing key", func(t *testing.T) {
		_, _, ok := parsePricingKey("default_model")
		if ok {
			t.Error("non-pricing key should return ok=false")
		}
	})

	t.Run("empty model segment", func(t *testing.T) {
		_, _, ok := parsePricingKey("pricing..input_per_1k")
		if ok {
			t.Error("empty model segment should return ok=false")
		}
	})

	t.Run("multi-level model name", func(t *testing.T) {
		model, field, ok := parsePricingKey("pricing.gemini-1.5-flash.output_per_1k")
		if !ok || model != "gemini-1.5-flash" || field != "output_per_1k" {
			t.Errorf("multi-dot model: model=%q field=%q ok=%v", model, field, ok)
		}
	})
}

// ---------------------------------------------------------------------------
// sameTags
// ---------------------------------------------------------------------------

func TestSameTags(t *testing.T) {
	tests := []struct {
		name string
		a, b []string
		want bool
	}{
		{"identical", []string{"go", "backend"}, []string{"go", "backend"}, true},
		{"case-insensitive", []string{"Go"}, []string{"go"}, true},
		{"both nil", nil, nil, true},
		{"different length", []string{"go"}, []string{"go", "backend"}, false},
		{"different values", []string{"go"}, []string{"backend"}, false},
		{"whitespace trimmed", []string{" go "}, []string{"go"}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := sameTags(tc.a, tc.b); got != tc.want {
				t.Errorf("sameTags(%v, %v) = %v, want %v", tc.a, tc.b, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// logEntriesMatch
// ---------------------------------------------------------------------------

func TestLogEntriesMatch(t *testing.T) {
	resetUnitTestState()

	base := LogEntry{
		Timestamp: "2026-05-06 10:00:00",
		Category:  "Dev",
		Message:   "hello",
		Model:     "gpt-4",
		Tags:      []string{"go"},
	}

	t.Run("identical", func(t *testing.T) {
		if !logEntriesMatch(base, base) {
			t.Error("identical entries should match")
		}
	})

	t.Run("different message", func(t *testing.T) {
		b := base
		b.Message = "world"
		if logEntriesMatch(base, b) {
			t.Error("different message should not match")
		}
	})

	t.Run("different timestamp", func(t *testing.T) {
		b := base
		b.Timestamp = "2026-05-07 10:00:00"
		if logEntriesMatch(base, b) {
			t.Error("different timestamp should not match")
		}
	})

	t.Run("different tags", func(t *testing.T) {
		b := base
		b.Tags = []string{"backend"}
		if logEntriesMatch(base, b) {
			t.Error("different tags should not match")
		}
	})

	t.Run("different token counts", func(t *testing.T) {
		b := base
		b.TokensIn = 999
		if logEntriesMatch(base, b) {
			t.Error("different token counts should not match")
		}
	})
}

// ---------------------------------------------------------------------------
// estimateTokens
// ---------------------------------------------------------------------------

func TestEstimateTokens(t *testing.T) {
	resetUnitTestState()
	config.TokenEstimation = TokenEstimationConfig{Enabled: true, CharsPerToken: 4.0}

	// 4 chars / 4.0 = 1.0 → ceil = 1
	if got := estimateTokens("1234"); got != 1 {
		t.Errorf("estimateTokens(4 chars): got %d, want 1", got)
	}
	// 5 chars / 4.0 = 1.25 → ceil = 2
	if got := estimateTokens("12345"); got != 2 {
		t.Errorf("estimateTokens(5 chars): got %d, want 2", got)
	}
	// empty text → 0
	if got := estimateTokens(""); got != 0 {
		t.Errorf("estimateTokens(empty): got %d, want 0", got)
	}
	// disabled → 0
	config.TokenEstimation.Enabled = false
	if got := estimateTokens("hello"); got != 0 {
		t.Errorf("estimateTokens(disabled): got %d, want 0", got)
	}
}

// ---------------------------------------------------------------------------
// calculateLogCost
// ---------------------------------------------------------------------------

func TestCalculateLogCost(t *testing.T) {
	resetUnitTestState()
	config.Pricing = PricingConfig{
		Currency: "USD",
		Models: map[string]ModelPrice{
			"gpt-4": {InputPer1K: 1.0, OutputPer1K: 2.0},
		},
	}

	t.Run("known model", func(t *testing.T) {
		log := LogEntry{Model: "gpt-4", TokensIn: 1000, TokensOut: 500}
		cost, ok := calculateLogCost(log)
		if !ok {
			t.Fatal("expected ok=true for known model")
		}
		// (1000/1000)*1.0 + (500/1000)*2.0 = 1.0 + 1.0 = 2.0
		want := 2.0
		if cost != want {
			t.Errorf("calculateLogCost: got %v, want %v", cost, want)
		}
	})

	t.Run("unknown model", func(t *testing.T) {
		log := LogEntry{Model: "unknown-xyz", TokensIn: 100, TokensOut: 100}
		_, ok := calculateLogCost(log)
		if ok {
			t.Error("unknown model should return ok=false")
		}
	})

	t.Run("zero tokens", func(t *testing.T) {
		log := LogEntry{Model: "gpt-4", TokensIn: 0, TokensOut: 0}
		cost, ok := calculateLogCost(log)
		if !ok {
			t.Fatal("expected ok=true")
		}
		if cost != 0 {
			t.Errorf("zero tokens cost: got %v, want 0", cost)
		}
	})
}

// ---------------------------------------------------------------------------
// collectModelStats
// ---------------------------------------------------------------------------

func TestCollectModelStats(t *testing.T) {
	resetUnitTestState()
	config.Pricing = PricingConfig{
		Currency: "USD",
		Models: map[string]ModelPrice{
			"gpt-4": {InputPer1K: 1.0, OutputPer1K: 2.0},
		},
	}
	config.DefaultModel = "gpt-4"

	logs := []LogEntry{
		{Model: "gpt-4", TokensIn: 100, TokensOut: 50},
		{Model: "gpt-4", TokensIn: 200, TokensOut: 100},
		{Model: "claude", TokensIn: 300, TokensOut: 150},
	}

	rows := collectModelStats(logs)
	if len(rows) != 2 {
		t.Fatalf("collectModelStats: got %d rows, want 2", len(rows))
	}

	// gpt-4 has 2 logs → first
	if rows[0].Model != "gpt-4" {
		t.Errorf("first row model: got %q, want gpt-4", rows[0].Model)
	}
	if rows[0].Logs != 2 {
		t.Errorf("gpt-4 log count: got %d, want 2", rows[0].Logs)
	}
	if rows[0].TokensIn != 300 || rows[0].TokensOut != 150 {
		t.Errorf("gpt-4 tokens: In=%d Out=%d", rows[0].TokensIn, rows[0].TokensOut)
	}
	if !rows[0].HasCost {
		t.Error("gpt-4 should have cost computed")
	}
	if rows[1].Model != "claude" {
		t.Errorf("second row model: got %q, want claude", rows[1].Model)
	}
	if rows[1].HasCost {
		t.Error("claude should not have cost (no pricing configured)")
	}
}

// ---------------------------------------------------------------------------
// parseDateFilters
// ---------------------------------------------------------------------------

func TestParseDateFilters(t *testing.T) {
	resetUnitTestState()

	t.Run("no flags, remaining preserved", func(t *testing.T) {
		filter, remaining, err := parseDateFilters([]string{"keyword"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(remaining) != 1 || remaining[0] != "keyword" {
			t.Errorf("remaining: %v", remaining)
		}
		if filter.From != nil || filter.To != nil {
			t.Error("no flags should leave From/To nil")
		}
	})

	t.Run("--from sets From", func(t *testing.T) {
		filter, remaining, err := parseDateFilters([]string{"--from", "2026-05-01", "keyword"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if filter.From == nil {
			t.Error("--from should set From")
		}
		if len(remaining) != 1 || remaining[0] != "keyword" {
			t.Errorf("remaining: %v", remaining)
		}
	})

	t.Run("--to sets To", func(t *testing.T) {
		filter, _, err := parseDateFilters([]string{"--to", "2026-05-31"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if filter.To == nil {
			t.Error("--to should set To")
		}
	})

	t.Run("both flags", func(t *testing.T) {
		filter, remaining, err := parseDateFilters([]string{"--from", "2026-05-01", "--to", "2026-05-31"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if filter.From == nil || filter.To == nil {
			t.Error("both From and To should be set")
		}
		if len(remaining) != 0 {
			t.Errorf("remaining should be empty: %v", remaining)
		}
	})

	t.Run("invalid --from date", func(t *testing.T) {
		_, _, err := parseDateFilters([]string{"--from", "not-a-date"})
		if err == nil {
			t.Error("invalid date should return error")
		}
	})

	t.Run("missing value for --from", func(t *testing.T) {
		_, _, err := parseDateFilters([]string{"--from"})
		if err == nil {
			t.Error("missing value should return error")
		}
	})

	t.Run("missing value for --to", func(t *testing.T) {
		_, _, err := parseDateFilters([]string{"--to"})
		if err == nil {
			t.Error("missing value should return error")
		}
	})
}

// ---------------------------------------------------------------------------
// getLogPath
// ---------------------------------------------------------------------------

func TestGetLogPath(t *testing.T) {
	dir := t.TempDir()
	appDir = dir
	config.Storage = StorageConfig{
		LogFilePrefix: "atrack_logs",
		Rotation:      "monthly",
	}

	ts := time.Date(2026, 5, 8, 10, 0, 0, 0, time.UTC)

	got := getLogPath(ts)
	want := filepath.Join(dir, "atrack_logs_2026_05.json")
	if got != want {
		t.Errorf("monthly rotation: got %q, want %q", got, want)
	}

	config.Storage.Rotation = "none"
	got = getLogPath(ts)
	want = filepath.Join(dir, "atrack_logs.json")
	if got != want {
		t.Errorf("no rotation: got %q, want %q", got, want)
	}
}

// ---------------------------------------------------------------------------
// getAppDir
// ---------------------------------------------------------------------------

func TestGetAppDir(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("ATRACK_HOME", dir)
	defer os.Unsetenv("ATRACK_HOME")

	got := getAppDir()
	if got != dir {
		t.Errorf("getAppDir with ATRACK_HOME: got %q, want %q", got, dir)
	}
}

func TestGetAppDir_Default(t *testing.T) {
	os.Unsetenv("ATRACK_HOME")

	got := getAppDir()
	if got == "" || got == "." {
		// "." is the fallback when home dir cannot be determined; acceptable in CI
		t.Logf("getAppDir fallback returned %q (acceptable if home dir unavailable)", got)
	}
	// Should be a valid path (non-empty)
	if got == "" {
		t.Error("getAppDir should never return empty string")
	}
}

// ---------------------------------------------------------------------------
// getAllLogFiles / getLogsFromAllFiles / saveLogsToFile
// ---------------------------------------------------------------------------

func TestGetLogsFromAllFiles(t *testing.T) {
	dir := t.TempDir()
	appDir = dir
	config.Storage = StorageConfig{
		LogFilePrefix: "atrack_logs",
		Rotation:      "none",
	}

	// Write two log files
	file1Logs := []LogEntry{
		{Timestamp: "2026-05-05 10:00:00", Message: "first"},
	}
	file2Logs := []LogEntry{
		{Timestamp: "2026-05-06 10:00:00", Message: "second"},
	}
	data1, _ := json.MarshalIndent(file1Logs, "", "  ")
	data2, _ := json.MarshalIndent(file2Logs, "", "  ")

	os.WriteFile(filepath.Join(dir, "atrack_logs.json"), data1, 0644)
	os.WriteFile(filepath.Join(dir, "atrack_logs_extra.json"), data2, 0644)

	got := getLogsFromAllFiles()
	if len(got) != 2 {
		t.Errorf("expected 2 logs from 2 files, got %d", len(got))
	}
}

func TestSaveLogsToFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test_logs.json")

	logs := []LogEntry{
		{Timestamp: "2026-05-06 10:00:00", Message: "saved"},
	}
	saveLogsToFile(path, logs)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	var readBack []LogEntry
	if err := json.Unmarshal(data, &readBack); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(readBack) != 1 || readBack[0].Message != "saved" {
		t.Errorf("unexpected content: %v", readBack)
	}
}

// ---------------------------------------------------------------------------
// getConfigValue
// ---------------------------------------------------------------------------

func TestGetConfigValue(t *testing.T) {
	resetUnitTestState()
	config.Pricing.Models = map[string]ModelPrice{
		"gpt-4": {InputPer1K: 1.5, OutputPer1K: 2.5},
	}

	tests := []struct {
		key  string
		want string
	}{
		{"default_model", "gemini-1.5-flash"},
		{"model", "gemini-1.5-flash"},
		{"timezone", "UTC"},
		{"token_estimation.enabled", "true"},
		{"token_estimation.chars_per_token", "3.5"},
		{"display.max_logs_view", "50"},
		{"display.show_workspace", "true"},
		{"display.reverse_order", "true"},
		{"storage.rotation", "monthly"},
		{"storage.log_file_prefix", "atrack_logs"},
		{"pricing.currency", "USD"},
	}
	for _, tc := range tests {
		got, err := getConfigValue(tc.key)
		if err != nil {
			t.Errorf("getConfigValue(%q): unexpected error %v", tc.key, err)
			continue
		}
		if got != tc.want {
			t.Errorf("getConfigValue(%q) = %q, want %q", tc.key, got, tc.want)
		}
	}

	t.Run("pricing model input", func(t *testing.T) {
		val, err := getConfigValue("pricing.gpt-4.input_per_1k")
		if err != nil || val != "1.5" {
			t.Errorf("pricing input: got %q err=%v", val, err)
		}
	})

	t.Run("pricing model output", func(t *testing.T) {
		val, err := getConfigValue("pricing.gpt-4.output_per_1k")
		if err != nil || val != "2.5" {
			t.Errorf("pricing output: got %q err=%v", val, err)
		}
	})

	t.Run("unknown key returns error", func(t *testing.T) {
		_, err := getConfigValue("totally.unknown.key")
		if err == nil {
			t.Error("unknown key should return an error")
		}
	})

	t.Run("pricing for unset model returns error", func(t *testing.T) {
		_, err := getConfigValue("pricing.no-such-model.input_per_1k")
		if err == nil {
			t.Error("unset model pricing should return an error")
		}
	})
}

// ---------------------------------------------------------------------------
// ListAllTimezones
// ---------------------------------------------------------------------------

func TestListAllTimezones(t *testing.T) {
	zones := ListAllTimezones()
	if len(zones) == 0 {
		t.Error("ListAllTimezones should return at least one timezone")
	}

	found := false
	for _, z := range zones {
		if z == "Asia/Bangkok" || z == "UTC" || z == "Asia/Tokyo" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("ListAllTimezones did not contain expected zones; got: %v", zones)
	}
}

// ---------------------------------------------------------------------------
// configPath
// ---------------------------------------------------------------------------

func TestConfigPath(t *testing.T) {
	dir := t.TempDir()
	appDir = dir

	got := configPath()
	want := filepath.Join(dir, "config.json")
	if got != want {
		t.Errorf("configPath: got %q, want %q", got, want)
	}
}
