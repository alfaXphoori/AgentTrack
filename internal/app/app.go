package app

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gofrs/flock"
)

type TokenEstimationConfig struct {
	Enabled       bool    `json:"enabled"`
	CharsPerToken float64 `json:"chars_per_token"`
}

type DisplayConfig struct {
	ShowWorkspace bool `json:"show_workspace"`
	ReverseOrder  bool `json:"reverse_order"`
	MaxLogsView   int  `json:"max_logs_view"`
	Quiet         bool `json:"quiet"`
}

type StorageConfig struct {
	LogFilePrefix string `json:"log_file_prefix"`
	Rotation      string `json:"rotation"` // "monthly", "none"
}

type ModelPrice struct {
	InputPer1K  float64 `json:"input_per_1k"`
	OutputPer1K float64 `json:"output_per_1k"`
}

type PricingConfig struct {
	Currency string                `json:"currency"`
	Models   map[string]ModelPrice `json:"models"`
}

type Config struct {
	ProjectName     string                `json:"project_name"`
	DefaultModel    string                `json:"default_model"`
	Timezone        string                `json:"timezone"`
	AutoRun         bool                  `json:"auto_run"`
	TokenEstimation TokenEstimationConfig `json:"token_estimation"`
	Display         DisplayConfig         `json:"display"`
	Storage         StorageConfig         `json:"storage"`
	Pricing         PricingConfig         `json:"pricing"`
}

type LogEntry struct {
	Timestamp   string   `json:"timestamp"`
	Category    string   `json:"category"`
	Message     string   `json:"message"`
	Question    string   `json:"question"`
	Answer      string   `json:"answer"`
	Workspace   string   `json:"workspace"`
	Model       string   `json:"model"`
	TokensIn    int      `json:"tokens_in"`
	TokensOut   int      `json:"tokens_out"`
	IsEstimated bool     `json:"is_estimated"`
	Duration    float64  `json:"duration,omitempty"`   // Time taken in seconds
	SessionID   string   `json:"session_id,omitempty"` // Conversation thread identifier
	Cost        float64  `json:"cost,omitempty"`       // Calculated cost at the time of logging
	Status      string   `json:"status,omitempty"`     // Success or error status
	ToolsUsed   []string `json:"tools_used,omitempty"` // Tools invoked by the AI
	Tags        []string `json:"tags,omitempty"`
}

type DateFilter struct {
	Exact string
	From  *time.Time
	To    *time.Time
}

type FilterOptions struct {
	DateFilter DateFilter
	Keyword    string
	Model      string
	Category   string
	Tag        string
}

const (
	Version     = "0.1.0"
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorGray   = "\033[37m"
	ColorBold   = "\033[1m"
)

var config Config
var appDir string
var openRouterPricingCache map[string]ModelPrice
var openRouterPricingLoaded bool

type OpenRouterModelPricing struct {
	Prompt     string `json:"prompt"`
	Completion string `json:"completion"`
}

type OpenRouterModelEntry struct {
	ID      string                 `json:"id"`
	Pricing OpenRouterModelPricing `json:"pricing"`
}

type OpenRouterModelsResponse struct {
	Data []OpenRouterModelEntry `json:"data"`
}

const splashBanner = `
     ___                       _   _____              _   
    / _ \                     | | |_   _|            | |  
   / /_\ \ __ _  ___ _ __  ___| |_  | | _ __ __ _  ___| | __
   |  _  |/ _` + "`" + ` |/ _ \ '_ \/ __| __| | || '__/ _` + "`" + ` |/ __| |/ /
   | | | | (_| |  __/ | | \__ \ |_  | || | | (_| | (__|   < 
   \_| |_/\__, |\___|_| |_|___/\__| \_/_|  \__,_|\___|_|\_\
          __/ |                                           
         |___/                                            `

func defaultConfig() Config {
	return Config{
		ProjectName:  "AgentTrack Activity Tracker",
		DefaultModel: "gemini-1.5-flash",
		Timezone:     "Asia/Bangkok",
		AutoRun:      false,
		TokenEstimation: TokenEstimationConfig{
			Enabled:       true,
			CharsPerToken: 3.5,
		},
		Display: DisplayConfig{
			ShowWorkspace: true,
			ReverseOrder:  true,
			MaxLogsView:   50,
		},
		Storage: StorageConfig{
			LogFilePrefix: "atrack_logs",
			Rotation:      "monthly",
		},
		Pricing: PricingConfig{
			Currency: "USD",
			Models:   map[string]ModelPrice{},
		},
	}
}

func getAppDir() string {
	if envDir := os.Getenv("ATRACK_HOME"); envDir != "" {
		return envDir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	dir := filepath.Join(home, ".atrack")
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		os.MkdirAll(dir, 0755)
	}
	return dir
}

func loadConfig() {
	appDir = getAppDir()
	config = defaultConfig()

	configPath := filepath.Join(appDir, "config.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		data, _ := json.MarshalIndent(config, "", "  ")
		os.WriteFile(configPath, data, 0644)
	} else {
		data, err := os.ReadFile(configPath)
		if err == nil {
			json.Unmarshal(data, &config)
		}
	}
}

func getLogPath(t time.Time) string {
	if config.Storage.Rotation == "monthly" {
		return filepath.Join(appDir, fmt.Sprintf("%s_%s.jsonl", config.Storage.LogFilePrefix, t.Format("2006_01")))
	}
	return filepath.Join(appDir, config.Storage.LogFilePrefix+".jsonl")
}

func getAllLogFiles() []string {
	files, _ := filepath.Glob(filepath.Join(appDir, config.Storage.LogFilePrefix+"*.jsonl"))
	sort.Strings(files)
	return files
}

func getLogsFromAllFiles() []LogEntry {
	var allLogs []LogEntry
	files := getAllLogFiles()
	for _, file := range files {
		logs := readLogsFromFile(file)
		allLogs = append(allLogs, logs...)
	}
	return allLogs
}

func readLogsFromFile(path string) []LogEntry {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()

	var logs []LogEntry
	scanner := bufio.NewScanner(file)
	const maxLogLineSize = 10 * 1024 * 1024
	scanner.Buffer(make([]byte, 0, 64*1024), maxLogLineSize)
	for scanner.Scan() {
		var entry LogEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err == nil {
			logs = append(logs, entry)
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Printf("Warning: could not read %s: %v\n", path, err)
	}
	return logs
}

func appendLogToFile(path string, entry LogEntry) error {
	fileLock := flock.New(path + ".lock")
	err := fileLock.Lock()
	if err != nil {
		return err
	}
	defer fileLock.Unlock()

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	if _, err := f.Write(append(data, '\n')); err != nil {
		return err
	}
	return nil
}

func saveLogsToFile(path string, logs []LogEntry) {
	fileLock := flock.New(path + ".lock")
	if err := fileLock.Lock(); err != nil {
		fmt.Printf("Warning: could not lock %s: %v\n", path, err)
		return
	}
	defer fileLock.Unlock()

	f, err := os.Create(path)
	if err != nil {
		return
	}
	defer f.Close()

	for _, entry := range logs {
		data, _ := json.Marshal(entry)
		f.Write(append(data, '\n'))
	}
}

func shortLogTime(timestamp string) string {
	if len(timestamp) >= 16 {
		return timestamp[11:16]
	}
	if timestamp == "" {
		return "n/a"
	}
	return timestamp
}

func shortCategory(category string) string {
	if category == "" {
		return "n/a"
	}
	if len(category) <= 4 {
		return category
	}
	return category[:4]
}

func getConfigLocation() *time.Location {
	loc, _ := time.LoadLocation(config.Timezone)
	if loc == nil {
		return time.Local
	}
	return loc
}

func normalizeTags(raw string) []string {
	if raw == "" {
		return nil
	}

	seen := make(map[string]bool)
	var tags []string
	for _, part := range strings.Split(raw, ",") {
		tag := strings.TrimSpace(part)
		if tag == "" {
			continue
		}
		key := strings.ToLower(tag)
		if seen[key] {
			continue
		}
		seen[key] = true
		tags = append(tags, tag)
	}
	return tags
}

func logModel(log LogEntry) string {
	if log.Model == "" {
		return config.DefaultModel
	}
	return log.Model
}

func logCategory(log LogEntry) string {
	if log.Category == "" {
		return "General"
	}
	return log.Category
}

func parseDateOnly(value string) (time.Time, error) {
	return time.ParseInLocation("2006-01-02", value, getConfigLocation())
}

func parseTimestamp(value string) (time.Time, error) {
	return time.ParseInLocation("2006-01-02 15:04:05", value, getConfigLocation())
}

func parseDateFilters(args []string) (DateFilter, []string, error) {
	var filter DateFilter
	var remaining []string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--from":
			if i+1 >= len(args) {
				return filter, nil, fmt.Errorf("missing value for --from")
			}
			parsed, err := parseDateOnly(args[i+1])
			if err != nil {
				return filter, nil, fmt.Errorf("invalid --from date: %v", err)
			}
			filter.From = &parsed
			i++
		case "--to":
			if i+1 >= len(args) {
				return filter, nil, fmt.Errorf("missing value for --to")
			}
			parsed, err := parseDateOnly(args[i+1])
			if err != nil {
				return filter, nil, fmt.Errorf("invalid --to date: %v", err)
			}
			filter.To = &parsed
			i++
		default:
			remaining = append(remaining, args[i])
		}
	}

	return filter, remaining, nil
}

func logHasTag(log LogEntry, tag string) bool {
	if tag == "" {
		return true
	}
	tag = strings.ToLower(tag)
	for _, existing := range log.Tags {
		if strings.Contains(strings.ToLower(existing), tag) {
			return true
		}
	}
	return false
}

func matchesKeyword(log LogEntry, keyword string) bool {
	keyword = strings.ToLower(keyword)
	if keyword == "" {
		return true
	}

	fields := []string{
		log.Message,
		log.Question,
		log.Answer,
		logCategory(log),
		logModel(log),
	}
	for _, tag := range log.Tags {
		fields = append(fields, tag)
	}

	for _, field := range fields {
		if strings.Contains(strings.ToLower(field), keyword) {
			return true
		}
	}
	return false
}

func matchesDateFilter(log LogEntry, filter DateFilter) bool {
	if filter.Exact != "" && !strings.HasPrefix(log.Timestamp, filter.Exact) {
		return false
	}

	if filter.From == nil && filter.To == nil {
		return true
	}

	logTime, err := parseTimestamp(log.Timestamp)
	if err != nil {
		return false
	}

	if filter.From != nil && logTime.Before(*filter.From) {
		return false
	}
	if filter.To != nil {
		endOfDay := filter.To.Add(24*time.Hour - time.Nanosecond)
		if logTime.After(endOfDay) {
			return false
		}
	}
	return true
}

func filterLogs(logs []LogEntry, filter FilterOptions) []LogEntry {
	var filtered []LogEntry
	for _, log := range logs {
		if !matchesDateFilter(log, filter.DateFilter) {
			continue
		}
		if filter.Keyword != "" && !matchesKeyword(log, filter.Keyword) {
			continue
		}
		if filter.Model != "" && !strings.Contains(strings.ToLower(logModel(log)), strings.ToLower(filter.Model)) {
			continue
		}
		if filter.Category != "" && !strings.Contains(strings.ToLower(logCategory(log)), strings.ToLower(filter.Category)) {
			continue
		}
		if filter.Tag != "" && !logHasTag(log, filter.Tag) {
			continue
		}
		filtered = append(filtered, log)
	}
	return filtered
}

func renderUsageSummary(itemLabel string, counts map[string]int) {
	type summaryRow struct {
		Name  string
		Count int
	}

	var rows []summaryRow
	for name, count := range counts {
		rows = append(rows, summaryRow{Name: name, Count: count})
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Count == rows[j].Count {
			return strings.ToLower(rows[i].Name) < strings.ToLower(rows[j].Name)
		}
		return rows[i].Count > rows[j].Count
	})

	fmt.Printf("%s%-4s | %-30s | %-5s%s\n", ColorBold, "ID", itemLabel, "Count", ColorReset)
	fmt.Println(ColorGray + strings.Repeat("=", 50) + ColorReset)
	for i, row := range rows {
		fmt.Printf(ColorBlue+"#%-3d"+ColorReset+" | "+ColorGreen+"%-30s"+ColorReset+" | "+ColorYellow+"%-5d"+ColorReset+"\n", i, row.Name, row.Count)
	}
}

func parseOpenRouterRate(raw string) float64 {
	if raw == "" {
		return 0
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0
	}
	return value * 1000
}

func registerOpenRouterAlias(prices map[string]ModelPrice, id string, price ModelPrice) {
	key := strings.ToLower(strings.TrimSpace(id))
	if key == "" {
		return
	}
	prices[key] = price
	if trimmed := strings.TrimPrefix(key, "~"); trimmed != key {
		prices[trimmed] = price
		key = trimmed
	}
	if idx := strings.LastIndex(key, "/"); idx >= 0 && idx+1 < len(key) {
		prices[key[idx+1:]] = price
	}
}

func fetchOpenRouterPricingData() (map[string]ModelPrice, map[string]ModelPrice, error) {
	url := os.Getenv("ATRACK_OPENROUTER_MODELS_URL")
	if url == "" {
		url = "https://openrouter.ai/api/v1/models"
	}
	if strings.EqualFold(url, "off") {
		return nil, nil, fmt.Errorf("OpenRouter pricing sync is disabled via ATRACK_OPENROUTER_MODELS_URL=off")
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("unexpected OpenRouter status: %s", resp.Status)
	}

	var payload OpenRouterModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, nil, err
	}

	canonical := make(map[string]ModelPrice)
	aliases := make(map[string]ModelPrice)
	for _, entry := range payload.Data {
		id := strings.ToLower(strings.TrimSpace(entry.ID))
		if id == "" {
			continue
		}
		price := ModelPrice{
			InputPer1K:  parseOpenRouterRate(entry.Pricing.Prompt),
			OutputPer1K: parseOpenRouterRate(entry.Pricing.Completion),
		}
		canonical[id] = price
		registerOpenRouterAlias(aliases, entry.ID, price)
	}

	return canonical, aliases, nil
}

func loadOpenRouterPricing() map[string]ModelPrice {
	if openRouterPricingLoaded {
		return openRouterPricingCache
	}

	openRouterPricingLoaded = true
	openRouterPricingCache = make(map[string]ModelPrice)

	_, aliases, err := fetchOpenRouterPricingData()
	if err != nil {
		return openRouterPricingCache
	}
	openRouterPricingCache = aliases

	return openRouterPricingCache
}

func sameModelPrice(a, b ModelPrice) bool {
	return a.InputPer1K == b.InputPer1K && a.OutputPer1K == b.OutputPer1K
}

func collectModelNamesForPricingSync() []string {
	seen := make(map[string]bool)
	for name := range config.Pricing.Models {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			continue
		}
		seen[trimmed] = true
	}
	for _, log := range getLogsFromAllFiles() {
		model := strings.TrimSpace(logModel(log))
		if model == "" {
			continue
		}
		seen[model] = true
	}

	var models []string
	for model := range seen {
		models = append(models, model)
	}
	sort.Slice(models, func(i, j int) bool {
		return strings.ToLower(models[i]) < strings.ToLower(models[j])
	})
	return models
}

func syncOpenRouterPricing(args []string) {
	loadConfig()

	canonical, aliases, err := fetchOpenRouterPricingData()
	if err != nil {
		fmt.Printf("Error fetching OpenRouter pricing: %v\n", err)
		return
	}
	if config.Pricing.Models == nil {
		config.Pricing.Models = make(map[string]ModelPrice)
	}

	var targets []string
	if len(args) == 0 {
		targets = collectModelNamesForPricingSync()
		if len(targets) == 0 {
			fmt.Println("No models found in logs/config to sync. Use `atrack pricing sync all` or specify model names.")
			return
		}
	} else if len(args) == 1 && strings.EqualFold(args[0], "all") {
		for model := range canonical {
			targets = append(targets, model)
		}
		sort.Slice(targets, func(i, j int) bool {
			return strings.ToLower(targets[i]) < strings.ToLower(targets[j])
		})
	} else {
		targets = args
	}

	var added, updated, unchanged int
	var changedLines []string
	var missing []string

	for _, target := range targets {
		name := strings.TrimSpace(target)
		if name == "" {
			continue
		}
		price, ok := aliases[strings.ToLower(name)]
		if !ok {
			price, ok = canonical[strings.ToLower(name)]
		}
		if !ok {
			missing = append(missing, name)
			continue
		}

		current, exists := config.Pricing.Models[name]
		if exists && sameModelPrice(current, price) {
			unchanged++
			continue
		}

		config.Pricing.Models[name] = price
		if exists {
			updated++
		} else {
			added++
		}
		changedLines = append(changedLines, fmt.Sprintf("  %s -> input=%s output=%s", name, strconv.FormatFloat(price.InputPer1K, 'f', -1, 64), strconv.FormatFloat(price.OutputPer1K, 'f', -1, 64)))
	}

	if added == 0 && updated == 0 {
		fmt.Printf("OpenRouter pricing checked. No changes needed. Unchanged: %d", unchanged)
		if len(missing) > 0 {
			fmt.Printf(" | Missing: %d", len(missing))
		}
		fmt.Println()
		if len(missing) > 0 {
			fmt.Printf("Missing models: %s\n", strings.Join(missing, ", "))
		}
		return
	}

	if err := saveConfig(); err != nil {
		fmt.Printf("Error saving config: %v\n", err)
		return
	}

	openRouterPricingLoaded = false
	openRouterPricingCache = nil

	fmt.Printf("OpenRouter pricing synced. Added: %d | Updated: %d | Unchanged: %d", added, updated, unchanged)
	if len(missing) > 0 {
		fmt.Printf(" | Missing: %d", len(missing))
	}
	fmt.Println()
	for _, line := range changedLines {
		fmt.Println(line)
	}
	if len(missing) > 0 {
		fmt.Printf("Missing models: %s\n", strings.Join(missing, ", "))
	}
}

func findModelPrice(model string) (ModelPrice, bool) {
	for name, price := range config.Pricing.Models {
		if strings.EqualFold(name, model) {
			return price, true
		}
	}
	if price, ok := loadOpenRouterPricing()[strings.ToLower(model)]; ok {
		return price, true
	}
	return ModelPrice{}, false
}

func calculateLogCost(log LogEntry) (float64, bool) {
	price, ok := findModelPrice(logModel(log))
	if !ok {
		return 0, false
	}

	inputCost := (float64(log.TokensIn) / 1000.0) * price.InputPer1K
	outputCost := (float64(log.TokensOut) / 1000.0) * price.OutputPer1K
	return inputCost + outputCost, true
}

func estimateTokens(text string) int {
	if text == "" || !config.TokenEstimation.Enabled {
		return 0
	}
	return int(math.Ceil(float64(len(text)) / config.TokenEstimation.CharsPerToken))
}

func addLog(entry LogEntry) {
	loadConfig()
	now := time.Now().In(getConfigLocation())
	entry.Timestamp = now.Format("2006-01-02 15:04:05")

	path := getLogPath(now)
	if entry.Category == "" {
		entry.Category = "General"
	}
	if entry.Workspace == "" {
		entry.Workspace, _ = os.Getwd()
	}
	if entry.Model == "" {
		entry.Model = config.DefaultModel
	}

	if err := appendLogToFile(path, entry); err != nil {
		fmt.Printf("❌ "+ColorRed+"Error adding log:"+ColorReset+" %v\n", err)
		return
	}

	if config.Display.Quiet {
		return
	}

	estStr := ""
	if entry.IsEstimated {
		estStr = " [" + ColorYellow + "Tokens Estimated" + ColorReset + "]"
	}
	fmt.Printf("✨ "+ColorGreen+"Log added:"+ColorReset+" ["+ColorCyan+"%s"+ColorReset+"] ("+ColorPurple+"%s"+ColorReset+")%s\n", entry.Timestamp, entry.Category, estStr)
}

func searchLogs(keyword string, dateFilter DateFilter) {
	loadConfig()
	logs := filterLogs(getLogsFromAllFiles(), FilterOptions{
		DateFilter: dateFilter,
		Keyword:    keyword,
	})
	if len(logs) == 0 {
		fmt.Printf("No logs found matching: %s\n", keyword)
		return
	}
	renderLogs(logs)
}

func searchLogsByModel(model string, dateFilter DateFilter) {
	loadConfig()
	logs := filterLogs(getLogsFromAllFiles(), FilterOptions{
		DateFilter: dateFilter,
		Model:      model,
	})
	if len(logs) == 0 {
		fmt.Printf("No logs found for model: %s\n", model)
		return
	}
	renderLogs(logs)
}

func searchLogsByTag(tag string, dateFilter DateFilter) {
	loadConfig()
	logs := filterLogs(getLogsFromAllFiles(), FilterOptions{
		DateFilter: dateFilter,
		Tag:        tag,
	})
	if len(logs) == 0 {
		fmt.Printf("No logs found for tag: %s\n", tag)
		return
	}
	renderLogs(logs)
}

func listLogsByModel(model string, dateFilter DateFilter) {
	loadConfig()
	logs := filterLogs(getLogsFromAllFiles(), FilterOptions{DateFilter: dateFilter})
	if model == "all" {
		if len(logs) == 0 {
			fmt.Println("No logs found.")
			return
		}
		renderModelUsage(logs)
		return
	}

	filtered := filterLogs(logs, FilterOptions{Model: model})
	if len(filtered) == 0 {
		fmt.Printf("No logs found for model: %s\n", model)
		return
	}
	renderLogs(filtered)
}

func listLogsByCategory(category string, dateFilter DateFilter) {
	loadConfig()
	logs := filterLogs(getLogsFromAllFiles(), FilterOptions{DateFilter: dateFilter})
	if strings.EqualFold(category, "all") {
		if len(logs) == 0 {
			fmt.Println("No logs found.")
			return
		}
		renderCategoryUsage(logs)
		return
	}

	filtered := filterLogs(logs, FilterOptions{Category: category})
	if len(filtered) == 0 {
		fmt.Printf("No logs found for category: %s\n", category)
		return
	}
	renderLogs(filtered)
}

func renderModelUsage(logs []LogEntry) {
	rows := collectModelStats(logs)
	fmt.Printf("%s%-4s | %-30s | %-5s | %-8s | %-12s%s\n", ColorBold, "ID", "Model", "Count", "Tokens", "Cost", ColorReset)
	fmt.Println(ColorGray + strings.Repeat("=", 74) + ColorReset)
	for i, row := range rows {
		cost := "n/a"
		if row.HasCost {
			cost = fmt.Sprintf("%s %.4f", config.Pricing.Currency, row.Cost)
		}
		fmt.Printf(ColorBlue+"#%-3d"+ColorReset+" | "+ColorGreen+"%-30s"+ColorReset+" | "+ColorYellow+"%-5d"+ColorReset+" | "+ColorCyan+"%-8d"+ColorReset+" | %-12s\n", i, row.Model, row.Logs, row.TokensIn+row.TokensOut, cost)
	}
}

func renderCategoryUsage(logs []LogEntry) {
	counts := make(map[string]int)
	for _, log := range logs {
		counts[logCategory(log)]++
	}
	renderUsageSummary("Category", counts)
}

func renderLogs(logs []LogEntry) {
	if config.Display.ReverseOrder {
		for i, j := 0, len(logs)-1; i < j; i, j = i+1, j-1 {
			logs[i], logs[j] = logs[j], logs[i]
		}
	}

	limit := config.Display.MaxLogsView
	if limit > len(logs) {
		limit = len(logs)
	}
	displayLogs := logs[:limit]

	fmt.Printf("%s%-5s | %-5s | %-8s | Metadata / Q&A%s\n", ColorBold, "ID", "Time", "Cat", ColorReset)
	fmt.Println(ColorGray + strings.Repeat("=", 100) + ColorReset)

	for i, log := range displayLogs {
		displayID := i
		if config.Display.ReverseOrder {
			displayID = len(logs) - 1 - i
		}
		category := logCategory(log)

		estIn, estOut := "", ""
		if log.IsEstimated {
			estIn = " (est)"
			estOut = " (est)"
		}

		metadata := fmt.Sprintf(ColorYellow+"[M: %s | T: %d%s/%d%s]"+ColorReset, logModel(log), log.TokensIn, estIn, log.TokensOut, estOut)
		if log.Duration > 0 {
			metadata += fmt.Sprintf(ColorCyan+" [%.2fs]"+ColorReset, log.Duration)
		}
		if log.Cost > 0 {
			metadata += fmt.Sprintf(ColorGreen+" [$: %.4f]"+ColorReset, log.Cost)
		}
		if log.SessionID != "" {
			sidShort := log.SessionID
			if len(sidShort) > 8 {
				sidShort = sidShort[:8]
			}
			metadata += fmt.Sprintf(ColorPurple+" [S: %s]"+ColorReset, sidShort)
		}
		if log.Status != "" && log.Status != "success" {
			metadata += fmt.Sprintf(ColorRed+" [! %s]"+ColorReset, log.Status)
		}
		
		tagsLine := ""
		if len(log.Tags) > 0 {
			tagsLine = fmt.Sprintf(ColorYellow+"🏷️  %s"+ColorReset, strings.Join(log.Tags, ", "))
		}
		if len(log.ToolsUsed) > 0 {
			toolsLine := fmt.Sprintf(ColorBlue+"🛠️  %s"+ColorReset, strings.Join(log.ToolsUsed, ", "))
			if tagsLine != "" {
				tagsLine += " " + toolsLine
			} else {
				tagsLine = toolsLine
			}
		}

		timeLabel := shortLogTime(log.Timestamp)
		categoryLabel := shortCategory(category)

		if category == "AutoLog" && log.Question != "" && log.Answer != "" {
			fmt.Printf(ColorBlue+"#%-4d"+ColorReset+" | "+ColorGreen+"%s"+ColorReset+" | "+ColorPurple+"%-8s"+ColorReset+" | %s\n", displayID, timeLabel, categoryLabel, metadata)
			if tagsLine != "" {
				fmt.Printf("%-5s | %-5s | %-8s | %s\n", "", "", "", tagsLine)
			}
			fmt.Printf("%-5s | %-5s | %-8s | 👤 "+ColorBold+"%s"+ColorReset+"\n", "", "", "", log.Question)
			fmt.Printf("%-5s | %-5s | %-8s | 🤖 "+ColorBold+"%s"+ColorReset+"\n", "", "", "", log.Answer)
		} else {
			fmt.Printf(ColorBlue+"#%-4d"+ColorReset+" | "+ColorGreen+"%s"+ColorReset+" | "+ColorPurple+"%-8s"+ColorReset+" | 📝 %s\n", displayID, timeLabel, categoryLabel, log.Message)
			fmt.Printf("%-5s | %-5s | %-8s | %s\n", "", "", "", metadata)
			if tagsLine != "" {
				fmt.Printf("%-5s | %-5s | %-8s | %s\n", "", "", "", tagsLine)
			}
		}
		fmt.Println(ColorGray + strings.Repeat("-", 100) + ColorReset)
	}
}

func showLastLogs(count int) {
	loadConfig()
	logs := getLogsFromAllFiles()
	if len(logs) == 0 {
		fmt.Println("No logs found.")
		return
	}
	if count > len(logs) {
		count = len(logs)
	}
	lastLogs := logs[len(logs)-count:]
	renderLogs(lastLogs)
}

func listLogs(dateFilter DateFilter) {
	loadConfig()
	logs := filterLogs(getLogsFromAllFiles(), FilterOptions{DateFilter: dateFilter})
	if len(logs) == 0 {
		if dateFilter.Exact != "" {
			fmt.Printf("No logs found for date: %s\n", dateFilter.Exact)
			return
		}
		if dateFilter.From != nil || dateFilter.To != nil {
			fmt.Println("No logs found for the requested date range.")
			return
		}
		fmt.Println("No logs found.")
		return
	}
	renderLogs(logs)
}

func sameTags(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !strings.EqualFold(strings.TrimSpace(a[i]), strings.TrimSpace(b[i])) {
			return false
		}
	}
	return true
}

func logEntriesMatch(a LogEntry, b LogEntry) bool {
	return a.Timestamp == b.Timestamp &&
		a.Category == b.Category &&
		a.Message == b.Message &&
		a.Question == b.Question &&
		a.Answer == b.Answer &&
		a.Workspace == b.Workspace &&
		strings.EqualFold(logModel(a), logModel(b)) &&
		a.TokensIn == b.TokensIn &&
		a.TokensOut == b.TokensOut &&
		a.IsEstimated == b.IsEstimated &&
		a.Duration == b.Duration &&
		a.SessionID == b.SessionID &&
		a.Cost == b.Cost &&
		a.Status == b.Status &&
		sameTags(a.ToolsUsed, b.ToolsUsed) &&
		sameTags(a.Tags, b.Tags)
}

func resolveLogIndex(index int) (string, []LogEntry, int, LogEntry, error) {
	loadConfig()
	logs := getLogsFromAllFiles()
	if index < 0 || index >= len(logs) {
		return "", nil, -1, LogEntry{}, fmt.Errorf("index %d out of range (0-%d)", index, len(logs)-1)
	}

	target := logs[index]
	t, err := parseTimestamp(target.Timestamp)
	if err != nil {
		return "", nil, -1, LogEntry{}, fmt.Errorf("error parsing timestamp: %v", err)
	}

	path := getLogPath(t)
	fileLogs := readLogsFromFile(path)

	for i, log := range fileLogs {
		if logEntriesMatch(log, target) {
			return path, fileLogs, i, target, nil
		}
	}

	return "", nil, -1, LogEntry{}, fmt.Errorf("could not find the exact log entry in the file")
}

func deleteLog(index int) {
	path, fileLogs, fileIndex, target, err := resolveLogIndex(index)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	newFileLogs := append(fileLogs[:fileIndex], fileLogs[fileIndex+1:]...)
	saveLogsToFile(path, newFileLogs)
	fmt.Printf("Log #%d [%s] deleted successfully.\n", index, target.Timestamp)
}

func editLog(index int, args []string) {
	if len(args) == 0 {
		fmt.Println("Error: Usage: atrack edit <index> [field] <value>")
		return
	}

	path, fileLogs, fileIndex, target, err := resolveLogIndex(index)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	field := ""
	value := ""
	if len(args) == 1 {
		if target.Category == "AutoLog" && target.Answer != "" {
			field = "answer"
		} else {
			field = "message"
		}
		value = args[0]
	} else {
		field = strings.ToLower(args[0])
		value = strings.Join(args[1:], " ")
	}

	updated := fileLogs[fileIndex]
	switch field {
	case "message":
		updated.Message = value
	case "question":
		updated.Question = value
	case "answer":
		updated.Answer = value
	case "category":
		updated.Category = value
	case "model":
		updated.Model = value
	case "tags":
		updated.Tags = normalizeTags(value)
	default:
		fmt.Printf("Error: Unknown edit field: %s\n", field)
		return
	}

	fileLogs[fileIndex] = updated
	saveLogsToFile(path, fileLogs)
	fmt.Printf("Log #%d updated: %s\n", index, field)
}

func clearLogs() {
	loadConfig()
	files := getAllLogFiles()
	for _, file := range files {
		os.WriteFile(file, []byte(""), 0644)
	}
	fmt.Println("All log files cleared.")

	// Also prime watchers to ignore existing history
	PrimeWatchers()
}

func hasConfirmFlag(args []string) bool {
	for _, arg := range args {
		if arg == "--yes" || arg == "-y" {
			return true
		}
	}
	return false
}

func confirmAction(prompt string, skipConfirm bool) bool {
	if skipConfirm {
		return true
	}

	fmt.Print(prompt)
	var input string
	if _, err := fmt.Scanln(&input); err != nil {
		return false
	}
	value := strings.ToLower(strings.TrimSpace(input))
	return value == "y" || value == "yes"
}

func resetAppData(skipConfirm bool) {
	loadConfig()
	if !confirmAction(fmt.Sprintf("This will delete all logs and reset config in %s. Continue? [y/N]: ", appDir), skipConfirm) {
		fmt.Println("Reset cancelled.")
		return
	}

	files := getAllLogFiles()
	for _, file := range files {
		if err := os.Remove(file); err != nil && !os.IsNotExist(err) {
			fmt.Printf("Warning: could not remove %s: %v\n", file, err)
		}
	}

	config = defaultConfig()
	if err := saveConfig(); err != nil {
		fmt.Printf("Error saving config: %v\n", err)
		return
	}

	openRouterPricingCache = nil
	openRouterPricingLoaded = false
	fmt.Println("AgentTrack has been reset (logs deleted, config restored to defaults).")
}

func removeManagedBlocks(content string, block string) (string, bool) {
	updated := strings.ReplaceAll(content, "\n"+block+"\n", "\n")
	updated = strings.ReplaceAll(updated, "\n"+block, "")
	updated = strings.ReplaceAll(updated, block+"\n", "")
	updated = strings.ReplaceAll(updated, block, "")
	return updated, updated != content
}

func installHooks() {
	home, _ := os.UserHomeDir()
	if home == "" {
		return
	}

	// ---------------------------------------------------------------------------
	// 1. Zsh / Bash Hooks (macOS/Linux)
	// ---------------------------------------------------------------------------
	profiles := []string{
		filepath.Join(home, ".zshrc"),
		filepath.Join(home, ".bashrc"),
		filepath.Join(home, ".bash_profile"),
		filepath.Join(home, ".profile"),
	}

	zshAutoInitBlock := "\n# AgentTrack Auto-Init Hook (Zsh)\natrack_auto_init() {\n  if [ -w \".\" ] && [ ! -f \".cursorrules\" ]; then\n      atrack init >/dev/null 2>&1\n  fi\n}\nautoload -U add-zsh-hook 2>/dev/null\nadd-zsh-hook chpwd atrack_auto_init 2>/dev/null\natrack_auto_init"
	bashAutoInitBlock := "\n# AgentTrack Auto-Init Hook (Bash)\natrack_auto_init() {\n  if [ -w \".\" ] && [ ! -f \".cursorrules\" ]; then\n      atrack init >/dev/null 2>&1\n  fi\n}\nif [[ ! \"$PROMPT_COMMAND\" == *\"atrack_auto_init\"* ]]; then\n    export PROMPT_COMMAND=\"atrack_auto_init; $PROMPT_COMMAND\"\nfi\natrack_auto_init"

	for _, profile := range profiles {
		if _, err := os.Stat(profile); err != nil {
			continue
		}
		data, _ := os.ReadFile(profile)
		content := string(data)
		block := bashAutoInitBlock
		if strings.HasSuffix(profile, ".zshrc") {
			block = zshAutoInitBlock
		}

		if !strings.Contains(content, "atrack_auto_init") {
			f, _ := os.OpenFile(profile, os.O_APPEND|os.O_WRONLY, 0644)
			if f != nil {
				f.WriteString(block)
				f.Close()
				fmt.Printf("✅ Added Auto-Init hook to %s\n", profile)
			}
		}
	}

	// ---------------------------------------------------------------------------
	// 2. PowerShell Hook (Windows)
	// ---------------------------------------------------------------------------
	if runtime.GOOS == "windows" {
		psHook := `
# AgentTrack Auto-Init Hook (PowerShell)
function atrack_auto_init {
    if (Test-Path -Path . -PathType Container) {
        if (-not (Test-Path -Path ".cursorrules") -and (New-Object System.IO.DirectoryInfo ".").Attributes.HasFlag([System.IO.FileAttributes]::ReadOnly) -eq $false) {
            atrack init > $null 2>&1
        }
    }
}
# Hook into prompt to check on every directory change
if (-not (Get-Command atrack_auto_init -ErrorAction SilentlyContinue)) {
    $old_prompt = $function:prompt
    function prompt {
        atrack_auto_init
        &$old_prompt
    }
}
`
		// Target both PowerShell 5.1 and PowerShell 7+ profile paths
		psProfiles := []string{
			filepath.Join(home, "Documents", "WindowsPowerShell", "Microsoft.PowerShell_profile.ps1"),
			filepath.Join(home, "Documents", "PowerShell", "Microsoft.PowerShell_profile.ps1"),
		}

		for _, profilePath := range psProfiles {
			os.MkdirAll(filepath.Dir(profilePath), 0755)
			data, _ := os.ReadFile(profilePath)
			if !strings.Contains(string(data), "atrack_auto_init") {
				f, _ := os.OpenFile(profilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
				if f != nil {
					f.WriteString(psHook)
					f.Close()
					fmt.Printf("✅ Added Auto-Init hook to PowerShell profile: %s\n", profilePath)
				}
			}
		}
	}
}

func removeInstallHooks() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Warning: could not resolve home directory: %v\n", err)
		return
	}

	if err := uninstallAutoStartService(); err != nil {
		fmt.Printf("Warning: could not remove auto-start service: %v\n", err)
	}

	profiles := []string{
		filepath.Join(home, ".zshrc"),
		filepath.Join(home, ".bashrc"),
		filepath.Join(home, ".bash_profile"),
		filepath.Join(home, ".profile"),
	}

	// PowerShell Profile cleanup
	if runtime.GOOS == "windows" {
		out, err := exec.Command("powershell", "-NoProfile", "-Command", "$PROFILE").Output()
		if err == nil {
			profilePath := strings.TrimSpace(string(out))
			if profilePath != "" {
				profiles = append(profiles, profilePath)
			}
		}
	}

	copilotBlock := `# AgentTrack GitHub Copilot Wrapper
gh_copilot_wrapper() {
  if [ "$1" = "copilot" ] && [ "$2" = "suggest" -o "$2" = "explain" ]; then
    command gh "$@"
    atrack auto "$*" "Copilot query executed" "gh-copilot" 0 0 >/dev/null 2>&1
  else
    command gh "$@"
  fi
}
alias gh="gh_copilot_wrapper"`

	zshAutoInitBlock := `# AgentTrack Auto-Init Hook (Zsh)
atrack_auto_init() {
  if [ -w "." ] && [ ! -f ".cursorrules" ]; then
      atrack init >/dev/null 2>&1
  fi
}
autoload -U add-zsh-hook 2>/dev/null
add-zsh-hook chpwd atrack_auto_init 2>/dev/null
atrack_auto_init`

	bashAutoInitBlock := `# AgentTrack Auto-Init Hook (Bash)
atrack_auto_init() {
  if [ -w "." ] && [ ! -f ".cursorrules" ]; then
      atrack init >/dev/null 2>&1
  fi
}
if [[ ! "$PROMPT_COMMAND" == *"atrack_auto_init"* ]]; then
    export PROMPT_COMMAND="atrack_auto_init; $PROMPT_COMMAND"
fi
atrack_auto_init`

	psAutoInitBlock := `# AgentTrack Auto-Init Hook (PowerShell)
function atrack_auto_init {
    if (Test-Path -Path . -PathType Container) {
        if (-not (Test-Path -Path ".cursorrules") -and (New-Object System.IO.DirectoryInfo ".").Attributes.HasFlag([System.IO.FileAttributes]::ReadOnly) -eq $false) {
            atrack init > $null 2>&1
        }
    }
}
# Hook into prompt to check on every directory change
if (-not (Get-Command atrack_auto_init -ErrorAction SilentlyContinue)) {
    $old_prompt = $function:prompt
    function prompt {
        atrack_auto_init
        &$old_prompt
    }
}
`

	for _, profile := range profiles {
		data, err := os.ReadFile(profile)
		if err != nil {
			continue
		}

		content := string(data)
		updated, changedCopilot := removeManagedBlocks(content, copilotBlock)
		updated, changedZsh := removeManagedBlocks(updated, zshAutoInitBlock)
		updated, changedBash := removeManagedBlocks(updated, bashAutoInitBlock)
		updated, changedPs := removeManagedBlocks(updated, psAutoInitBlock)
		if !changedCopilot && !changedZsh && !changedBash && !changedPs {
			continue
		}

		if err := os.WriteFile(profile, []byte(updated), 0644); err != nil {
			fmt.Printf("Warning: could not update %s: %v\n", profile, err)
			continue
		}
		fmt.Printf("Removed AgentTrack hooks from %s\n", profile)
	}
}

func removeBinaryCandidates() {
	paths := map[string]bool{}
	if execPath, err := os.Executable(); err == nil {
		paths[execPath] = true
		if resolved, err := filepath.EvalSymlinks(execPath); err == nil {
			paths[resolved] = true
		}
	}

	home, err := os.UserHomeDir()
	if err == nil {
		paths[filepath.Join(home, "go", "bin", "atrack")] = true
		paths[filepath.Join(home, "go", "bin", "atrack.exe")] = true
	}

	for path := range paths {
		if path == "" {
			continue
		}
		base := strings.ToLower(filepath.Base(path))
		if base != "atrack" && base != "atrack.exe" {
			continue
		}
		if err := os.Remove(path); err == nil {
			fmt.Printf("Removed binary: %s\n", path)
		} else if !os.IsNotExist(err) {
			fmt.Printf("Warning: could not remove binary %s: %v\n", path, err)
		}
	}
}

func uninstallApp(skipConfirm bool) {
	loadConfig()
	if !confirmAction(fmt.Sprintf("This will remove AgentTrack data in %s and uninstall hooks. Continue? [y/N]: ", appDir), skipConfirm) {
		fmt.Println("Uninstall cancelled.")
		return
	}

	removeInstallHooks()
	if err := os.RemoveAll(appDir); err != nil {
		fmt.Printf("Warning: could not remove %s: %v\n", appDir, err)
	} else {
		fmt.Printf("Removed data directory: %s\n", appDir)
	}

	removeBinaryCandidates()
	fmt.Println("AgentTrack uninstall complete.")
}

func updateApp() {
	fmt.Println("🔄 Updating AgentTrack...")
	fmt.Println("Attempting to update via 'go install'...")
	
	cmd := exec.Command("go", "install", "github.com/alfaXphoori/AgentTrack/cmd/atrack@latest")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	if err := cmd.Run(); err != nil {
		fmt.Println("\n❌ Update failed or 'go' is not installed.")
		fmt.Println("\nPlease update manually based on your installation method:")
		fmt.Println("  " + ColorCyan + "Homebrew" + ColorReset + " (macOS/Linux): brew upgrade atrack")
		fmt.Println("  " + ColorCyan + "Scoop" + ColorReset + " (Windows):        scoop update atrack")
		fmt.Println("  " + ColorCyan + "Binary Release:" + ColorReset + "       Download the latest version from https://github.com/alfaXphoori/AgentTrack/releases")
	} else {
		fmt.Println("\n✅ " + ColorGreen + "AgentTrack updated successfully!" + ColorReset)
		fmt.Println("🔧 Refreshing background services...")
		if err := installAutoStartService(); err != nil {
			fmt.Printf("⚠️  Warning: could not refresh autostart: %v\n", err)
		} else {
			fmt.Println("✨ Background services refreshed and running.")
		}
	}
}

func watchLogs(interval time.Duration) {
	loadConfig()
	fmt.Printf(ColorCyan+"👀 Watching for new AgentTrack logs in real-time... (Interval: %v, Press Ctrl+C to stop)\n"+ColorReset, interval)
	fmt.Println(ColorGray + strings.Repeat("-", 110) + ColorReset)

	lastCount := len(getLogsFromAllFiles())

	for {
		time.Sleep(interval)
		logs := getLogsFromAllFiles()
		currentCount := len(logs)

		if currentCount > lastCount {
			for i := lastCount; i < currentCount; i++ {
				log := logs[i]
				category := logCategory(log)
				metadata := fmt.Sprintf(ColorYellow+"[Model: %s | Tokens: In=%d, Out=%d]"+ColorReset, logModel(log), log.TokensIn, log.TokensOut)
				now := time.Now().Format("15:04:05")

				if category == "AutoLog" && log.Question != "" {
					fmt.Printf("[%s] ✨ "+ColorGreen+"New Log:"+ColorReset+" ["+ColorCyan+"%s"+ColorReset+"] | %s\n", now, log.Timestamp, metadata)
					fmt.Printf("  👤 "+ColorBold+"Q: %s"+ColorReset+"\n", log.Question)
					fmt.Printf("  🤖 "+ColorBold+"A: %s"+ColorReset+"\n", log.Answer)
				} else {
					fmt.Printf("[%s] ✨ "+ColorGreen+"New Log:"+ColorReset+" ["+ColorCyan+"%s"+ColorReset+"] ("+ColorPurple+"%s"+ColorReset+") | 📝 %s\n", now, log.Timestamp, category, log.Message)
				}
				fmt.Println(ColorGray + strings.Repeat("-", 110) + ColorReset)
			}
			lastCount = currentCount
		} else if currentCount < lastCount {
			lastCount = currentCount
		}
	}
}

func initRules() {
	executable, err := serviceExecutablePath()
	if err != nil {
		executable = "atrack"
	}
	// Use forward slashes for cross-platform compatibility in rule files
	executable = filepath.ToSlash(executable)

	ruleContent := fmt.Sprintf(`# AgentTrack Log Trigger
On every chat turn, you MUST execute this shell command to log the activity:
"%s" auto "<user_q>" "<ai_a_summary>" "<model>" 0 0
- Q: Short prompt summary.
- A: Short answer summary (max 50 chars).
- Do not add conversational filler.
`, executable)
	files := []string{
		".cursorrules",
		".clinerules",
		".roorules",
		".windsurfrules",
		"CLAUDE.md",
		".aider.rules.md",
		"QWEN.md",
	}

	createdCount := 0
	for _, file := range files {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			err := os.WriteFile(file, []byte(ruleContent), 0644)
			if err == nil {
				fmt.Printf("✨ Created %s\n", file)
				createdCount++
			} else {
				fmt.Printf("❌ Error creating %s: %v\n", file, err)
			}
		} else {
			fmt.Printf("⚠️  Skipped %s (already exists)\n", file)
		}
	}

	if createdCount > 0 {
		fmt.Println("\n✅ AgentTrack rules initialized successfully!")
		fmt.Println("Agents like Cursor, Cline, Roo, Windsurf, Claude Code, and Aider will now auto-log to AgentTrack in this project.")
	} else {
		fmt.Println("\nNo new rule files were created.")
	}
}

func Run() {
	loadConfig()

	if len(os.Args) < 2 {
		printUsage()
		return
	}

	command := os.Args[1]
	switch command {
	case "log":
		if len(os.Args) < 3 {
			fmt.Println("Error: Please provide a message to log.")
			return
		}
		message := os.Args[2]
		category := "General"
		var tags []string
		for i := 3; i < len(os.Args)-1; i++ {
			if os.Args[i] == "-c" || os.Args[i] == "--category" {
				category = os.Args[i+1]
			}
			if os.Args[i] == "-t" || os.Args[i] == "--tags" {
				tags = normalizeTags(os.Args[i+1])
			}
		}
		addLog(LogEntry{Message: message, Category: category, Tags: tags})

	case "auto":
		question, answer, model := "", "", config.DefaultModel
		if len(os.Args) > 2 {
			question = os.Args[2]
		}
		if len(os.Args) > 3 {
			answer = os.Args[3]
		}
		if len(os.Args) > 4 && os.Args[4] != "" {
			model = os.Args[4]
		}

		tIn, tOut := 0, 0
		duration := 0.0
		sessionID := ""
		status := "success"
		var toolsUsed []string

		if len(os.Args) > 5 {
			tIn, _ = strconv.Atoi(os.Args[5])
		}
		if len(os.Args) > 6 {
			tOut, _ = strconv.Atoi(os.Args[6])
		}
		if len(os.Args) > 7 {
			duration, _ = strconv.ParseFloat(os.Args[7], 64)
		}
		if len(os.Args) > 8 {
			sessionID = os.Args[8]
		}
		if len(os.Args) > 9 {
			status = os.Args[9]
		}
		if len(os.Args) > 10 {
			toolsUsed = strings.Split(os.Args[10], ",")
		}
		var tags []string
		if len(os.Args) > 11 {
			tags = normalizeTags(os.Args[11])
		}

		isEst := false
		if tIn == 0 && tOut == 0 {
			tIn = estimateTokens(question)
			tOut = estimateTokens(answer)
			isEst = true
		}

		entry := LogEntry{
			Category:    "AutoLog",
			Question:    question,
			Answer:      answer,
			Model:       model,
			TokensIn:    tIn,
			TokensOut:   tOut,
			IsEstimated: isEst,
			Duration:    duration,
			SessionID:   sessionID,
			Status:      status,
			ToolsUsed:   toolsUsed,
			Tags:        tags,
		}

		// Calculate cost if pricing is available
		loadConfig()
		if cost, ok := calculateLogCost(entry); ok {
			entry.Cost = cost
		}

		addLog(entry)

	case "list":
		dateFilter, remaining, err := parseDateFilters(os.Args[2:])
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}

		if len(remaining) == 0 {
			listLogs(dateFilter)
			return
		}

		switch strings.ToLower(remaining[0]) {
		case "last":
			count := 1
			if len(remaining) > 1 {
				if c, err := strconv.Atoi(remaining[1]); err == nil {
					count = c
				}
			}
			showLastLogs(count)
			return
		case "model":
			if len(remaining) < 2 {
				fmt.Println("Error: Usage: atrack list model <model> [--from YYYY-MM-DD] [--to YYYY-MM-DD]")
				return
			}
			listLogsByModel(remaining[1], dateFilter)
		case "category":
			if len(remaining) < 2 {
				fmt.Println("Error: Usage: atrack list category <category> [--from YYYY-MM-DD] [--to YYYY-MM-DD]")
				return
			}
			listLogsByCategory(remaining[1], dateFilter)
		default:
			if len(remaining) > 1 {
				fmt.Println("Error: Usage: atrack list [date] [--from YYYY-MM-DD] [--to YYYY-MM-DD]")
				return
			}
			dateFilter.Exact = remaining[0]
			listLogs(dateFilter)
		}

	case "delete":
		if len(os.Args) < 3 {
			fmt.Println("Error: Please provide a log index to delete.")
			return
		}
		idx, err := strconv.Atoi(os.Args[2])
		if err == nil {
			deleteLog(idx)
		} else {
			fmt.Printf("Error: Invalid index: %v\n", err)
		}

	case "edit":
		if len(os.Args) < 4 {
			fmt.Println("Error: Usage: atrack edit <index> [field] <value>")
			return
		}
		idx, err := strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Printf("Error: Invalid index: %v\n", err)
			return
		}
		editLog(idx, os.Args[3:])

	case "search":
		dateFilter, remaining, err := parseDateFilters(os.Args[2:])
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		if len(remaining) == 0 {
			fmt.Println("Error: Please provide a keyword to search.")
			return
		}
		switch strings.ToLower(remaining[0]) {
		case "model":
			if len(remaining) < 2 {
				fmt.Println("Error: Usage: atrack search model <model> [--from YYYY-MM-DD] [--to YYYY-MM-DD]")
				return
			}
			searchLogsByModel(strings.Join(remaining[1:], " "), dateFilter)
		case "tag":
			if len(remaining) < 2 {
				fmt.Println("Error: Usage: atrack search tag <tag> [--from YYYY-MM-DD] [--to YYYY-MM-DD]")
				return
			}
			searchLogsByTag(strings.Join(remaining[1:], " "), dateFilter)
		default:
			searchLogs(strings.Join(remaining, " "), dateFilter)
		}

	case "clear":
		clearLogs()

	case "reset":
		resetAppData(hasConfirmFlag(os.Args[2:]))

	case "uninstall":
		uninstallApp(hasConfirmFlag(os.Args[2:]))

	case "update":
		updateApp()

	case "autostart":
		handleAutoStartCommand(os.Args[2:])

	case "summary":
		period := "today"
		if len(os.Args) > 2 {
			period = strings.ToLower(os.Args[2])
		}
		showSummary(period)

	case "tag":
		if len(os.Args) > 2 && strings.ToLower(os.Args[2]) == "list" {
			showTagList()
		} else {
			fmt.Println("Usage: atrack tag list")
		}

	case "watch":
		interval := 1 * time.Second
		if len(os.Args) > 2 {
			dur, err := time.ParseDuration(os.Args[2])
			if err == nil {
				interval = dur
			}
		}
		watchLogs(interval)

	case "init":
		initRules()

	case "prime":
		PrimeWatchers()

	case "internal-watch-copilot":
		watchCopilot()

	case "internal-watch-gemini":
		watchGemini()

	case "internal-detect-gemini":
		detectGeminiModel()

	case "dashboard":
		runDashboard()

	case "pricing":
		if len(os.Args) < 3 || !strings.EqualFold(os.Args[2], "sync") {
			fmt.Println("Usage: atrack pricing sync [all|model ...]")
			return
		}
		syncOpenRouterPricing(os.Args[3:])

	case "info":
		fmt.Printf("AgentTrack Global CLI\n")
		fmt.Printf("Version:       %s\n", Version)
		fmt.Printf("App Directory: %s\n", appDir)
		fmt.Printf("Config File:   %s\n", filepath.Join(appDir, "config.json"))
		fmt.Printf("Current Log:   %s\n", getLogPath(time.Now()))
		fmt.Printf("Total Files:   %d\n", len(getAllLogFiles()))

	case "version":
		fmt.Printf("AgentTrack version %s\n", Version)

	case "stats":
		if len(os.Args) > 2 {
			switch strings.ToLower(os.Args[2]) {
			case "model":
				showModelStats()
			case "cost":
				showCostStats()
			case "today":
				showStatsToday()
			default:
				fmt.Printf("Error: Unknown stats command: %s\n", os.Args[2])
			}
		} else {
			showStats()
		}

	case "export":
		format := "md"
		if len(os.Args) > 2 {
			format = os.Args[2]
		}
		exportLogs(format)

	case "config":
		if len(os.Args) == 2 || os.Args[2] == "show" {
			showConfig()
			return
		}
		switch os.Args[2] {
		case "set":
			if len(os.Args) < 5 {
				fmt.Println("Error: Usage: atrack config set <key> <value>")
				return
			}
			updateConfig(os.Args[3], os.Args[4:])
		case "get":
			if len(os.Args) < 4 {
				fmt.Println("Error: Usage: atrack config get <key>")
				return
			}
			showConfigValue(os.Args[3])
		case "reset":
			resetConfig()
		case "help":
			showConfigHelp()
		default:
			fmt.Printf("Error: Unknown config command: %s\n", os.Args[2])
			showConfigHelp()
		}

	case "help", "-h", "--help":
		printFullUsage()

	default:
		printUsage()
	}
}

func configPath() string {
	return filepath.Join(appDir, "config.json")
}

func saveConfig() error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath(), data, 0644)
}

func showConfig() {
	loadConfig()
	data, _ := json.MarshalIndent(config, "", "  ")
	fmt.Printf("Current Configuration (%s):\n", configPath())
	fmt.Println(string(data))
	fmt.Println()
	showConfigHelp()
}

func showConfigHelp() {
	fmt.Println("Config commands:")
	fmt.Println("  atrack config show")
	fmt.Println("      Show the full config file")
	fmt.Println("  atrack config get <key>")
	fmt.Println("      Show one config value")
	fmt.Println("  atrack config set <key> <value>")
	fmt.Println("      Update one config value")
	fmt.Println("  atrack config reset")
	fmt.Println("      Reset config to defaults")
	fmt.Println()
	fmt.Println("Available keys:")
	fmt.Println("  default_model")
	fmt.Println("  timezone")
	fmt.Println("  auto_run (true/false)")
	fmt.Println("  token_estimation.enabled")
	fmt.Println("  token_estimation.chars_per_token")
	fmt.Println("  display.show_workspace")
	fmt.Println("  display.reverse_order")
	fmt.Println("  display.max_logs_view")
	fmt.Println("  display.quiet (true/false)")
	fmt.Println("  storage.log_file_prefix")
	fmt.Println("  storage.rotation")
	fmt.Println("  pricing.currency")
	fmt.Println("  pricing.<model>.input_per_1k")
	fmt.Println("  pricing.<model>.output_per_1k")
}

func parseBool(value string) (bool, error) {
	switch strings.ToLower(value) {
	case "true", "1", "yes", "on":
		return true, nil
	case "false", "0", "no", "off":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean value: %s", value)
	}
}

func showConfigValue(key string) {
	loadConfig()
	value, err := getConfigValue(key)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("%s = %s\n", key, value)
}

func parsePricingKey(key string) (string, string, bool) {
	if strings.HasPrefix(strings.ToLower(key), "pricing.") && strings.HasSuffix(strings.ToLower(key), ".input_per_1k") {
		model := key[len("pricing.") : len(key)-len(".input_per_1k")]
		if model != "" {
			return model, "input_per_1k", true
		}
	}
	if strings.HasPrefix(strings.ToLower(key), "pricing.") && strings.HasSuffix(strings.ToLower(key), ".output_per_1k") {
		model := key[len("pricing.") : len(key)-len(".output_per_1k")]
		if model != "" {
			return model, "output_per_1k", true
		}
	}
	return "", "", false
}

func getConfigValue(key string) (string, error) {
	if model, field, ok := parsePricingKey(key); ok {
		price, exists := findModelPrice(model)
		if !exists {
			return "", fmt.Errorf("pricing not configured for model: %s", model)
		}
		if field == "input_per_1k" {
			return strconv.FormatFloat(price.InputPer1K, 'f', -1, 64), nil
		}
		return strconv.FormatFloat(price.OutputPer1K, 'f', -1, 64), nil
	}

	switch strings.ToLower(key) {
	case "model", "default_model":
		return config.DefaultModel, nil
	case "timezone":
		return config.Timezone, nil
	case "token_estimation.enabled":
		return strconv.FormatBool(config.TokenEstimation.Enabled), nil
	case "chars_per_token", "token_estimation.chars_per_token":
		return strconv.FormatFloat(config.TokenEstimation.CharsPerToken, 'f', -1, 64), nil
	case "max_logs", "display.max_logs_view":
		return strconv.Itoa(config.Display.MaxLogsView), nil
	case "show_workspace", "display.show_workspace":
		return strconv.FormatBool(config.Display.ShowWorkspace), nil
	case "display.reverse_order":
		return strconv.FormatBool(config.Display.ReverseOrder), nil
	case "display.quiet":
		return strconv.FormatBool(config.Display.Quiet), nil
	case "rotation", "storage.rotation":
		return config.Storage.Rotation, nil
	case "auto_run":
		return strconv.FormatBool(config.AutoRun), nil
	case "storage.log_file_prefix":
		return config.Storage.LogFilePrefix, nil
	case "pricing.currency":
		return config.Pricing.Currency, nil
	default:
		return "", fmt.Errorf("unknown config key: %s", key)
	}
}

func resetConfig() {
	loadConfig()
	config = defaultConfig()
	if err := saveConfig(); err != nil {
		fmt.Printf("Error saving config: %v\n", err)
		return
	}
	fmt.Printf("Config reset to defaults: %s\n", configPath())
}

func updateConfig(key string, values []string) {
	loadConfig()
	if len(values) == 0 {
		fmt.Println("Error: Missing value for config set.")
		return
	}
	val := values[0]

	if model, field, ok := parsePricingKey(key); ok {
		priceValue, err := strconv.ParseFloat(val, 64)
		if err != nil {
			fmt.Printf("Error: Invalid number for %s: %v\n", field, err)
			return
		}
		if config.Pricing.Models == nil {
			config.Pricing.Models = make(map[string]ModelPrice)
		}
		price := config.Pricing.Models[model]
		if field == "input_per_1k" {
			price.InputPer1K = priceValue
		} else {
			price.OutputPer1K = priceValue
		}
		config.Pricing.Models[model] = price
		err = saveConfig()
		if err == nil {
			fmt.Printf("Config updated: %s = %s\n", key, val)
		} else {
			fmt.Printf("Error saving config: %v\n", err)
		}
		return
	}

	switch strings.ToLower(key) {
	case "model", "default_model":
		config.DefaultModel = val
	case "timezone":
		config.Timezone = val
	case "auto_run":
		b, err := parseBool(val)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		config.AutoRun = b
	case "token_estimation.enabled":
		b, err := parseBool(val)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		config.TokenEstimation.Enabled = b
	case "chars_per_token":
		fallthrough
	case "token_estimation.chars_per_token":
		f, err := strconv.ParseFloat(val, 64)
		if err == nil {
			config.TokenEstimation.CharsPerToken = f
		} else {
			fmt.Printf("Error: Invalid number for chars_per_token: %v\n", err)
			return
		}
	case "max_logs":
		fallthrough
	case "display.max_logs_view":
		i, err := strconv.Atoi(val)
		if err == nil {
			config.Display.MaxLogsView = i
		} else {
			fmt.Printf("Error: Invalid number for max_logs: %v\n", err)
			return
		}
	case "show_workspace":
		fallthrough
	case "display.show_workspace":
		b, err := parseBool(val)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		config.Display.ShowWorkspace = b
	case "display.reverse_order":
		b, err := parseBool(val)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		config.Display.ReverseOrder = b
	case "display.quiet":
		b, err := parseBool(val)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		config.Display.Quiet = b
	case "rotation", "storage.rotation":
		config.Storage.Rotation = val
	case "storage.log_file_prefix":
		config.Storage.LogFilePrefix = val
	case "pricing.currency":
		config.Pricing.Currency = val
	default:
		fmt.Printf("Error: Unknown config key: %s\n", key)
		showConfigHelp()
		return
	}

	err := saveConfig()
	if err == nil {
		fmt.Printf("Config updated: %s = %s\n", key, val)
	} else {
		fmt.Printf("Error saving config: %v\n", err)
	}
}

type ModelStats struct {
	Model     string
	Logs      int
	TokensIn  int
	TokensOut int
	Cost      float64
	HasCost   bool
}

func collectModelStats(logs []LogEntry) []ModelStats {
	statsByModel := make(map[string]*ModelStats)
	for _, log := range logs {
		model := logModel(log)
		if statsByModel[model] == nil {
			statsByModel[model] = &ModelStats{Model: model}
		}
		row := statsByModel[model]
		row.Logs++
		row.TokensIn += log.TokensIn
		row.TokensOut += log.TokensOut
		if cost, ok := calculateLogCost(log); ok {
			row.Cost += cost
			row.HasCost = true
		}
	}

	var rows []ModelStats
	for _, row := range statsByModel {
		rows = append(rows, *row)
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Logs == rows[j].Logs {
			return strings.ToLower(rows[i].Model) < strings.ToLower(rows[j].Model)
		}
		return rows[i].Logs > rows[j].Logs
	})

	return rows
}

func showStats() {
	loadConfig()
	logs := getLogsFromAllFiles()
	total := len(logs)
	autoLogs := 0
	manualLogs := 0
	tIn := 0
	tOut := 0

	for _, log := range logs {
		if log.Category == "AutoLog" {
			autoLogs++
		} else {
			manualLogs++
		}
		tIn += log.TokensIn
		tOut += log.TokensOut
	}

	fmt.Printf(ColorBold + "📊 AgentTrack Usage Statistics (Across All Files)\n" + ColorReset)
	fmt.Println(ColorGray + strings.Repeat("-", 40) + ColorReset)
	fmt.Printf("Total Logs:       "+ColorCyan+"%d"+ColorReset+"\n", total)
	fmt.Printf("  - Auto:         "+ColorGreen+"%d"+ColorReset+"\n", autoLogs)
	fmt.Printf("  - Manual:       "+ColorPurple+"%d"+ColorReset+"\n", manualLogs)
	fmt.Printf("Total Tokens In:  "+ColorYellow+"%d"+ColorReset+"\n", tIn)
	fmt.Printf("Total Tokens Out: "+ColorYellow+"%d"+ColorReset+"\n", tOut)
	fmt.Printf(ColorBold+"Total Tokens:     "+ColorRed+"%d"+ColorReset+"\n", tIn+tOut)
}

func showModelStats() {
	loadConfig()
	rows := collectModelStats(getLogsFromAllFiles())
	if len(rows) == 0 {
		fmt.Println("No logs found.")
		return
	}

	fmt.Printf("%s%-24s | %-5s | %-9s | %-9s | %-9s | %-12s%s\n", ColorBold, "Model", "Logs", "Tokens In", "Tokens Out", "Total", "Cost", ColorReset)
	fmt.Println(ColorGray + strings.Repeat("=", 84) + ColorReset)
	for _, row := range rows {
		cost := "n/a"
		if row.HasCost {
			cost = fmt.Sprintf("%s %.4f", config.Pricing.Currency, row.Cost)
		}
		fmt.Printf("%-24s | %-5d | %-9d | %-9d | %-9d | %-12s\n", row.Model, row.Logs, row.TokensIn, row.TokensOut, row.TokensIn+row.TokensOut, cost)
	}
}

func showCostStats() {
	loadConfig()
	rows := collectModelStats(getLogsFromAllFiles())
	if len(rows) == 0 {
		fmt.Println("No logs found.")
		return
	}

	totalCost := 0.0
	var unpriced []string

	fmt.Printf("%sEstimated Token Cost (%s)%s\n", ColorBold, config.Pricing.Currency, ColorReset)
	fmt.Println(ColorGray + strings.Repeat("-", 40) + ColorReset)
	for _, row := range rows {
		if !row.HasCost {
			unpriced = append(unpriced, row.Model)
			fmt.Printf("%-24s %s\n", row.Model, "n/a")
			continue
		}
		totalCost += row.Cost
		fmt.Printf("%-24s %.4f\n", row.Model, row.Cost)
	}
	fmt.Printf("%sTotal Estimated Cost:%s %.4f %s\n", ColorBold, ColorReset, totalCost, config.Pricing.Currency)
	if len(unpriced) > 0 {
		fmt.Printf("Unpriced Models: %s\n", strings.Join(unpriced, ", "))
	}
}

func showSummary(period string) {
	loadConfig()
	logs := getLogsFromAllFiles()
	now := time.Now()

	var from, to time.Time
	var label string
	switch period {
	case "week":
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		d := now.AddDate(0, 0, -(weekday - 1))
		from = time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, now.Location())
		to = now
		label = "This Week (" + from.Format("Jan 2") + " – " + to.Format("Jan 2, 2006") + ")"
	case "month":
		from = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		to = now
		label = "This Month (" + from.Format("Jan 2006") + ")"
	default: // today
		from = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		to = now
		label = "Today (" + now.Format("Mon, Jan 2, 2006") + ")"
	}

	var filtered []LogEntry
	for _, l := range logs {
		t, err := time.ParseInLocation("2006-01-02 15:04:05", l.Timestamp, now.Location())
		if err != nil {
			continue
		}
		if (t.Equal(from) || t.After(from)) && (t.Equal(to) || t.Before(to)) {
			filtered = append(filtered, l)
		}
	}

	autoCount, manualCount := 0, 0
	tIn, tOut := 0, 0
	models := map[string]int{}
	categories := map[string]int{}
	tagCounts := map[string]int{}

	for _, l := range filtered {
		if l.Category == "AutoLog" {
			autoCount++
		} else {
			manualCount++
		}
		tIn += l.TokensIn
		tOut += l.TokensOut
		m := logModel(l)
		if m != "" {
			models[m]++
		}
		c := logCategory(l)
		if c != "" {
			categories[c]++
		}
		for _, tag := range l.Tags {
			tagCounts[tag]++
		}
	}

	fmt.Printf("%s📅 Summary: %s%s\n", ColorBold, label, ColorReset)
	fmt.Println(ColorGray + strings.Repeat("─", 50) + ColorReset)
	fmt.Printf("Total Logs:    %s%d%s  (Auto: %s%d%s | Manual: %s%d%s)\n",
		ColorCyan, len(filtered), ColorReset,
		ColorGreen, autoCount, ColorReset,
		ColorPurple, manualCount, ColorReset)
	fmt.Printf("Tokens:        In=%s%d%s  Out=%s%d%s  Total=%s%d%s\n",
		ColorYellow, tIn, ColorReset,
		ColorYellow, tOut, ColorReset,
		ColorBold, tIn+tOut, ColorReset)

	if len(models) > 0 {
		fmt.Println()
		fmt.Printf("%sModels used:%s\n", ColorCyan, ColorReset)
		type kv struct {
			k string
			v int
		}
		var sorted []kv
		for k, v := range models {
			sorted = append(sorted, kv{k, v})
		}
		sort.Slice(sorted, func(i, j int) bool { return sorted[i].v > sorted[j].v })
		for _, item := range sorted {
			fmt.Printf("  %-30s %s%d logs%s\n", item.k, ColorGreen, item.v, ColorReset)
		}
	}

	if len(categories) > 0 {
		fmt.Println()
		fmt.Printf("%sCategories:%s\n", ColorCyan, ColorReset)
		type kv struct {
			k string
			v int
		}
		var sorted []kv
		for k, v := range categories {
			sorted = append(sorted, kv{k, v})
		}
		sort.Slice(sorted, func(i, j int) bool { return sorted[i].v > sorted[j].v })
		for _, item := range sorted {
			fmt.Printf("  %-20s %s%d%s\n", item.k, ColorGreen, item.v, ColorReset)
		}
	}

	if len(tagCounts) > 0 {
		fmt.Println()
		fmt.Printf("%sTags:%s\n", ColorCyan, ColorReset)
		type kv struct {
			k string
			v int
		}
		var sorted []kv
		for k, v := range tagCounts {
			sorted = append(sorted, kv{k, v})
		}
		sort.Slice(sorted, func(i, j int) bool { return sorted[i].v > sorted[j].v })
		tags := []string{}
		for _, item := range sorted {
			tags = append(tags, fmt.Sprintf("%s(%d)", item.k, item.v))
		}
		fmt.Printf("  %s\n", strings.Join(tags, "  "))
	}

	if len(filtered) == 0 {
		fmt.Printf("%sNo logs found for this period.%s\n", ColorGray, ColorReset)
	}
}

func showTagList() {
	loadConfig()
	logs := getLogsFromAllFiles()
	tagCounts := map[string]int{}
	for _, l := range logs {
		for _, tag := range l.Tags {
			tagCounts[tag]++
		}
	}
	if len(tagCounts) == 0 {
		fmt.Println("No tags found.")
		return
	}

	type kv struct {
		k string
		v int
	}
	var sorted []kv
	for k, v := range tagCounts {
		sorted = append(sorted, kv{k, v})
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].v > sorted[j].v })

	fmt.Printf("%s%-24s | %s%s\n", ColorBold, "Tag", "Count", ColorReset)
	fmt.Println(ColorGray + strings.Repeat("=", 36) + ColorReset)
	for _, item := range sorted {
		fmt.Printf("%-24s | %s%d%s\n", item.k, ColorGreen, item.v, ColorReset)
	}
}

func showStatsToday() {
	loadConfig()
	logs := getLogsFromAllFiles()
	today := time.Now().Format("2006-01-02")
	var todayLogs []LogEntry
	for _, l := range logs {
		if strings.HasPrefix(l.Timestamp, today) {
			todayLogs = append(todayLogs, l)
		}
	}

	tIn, tOut, auto, manual := 0, 0, 0, 0
	for _, l := range todayLogs {
		tIn += l.TokensIn
		tOut += l.TokensOut
		if l.Category == "AutoLog" {
			auto++
		} else {
			manual++
		}
	}

	fmt.Printf("%s📊 Today's Stats (%s)%s\n", ColorBold, today, ColorReset)
	fmt.Println(ColorGray + strings.Repeat("─", 40) + ColorReset)
	fmt.Printf("Total Logs:   %s%d%s  (Auto: %d | Manual: %d)\n", ColorCyan, len(todayLogs), ColorReset, auto, manual)
	fmt.Printf("Tokens In:    %s%d%s\n", ColorYellow, tIn, ColorReset)
	fmt.Printf("Tokens Out:   %s%d%s\n", ColorYellow, tOut, ColorReset)
	fmt.Printf("%sTotal Tokens: %s%d%s\n", ColorBold, ColorRed, tIn+tOut, ColorReset)
}

func exportLogs(format string) {
	loadConfig()
	logs := getLogsFromAllFiles()
	if len(logs) == 0 {
		fmt.Println("No logs to export.")
		return
	}

	format = strings.ToLower(format)
	if format != "md" && format != "csv" && format != "json" {
		fmt.Printf("Format '%s' not supported yet. Using 'md'.\n", format)
		format = "md"
	}

	filename := fmt.Sprintf("atrack_export_%s.%s", time.Now().Format("20060102_150405"), format)
	file, err := os.Create(filename)
	if err != nil {
		fmt.Printf("Error creating export file: %v\n", err)
		return
	}
	defer file.Close()

	switch format {
	case "md":
		file.WriteString("# AgentTrack Activity Export\n\n")
		file.WriteString(fmt.Sprintf("Exported on: %s\n\n", time.Now().Format(time.RFC1123)))

		for _, log := range logs {
			file.WriteString(fmt.Sprintf("## [%s] %s\n", log.Timestamp, logCategory(log)))
			file.WriteString(fmt.Sprintf("**Model:** %s | **Tokens:** In=%d, Out=%d\n\n", logModel(log), log.TokensIn, log.TokensOut))
			if len(log.Tags) > 0 {
				file.WriteString(fmt.Sprintf("**Tags:** %s\n\n", strings.Join(log.Tags, ", ")))
			}
			if log.Category == "AutoLog" {
				file.WriteString(fmt.Sprintf("### Q: %s\n\n", log.Question))
				file.WriteString(fmt.Sprintf("### A:\n%s\n\n", log.Answer))
			} else {
				file.WriteString(fmt.Sprintf("%s\n\n", log.Message))
			}
			file.WriteString("---\n\n")
		}
	case "json":
		data, err := json.MarshalIndent(logs, "", "  ")
		if err != nil {
			fmt.Printf("Error encoding JSON export: %v\n", err)
			return
		}
		if _, err := file.Write(data); err != nil {
			fmt.Printf("Error writing JSON export: %v\n", err)
			return
		}
	case "csv":
		writer := csv.NewWriter(file)
		defer writer.Flush()

		if err := writer.Write([]string{"timestamp", "category", "message", "question", "answer", "workspace", "model", "tokens_in", "tokens_out", "is_estimated", "tags"}); err != nil {
			fmt.Printf("Error writing CSV header: %v\n", err)
			return
		}
		for _, log := range logs {
			record := []string{
				log.Timestamp,
				logCategory(log),
				log.Message,
				log.Question,
				log.Answer,
				log.Workspace,
				logModel(log),
				strconv.Itoa(log.TokensIn),
				strconv.Itoa(log.TokensOut),
				strconv.FormatBool(log.IsEstimated),
				strings.Join(log.Tags, ","),
			}
			if err := writer.Write(record); err != nil {
				fmt.Printf("Error writing CSV record: %v\n", err)
				return
			}
		}
		if err := writer.Error(); err != nil {
			fmt.Printf("Error finalizing CSV export: %v\n", err)
			return
		}
	}

	fmt.Printf("Logs exported successfully to: %s\n", filename)
}

func printUsageItem(command string, description string) {
	fmt.Printf("  %s%s%s\n", ColorGreen, command, ColorReset)
	fmt.Printf("      %s\n", description)
}

func printUsage() {
	fmt.Println(ColorGreen + ColorBold + splashBanner + ColorReset)
	fmt.Printf("            %s%sv%s%s | Agent Track: The Cross-Platform AI Activity Tracker\n\n", ColorGreen, ColorBold, Version, ColorReset)

	fmt.Println(ColorYellow + "📚 Usage" + ColorReset)
	fmt.Println("  atrack <command> [arguments]")
	fmt.Println()

	fmt.Println(ColorCyan + "✨ Essential Commands" + ColorReset)
	printUsageItem(`atrack log "message"`, "Add a manual activity log")
	printUsageItem(`atrack list`, "Show recent logs (use 'list last' for the latest)")
	printUsageItem(`atrack dashboard`, "Open the interactive CLI dashboard")
	printUsageItem(`atrack stats [today|model]`, "Show your activity statistics")
	printUsageItem(`atrack summary`, "Get a quick activity summary")
	printUsageItem(`atrack autostart [install|uninstall|run]`, "Manage the background auto-run service")
	printUsageItem(`atrack help | -h`, "Show all available commands and detailed usage")
	fmt.Println()

	fmt.Println(ColorCyan + "⚡ Quick start" + ColorReset)
	fmt.Println("  " + ColorGreen + `atrack log "Working on new feature" -c Dev -t golang` + ColorReset)
	fmt.Println("  " + ColorGreen + `atrack stats today` + ColorReset)
	fmt.Println()

	loadConfig()
	logs := getLogsFromAllFiles()
	fmt.Printf("%s💡 Quick Stats:%s You have %s%d%s total logs recorded.\n", ColorGray, ColorReset, ColorCyan, len(logs), ColorReset)
}

func printFullUsage() {
	fmt.Println(ColorGreen + ColorBold + splashBanner + ColorReset)
	fmt.Printf("            %s%sv%s%s | Agent Track: Detailed Help\n\n", ColorGreen, ColorBold, Version, ColorReset)

	fmt.Println(ColorYellow + "📚 Full Command List" + ColorReset)
	fmt.Println()

	fmt.Println(ColorCyan + "📝 Logging & Viewing" + ColorReset)
	printUsageItem(`atrack log "message" [-c category] [-t tag1,tag2]`, "Add a manual log entry")
	printUsageItem(`atrack auto "q" "a" "model" in out`, "Save an AI Q&A log (internal use/scripts)")
	printUsageItem(`atrack list [date] [--from YYYY-MM-DD --to YYYY-MM-DD]`, "Filter logs by date")
	printUsageItem(`atrack list model "name"|all`, "List logs by AI model")
	printUsageItem(`atrack list category "name"|all`, "List logs by category")
	printUsageItem(`atrack watch`, "Monitor logs in real-time")
	printUsageItem(`atrack dashboard`, "Open the interactive dashboard")
	fmt.Println()

	fmt.Println(ColorCyan + "🔍 Search & Analysis" + ColorReset)
	printUsageItem(`atrack search "keyword"`, "Find logs by keyword")
	printUsageItem(`atrack search model|tag "value"`, "Find logs by model or tag")
	printUsageItem(`atrack summary [today|week|month]`, "Periodic activity summary")
	printUsageItem(`atrack stats | model | cost | today`, "Detailed statistics and costs")
	printUsageItem(`atrack tag list`, "List all used tags")
	fmt.Println()

	fmt.Println(ColorCyan + "🛠 Management" + ColorReset)
	printUsageItem(`atrack init`, "Initialize AgentTrack rules in the current project")
	printUsageItem(`atrack edit <index> [field] <value>`, "Edit a log entry")
	printUsageItem(`atrack delete <index>`, "Delete a log entry")
	printUsageItem(`atrack export [md|csv|json]`, "Export data to files")
	printUsageItem(`atrack pricing sync [all|model]`, "Sync model prices from OpenRouter")
	printUsageItem(`atrack config [show|get|set|reset]`, "Manage application configuration")
	printUsageItem(`atrack autostart [install|uninstall|run]`, "Install or run the auto-start service")
	printUsageItem(`atrack reset [--yes]`, "Delete all logs and reset config to defaults")
	printUsageItem(`atrack uninstall [--yes]`, "Remove app data, shell hooks, and local atrack binary")
	printUsageItem(`atrack update`, "Attempt to self-update or show update instructions")
	printUsageItem(`atrack info`, "Show system paths and info")
	printUsageItem(`atrack version`, "Show app version")
	printUsageItem(`atrack clear`, "Wipe all log data")
	fmt.Println()
}
