package main

import (
	"encoding/json"
	"fmt"
	"math"
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

type Config struct {
	ProjectName     string                `json:"project_name"`
	DefaultModel    string                `json:"default_model"`
	Timezone        string                `json:"timezone"`
	TokenEstimation TokenEstimationConfig `json:"token_estimation"`
	Display         DisplayConfig         `json:"display"`
	Storage         StorageConfig         `json:"storage"`
}

type LogEntry struct {
	Timestamp   string `json:"timestamp"`
	Category    string `json:"category"`
	Message     string `json:"message"`
	Question    string `json:"question"`
	Answer      string `json:"answer"`
	Workspace   string `json:"workspace"`
	Model       string `json:"model"`
	TokensIn    int    `json:"tokens_in"`
	TokensOut   int    `json:"tokens_out"`
	IsEstimated bool   `json:"is_estimated"`
}

var config Config
var appDir string

func getAppDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	dir := filepath.Join(home, ".aikore")
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		os.MkdirAll(dir, 0755)
	}
	return dir
}

func loadConfig() {
	appDir = getAppDir()
	config = Config{
		ProjectName:  "AiKore Activity Tracker",
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
			LogFilePrefix: "aikore_logs",
			Rotation:      "monthly",
		},
	}

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

func estimateTokens(text string) int {
	if text == "" || !config.TokenEstimation.Enabled {
		return 0
	}
	return int(math.Ceil(float64(len(text)) / config.TokenEstimation.CharsPerToken))
}

func addLog(entry LogEntry) {
	loadConfig()
	loc, _ := time.LoadLocation(config.Timezone)
	if loc == nil {
		loc = time.Local
	}
	now := time.Now().In(loc)
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
		estStr = " [Tokens Estimated]"
	}
	fmt.Printf("Log added: [%s] (%s)%s\n", entry.Timestamp, entry.Category, estStr)
}

func searchLogs(keyword string) {
	loadConfig()
	logs := getLogsFromAllFiles()
	keyword = strings.ToLower(keyword)
	var found []LogEntry

	for _, log := range logs {
		if strings.Contains(strings.ToLower(log.Message), keyword) ||
			strings.Contains(strings.ToLower(log.Question), keyword) ||
			strings.Contains(strings.ToLower(log.Answer), keyword) ||
			strings.Contains(strings.ToLower(log.Category), keyword) {
			found = append(found, log)
		}
	}

	if len(found) == 0 {
		fmt.Printf("No logs found matching: %s\n", keyword)
		return
	}

	renderLogs(found)
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

	fmt.Printf("%-5s | %-20s | %-12s | Metadata / Q&A\n", "ID", "Timestamp", "Category")
	fmt.Println(strings.Repeat("=", 110))

	for i, log := range displayLogs {
		displayID := i
		if config.Display.ReverseOrder {
			displayID = len(logs) - 1 - i
		}

		estIn, estOut := "", ""
		if log.IsEstimated {
			estIn = " (est)"
			estOut = " (est)"
		}
		metadata := fmt.Sprintf("[Model: %s | Tokens: In=%d%s, Out=%d%s]", log.Model, log.TokensIn, estIn, log.TokensOut, estOut)
		workspace := fmt.Sprintf("[WS: %s]", log.Workspace)

		if log.Category == "AutoLog" && log.Question != "" && log.Answer != "" {
			fmt.Printf("#%-4d | %-20s | %-12s | %s\n", displayID, log.Timestamp, log.Category, metadata)
			if config.Display.ShowWorkspace {
				fmt.Printf("%-5s | %-20s | %-12s | %s\n", "", "", "", workspace)
			}
			fmt.Printf("%-5s | %-20s | %-12s | Q: %s\n", "", "", "", log.Question)
			fmt.Printf("%-5s | %-20s | %-12s | A: %s\n", "", "", "", log.Answer)
		} else {
			fmt.Printf("#%-4d | %-20s | %-12s | %s\n", displayID, log.Timestamp, log.Category, log.Message)
			fmt.Printf("%-5s | %-20s | %-12s | %s\n", "", "", "", metadata)
		}
		fmt.Println(strings.Repeat("-", 110))
	}
}

func listLogs(dateFilter string) {
	loadConfig()
	logs := getLogsFromAllFiles()
	if len(logs) == 0 {
		fmt.Println("No logs found.")
		return
	}

	if dateFilter != "" {
		var filtered []LogEntry
		for _, log := range logs {
			if strings.HasPrefix(log.Timestamp, dateFilter) {
				filtered = append(filtered, log)
			}
		}
		if len(filtered) == 0 {
			fmt.Printf("No logs found for date: %s\n", dateFilter)
			return
		}
		renderLogs(filtered)
	} else {
		renderLogs(logs)
	}
}

func deleteLog(index int) {
	loadConfig()
	logs := getLogsFromAllFiles()
	if index < 0 || index >= len(logs) {
		fmt.Printf("Error: Index %d out of range (0-%d)\n", index, len(logs)-1)
		return
	}

	target := logs[index]

	t, err := time.Parse("2006-01-02 15:04:05", target.Timestamp)
	if err != nil {
		fmt.Printf("Error parsing timestamp for deletion: %v\n", err)
		return
	}

	path := getLogPath(t)
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("Error reading log file for deletion: %v\n", err)
		return
	}

	var fileLogs []LogEntry
	json.Unmarshal(data, &fileLogs)

	var newFileLogs []LogEntry
	found := false
	for _, l := range fileLogs {
		if !found && l.Timestamp == target.Timestamp && l.Category == target.Category &&
			l.Message == target.Message && l.Question == target.Question {
			found = true
			continue
		}
		newFileLogs = append(newFileLogs, l)
	}

	if !found {
		fmt.Println("Error: Could not find the exact log entry in the file.")
		return
	}

	saveLogsToFile(path, newFileLogs)
	fmt.Printf("Log #%d [%s] deleted successfully.\n", index, target.Timestamp)
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
		for i := 3; i < len(os.Args)-1; i++ {
			if os.Args[i] == "-c" || os.Args[i] == "--category" {
				category = os.Args[i+1]
				break
			}
		}
		addLog(LogEntry{Message: message, Category: category})

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
		date := ""
		if len(os.Args) > 2 {
			date = os.Args[2]
		}
		listLogs(date)

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

	case "search":
		if len(os.Args) < 3 {
			fmt.Println("Error: Please provide a keyword to search.")
			return
		}
		searchLogs(os.Args[2])

	case "clear":
		clearLogs()

	case "info":
		fmt.Printf("AiKore Global CLI\n")
		fmt.Printf("App Directory: %s\n", appDir)
		fmt.Printf("Config File:   %s\n", filepath.Join(appDir, "config.json"))
		fmt.Printf("Current Log:   %s\n", getLogPath(time.Now()))
		fmt.Printf("Total Files:   %d\n", len(getAllLogFiles()))

	case "stats":
		showStats()

	case "export":
		format := "md"
		if len(os.Args) > 2 {
			format = os.Args[2]
		}
		exportLogs(format)

	case "config":
		if len(os.Args) > 3 && os.Args[2] == "set" {
			updateConfig(os.Args[3], os.Args[4:])
		} else {
			showConfig()
		}

	default:
		printUsage()
	}
}

func showConfig() {
	loadConfig()
	data, _ := json.MarshalIndent(config, "", "  ")
	fmt.Printf("Current Configuration (%s):\n", filepath.Join(appDir, "config.json"))
	fmt.Println(string(data))
}

func updateConfig(key string, values []string) {
	loadConfig()
	if len(values) == 0 {
		fmt.Println("Error: Missing value for config set.")
		return
	}
	val := values[0]

	switch strings.ToLower(key) {
	case "model", "default_model":
		config.DefaultModel = val
	case "timezone":
		config.Timezone = val
	case "chars_per_token":
		f, err := strconv.ParseFloat(val, 64)
		if err == nil {
			config.TokenEstimation.CharsPerToken = f
		} else {
			fmt.Printf("Error: Invalid number for chars_per_token: %v\n", err)
			return
		}
	case "max_logs":
		i, err := strconv.Atoi(val)
		if err == nil {
			config.Display.MaxLogsView = i
		} else {
			fmt.Printf("Error: Invalid number for max_logs: %v\n", err)
			return
		}
	case "show_workspace":
		config.Display.ShowWorkspace = (strings.ToLower(val) == "true")
	case "rotation":
		config.Storage.Rotation = val
	default:
		fmt.Printf("Error: Unknown config key: %s\n", key)
		return
	}

	configPath := filepath.Join(appDir, "config.json")
	data, _ := json.MarshalIndent(config, "", "  ")
	err := os.WriteFile(configPath, data, 0644)
	if err == nil {
		fmt.Printf("Config updated: %s = %s\n", key, val)
	} else {
		fmt.Printf("Error saving config: %v\n", err)
	}
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

	fmt.Printf("AiKore Usage Statistics (Across All Files)\n")
	fmt.Println(strings.Repeat("-", 40))
	fmt.Printf("Total Logs:     %d\n", total)
	fmt.Printf("  - Auto:       %d\n", autoLogs)
	fmt.Printf("  - Manual:     %d\n", manualLogs)
	fmt.Printf("Total Tokens In:  %d\n", tIn)
	fmt.Printf("Total Tokens Out: %d\n", tOut)
	fmt.Printf("Total Tokens:     %d\n", tIn+tOut)
}

func exportLogs(format string) {
	loadConfig()
	logs := getLogsFromAllFiles()
	if len(logs) == 0 {
		fmt.Println("No logs to export.")
		return
	}

	if format != "md" {
		fmt.Printf("Format '%s' not supported yet. Using 'md'.\n", format)
		format = "md"
	}

	filename := fmt.Sprintf("aikore_export_%s.md", time.Now().Format("20060102_150405"))
	file, err := os.Create(filename)
	if err != nil {
		fmt.Printf("Error creating export file: %v\n", err)
		return
	}
	defer file.Close()

	file.WriteString("# AiKore Activity Export\n\n")
	file.WriteString(fmt.Sprintf("Exported on: %s\n\n", time.Now().Format(time.RFC1123)))

	for _, log := range logs {
		file.WriteString(fmt.Sprintf("## [%s] %s\n", log.Timestamp, log.Category))
		if log.Category == "AutoLog" {
			file.WriteString(fmt.Sprintf("**Model:** %s | **Tokens:** In=%d, Out=%d\n\n", log.Model, log.TokensIn, log.TokensOut))
			file.WriteString(fmt.Sprintf("### Q: %s\n\n", log.Question))
			file.WriteString(fmt.Sprintf("### A:\n%s\n\n", log.Answer))
		} else {
			file.WriteString(fmt.Sprintf("%s\n\n", log.Message))
		}
		file.WriteString("---\n\n")
	}

	fmt.Printf("Logs exported successfully to: %s\n", filename)
}

func printUsage() {
	fmt.Println("AiKore Activity Tracker (Go)")
	fmt.Println("Usage:")
	fmt.Println("  aikore log \"message\" [-c category]")
	fmt.Println("  aikore auto \"question\" \"answer\" \"model\" \"tokens_in\" \"tokens_out\"")
	fmt.Println("  aikore list [date]")
	fmt.Println("  aikore search \"keyword\"")
	fmt.Println("  aikore delete <index>")
	fmt.Println("  aikore stats")
	fmt.Println("  aikore export [md]")
	fmt.Println("  aikore clear")
	fmt.Println("  aikore info")
}
