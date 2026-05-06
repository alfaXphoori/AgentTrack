package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type TokenEstimationConfig struct {
	Enabled       bool    `json:"enabled"`
	CharsPerToken float64 `json:"chars_per_token"`
}

type DisplayConfig struct {
	ShowWorkspace bool `json:"show_workspace"`
	ReverseOrder  bool `json:"reverse_order"`
	MaxLogsView   int  `json:"max_logs_view"`
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
	Version     = "0.12"
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
 _____               _   _____  _      _____
|_   _|             | | /  __ \| |    |_   _|
  | | _ __ __ _  ___| | | /  \/| |      | |
  | || '__/ _` + "`" + ` |/ __| | | |    | |      | |
  | || | | (_| | (__| | | \__/\| |____ _| |_
  \_/|_|  \__,_|\___|_|  \____/\_____/\___/`

func defaultConfig() Config {
	return Config{
		ProjectName:  "TrackCLI Activity Tracker",
		DefaultModel: "gemini-1.5-flash",
		Timezone:     "Asia/Bangkok",
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
			LogFilePrefix: "trackcli_logs",
			Rotation:      "monthly",
		},
		Pricing: PricingConfig{
			Currency: "USD",
			Models:   map[string]ModelPrice{},
		},
	}
}

func getAppDir() string {
	if envDir := os.Getenv("TRACKCLI_HOME"); envDir != "" {
		return envDir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	dir := filepath.Join(home, ".trackcli")
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
		return filepath.Join(appDir, fmt.Sprintf("%s_%s.json", config.Storage.LogFilePrefix, t.Format("2006_01")))
	}
	return filepath.Join(appDir, config.Storage.LogFilePrefix+".json")
}

func getAllLogFiles() []string {
	files, _ := filepath.Glob(filepath.Join(appDir, config.Storage.LogFilePrefix+"*.json"))
	sort.Strings(files)
	return files
}

func getLogsFromAllFiles() []LogEntry {
	var allLogs []LogEntry
	files := getAllLogFiles()
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err == nil {
			var logs []LogEntry
			json.Unmarshal(data, &logs)
			allLogs = append(allLogs, logs...)
		}
	}
	return allLogs
}

func saveLogsToFile(path string, logs []LogEntry) {
	data, _ := json.MarshalIndent(logs, "", "  ")
	os.WriteFile(path, data, 0644)
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

func loadOpenRouterPricing() map[string]ModelPrice {
	if openRouterPricingLoaded {
		return openRouterPricingCache
	}

	openRouterPricingLoaded = true
	openRouterPricingCache = make(map[string]ModelPrice)

	url := os.Getenv("TRACKCLI_OPENROUTER_MODELS_URL")
	if url == "" {
		url = "https://openrouter.ai/api/v1/models"
	}
	if strings.EqualFold(url, "off") {
		return openRouterPricingCache
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return openRouterPricingCache
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return openRouterPricingCache
	}

	var payload OpenRouterModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return openRouterPricingCache
	}

	for _, entry := range payload.Data {
		price := ModelPrice{
			InputPer1K:  parseOpenRouterRate(entry.Pricing.Prompt),
			OutputPer1K: parseOpenRouterRate(entry.Pricing.Completion),
		}
		registerOpenRouterAlias(openRouterPricingCache, entry.ID, price)
	}

	return openRouterPricingCache
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
	var logs []LogEntry
	data, err := os.ReadFile(path)
	if err == nil {
		json.Unmarshal(data, &logs)
	}

	if entry.Category == "" {
		entry.Category = "General"
	}
	if entry.Workspace == "" {
		entry.Workspace, _ = os.Getwd()
	}
	if entry.Model == "" {
		entry.Model = config.DefaultModel
	}

	logs = append(logs, entry)
	saveLogsToFile(path, logs)

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

	fmt.Printf("%s%-5s | %-20s | %-12s | Metadata / Q&A%s\n", ColorBold, "ID", "Timestamp", "Category", ColorReset)
	fmt.Println(ColorGray + strings.Repeat("=", 110) + ColorReset)

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
		metadata := fmt.Sprintf(ColorYellow+"[Model: %s | Tokens: In=%d%s, Out=%d%s]"+ColorReset, logModel(log), log.TokensIn, estIn, log.TokensOut, estOut)
		workspace := fmt.Sprintf(ColorCyan+"[WS: %s]"+ColorReset, log.Workspace)
		tagsLine := ""
		if len(log.Tags) > 0 {
			tagsLine = fmt.Sprintf(ColorYellow+"[Tags: %s]"+ColorReset, strings.Join(log.Tags, ", "))
		}

		if category == "AutoLog" && log.Question != "" && log.Answer != "" {
			fmt.Printf(ColorBlue+"#%-4d"+ColorReset+" | "+ColorGreen+"%-20s"+ColorReset+" | "+ColorPurple+"%-12s"+ColorReset+" | %s\n", displayID, log.Timestamp, category, metadata)
			if config.Display.ShowWorkspace {
				fmt.Printf("%-5s | %-20s | %-12s | 📁 %s\n", "", "", "", workspace)
			}
			if tagsLine != "" {
				fmt.Printf("%-5s | %-20s | %-12s | 🏷️ %s\n", "", "", "", tagsLine)
			}
			fmt.Printf("%-5s | %-20s | %-12s | 👤 "+ColorBold+"Q: %s"+ColorReset+"\n", "", "", "", log.Question)
			fmt.Printf("%-5s | %-20s | %-12s | 🤖 "+ColorBold+"A: %s"+ColorReset+"\n", "", "", "", log.Answer)
		} else {
			fmt.Printf(ColorBlue+"#%-4d"+ColorReset+" | "+ColorGreen+"%-20s"+ColorReset+" | "+ColorPurple+"%-12s"+ColorReset+" | 📝 %s\n", displayID, log.Timestamp, category, log.Message)
			fmt.Printf("%-5s | %-20s | %-12s | %s\n", "", "", "", metadata)
			if tagsLine != "" {
				fmt.Printf("%-5s | %-20s | %-12s | 🏷️ %s\n", "", "", "", tagsLine)
			}
		}
		fmt.Println(ColorGray + strings.Repeat("-", 110) + ColorReset)
	}
}

func showLastLog() {
	loadConfig()
	logs := getLogsFromAllFiles()
	if len(logs) == 0 {
		fmt.Println("No logs found.")
		return
	}
	lastLog := []LogEntry{logs[len(logs)-1]}
	renderLogs(lastLog)
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
	data, err := os.ReadFile(path)
	if err != nil {
		return "", nil, -1, LogEntry{}, fmt.Errorf("error reading log file: %v", err)
	}

	var fileLogs []LogEntry
	json.Unmarshal(data, &fileLogs)

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
		fmt.Println("Error: Usage: trackcli edit <index> [field] <value>")
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
		os.WriteFile(file, []byte("[]"), 0644)
	}
	fmt.Println("All log files cleared.")
}

func main() {
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
		if len(os.Args) > 5 {
			tIn, _ = strconv.Atoi(os.Args[5])
		}
		if len(os.Args) > 6 {
			tOut, _ = strconv.Atoi(os.Args[6])
		}

		isEst := false
		if tIn == 0 && tOut == 0 {
			tIn = estimateTokens(question)
			tOut = estimateTokens(answer)
			isEst = true
		}

		addLog(LogEntry{
			Category:    "AutoLog",
			Question:    question,
			Answer:      answer,
			Model:       model,
			TokensIn:    tIn,
			TokensOut:   tOut,
			IsEstimated: isEst,
		})

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
			showLastLog()
			return
		case "model":
			if len(remaining) < 2 {
				fmt.Println("Error: Usage: trackcli list model <model> [--from YYYY-MM-DD] [--to YYYY-MM-DD]")
				return
			}
			listLogsByModel(remaining[1], dateFilter)
		case "category":
			if len(remaining) < 2 {
				fmt.Println("Error: Usage: trackcli list category <category> [--from YYYY-MM-DD] [--to YYYY-MM-DD]")
				return
			}
			listLogsByCategory(remaining[1], dateFilter)
		default:
			if len(remaining) > 1 {
				fmt.Println("Error: Usage: trackcli list [date] [--from YYYY-MM-DD] [--to YYYY-MM-DD]")
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
			fmt.Println("Error: Usage: trackcli edit <index> [field] <value>")
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
				fmt.Println("Error: Usage: trackcli search model <model> [--from YYYY-MM-DD] [--to YYYY-MM-DD]")
				return
			}
			searchLogsByModel(strings.Join(remaining[1:], " "), dateFilter)
		case "tag":
			if len(remaining) < 2 {
				fmt.Println("Error: Usage: trackcli search tag <tag> [--from YYYY-MM-DD] [--to YYYY-MM-DD]")
				return
			}
			searchLogsByTag(strings.Join(remaining[1:], " "), dateFilter)
		default:
			searchLogs(strings.Join(remaining, " "), dateFilter)
		}

	case "clear":
		clearLogs()

	case "info":
		fmt.Printf("TrackCLI Global CLI\n")
		fmt.Printf("Version:       %s\n", Version)
		fmt.Printf("App Directory: %s\n", appDir)
		fmt.Printf("Config File:   %s\n", filepath.Join(appDir, "config.json"))
		fmt.Printf("Current Log:   %s\n", getLogPath(time.Now()))
		fmt.Printf("Total Files:   %d\n", len(getAllLogFiles()))

	case "version":
		fmt.Printf("TrackCLI version %s\n", Version)

	case "stats":
		if len(os.Args) > 2 {
			switch strings.ToLower(os.Args[2]) {
			case "model":
				showModelStats()
			case "cost":
				showCostStats()
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
				fmt.Println("Error: Usage: trackcli config set <key> <value>")
				return
			}
			updateConfig(os.Args[3], os.Args[4:])
		case "get":
			if len(os.Args) < 4 {
				fmt.Println("Error: Usage: trackcli config get <key>")
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
	fmt.Println("  trackcli config show")
	fmt.Println("      Show the full config file")
	fmt.Println("  trackcli config get <key>")
	fmt.Println("      Show one config value")
	fmt.Println("  trackcli config set <key> <value>")
	fmt.Println("      Update one config value")
	fmt.Println("  trackcli config reset")
	fmt.Println("      Reset config to defaults")
	fmt.Println()
	fmt.Println("Available keys:")
	fmt.Println("  default_model")
	fmt.Println("  timezone")
	fmt.Println("  token_estimation.enabled")
	fmt.Println("  token_estimation.chars_per_token")
	fmt.Println("  display.show_workspace")
	fmt.Println("  display.reverse_order")
	fmt.Println("  display.max_logs_view")
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
	case "rotation", "storage.rotation":
		return config.Storage.Rotation, nil
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
	case "rotation":
		fallthrough
	case "storage.rotation":
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

	fmt.Printf(ColorBold + "📊 TrackCLI Usage Statistics (Across All Files)\n" + ColorReset)
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

	filename := fmt.Sprintf("trackcli_export_%s.%s", time.Now().Format("20060102_150405"), format)
	file, err := os.Create(filename)
	if err != nil {
		fmt.Printf("Error creating export file: %v\n", err)
		return
	}
	defer file.Close()

	switch format {
	case "md":
		file.WriteString("# TrackCLI Activity Export\n\n")
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
	fmt.Printf("            %s%sv%s%s | The Cross-Platform AI Activity Tracker\n\n", ColorGreen, ColorBold, Version, ColorReset)

	fmt.Println(ColorYellow + "📚 Usage" + ColorReset)
	fmt.Println("  trackcli <command> [arguments]")
	fmt.Println()

	fmt.Println(ColorCyan + "✨ Most common" + ColorReset)
	printUsageItem(`trackcli log "message" [-c category] [-t tag1,tag2]`, "Add a manual log entry with optional tags")
	printUsageItem(`trackcli auto "question" "answer" "model" tokens_in tokens_out`, "Save an AI Q&A log")
	printUsageItem(`trackcli list [date] [--from YYYY-MM-DD] [--to YYYY-MM-DD]`, "Show logs with exact-date or date-range filters")
	printUsageItem(`trackcli list last`, "Quickly view only the most recent log entry")
	printUsageItem(`trackcli list model "model_name"|all [--from ...] [--to ...]`, "List logs for one model, or summarize all models")
	printUsageItem(`trackcli list category "category"|all [--from ...] [--to ...]`, "List logs for one category, or summarize all categories")
	printUsageItem(`trackcli search "keyword" [--from YYYY-MM-DD] [--to YYYY-MM-DD]`, "Find logs by keyword with optional date range")
	printUsageItem(`trackcli search model "model_name" [--from ...] [--to ...]`, "Find logs by AI model")
	printUsageItem(`trackcli search tag "tag" [--from ...] [--to ...]`, "Find logs by tag")
	fmt.Println()

	fmt.Println(ColorCyan + "🛠 Management" + ColorReset)
	printUsageItem(`trackcli edit <index> [field] <value>`, "Edit a log entry; default field is message or answer")
	printUsageItem(`trackcli delete <index>`, "Delete a single log entry")
	printUsageItem(`trackcli stats | stats model | stats cost`, "Show total stats, per-model stats, or estimated cost")
	printUsageItem(`trackcli export [md|csv|json]`, "Export logs to Markdown, CSV, or JSON")
	printUsageItem(`trackcli config [show|get|set|reset]`, "View or update app configuration")
	printUsageItem(`trackcli info`, "Show config and storage paths")
	printUsageItem(`trackcli version`, "Show the current version")
	printUsageItem(`trackcli clear`, "Clear all saved logs")
	fmt.Println()

	fmt.Println(ColorCyan + "⚡ Quick start" + ColorReset)
	fmt.Println("  " + ColorGreen + `trackcli log "Started research on project" -c Research -t planning,cli` + ColorReset)
	fmt.Println("  " + ColorGreen + `trackcli search tag "planning" --from 2026-05-01 --to 2026-05-31` + ColorReset)
	fmt.Println()

	loadConfig()
	logs := getLogsFromAllFiles()
	fmt.Printf("%s💡 Quick Stats:%s You have %s%d%s total logs recorded.\n", ColorGray, ColorReset, ColorCyan, len(logs), ColorReset)
}
