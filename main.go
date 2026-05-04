package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
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
	LogFile string `json:"log_file"`
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
var dbPath string

func loadConfig() {
	config = Config{
		ProjectName:  "AiKore Activity Tracker",
		DefaultModel: "unknown",
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
			LogFile: "aikore_logs.json",
		},
	}

	cwd, _ := os.Getwd()
	configPath := filepath.Join(cwd, "config.json")
	if data, err := os.ReadFile(configPath); err == nil {
		json.Unmarshal(data, &config)
	}

	dbPath = filepath.Join(cwd, config.Storage.LogFile)
}

func initDb() {
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		os.WriteFile(dbPath, []byte("[]"), 0644)
	}
}

func getLogs() []LogEntry {
	initDb()
	data, err := os.ReadFile(dbPath)
	if err != nil {
		return []LogEntry{}
	}
	var logs []LogEntry
	json.Unmarshal(data, &logs)
	return logs
}

func saveLogs(logs []LogEntry) {
	data, _ := json.MarshalIndent(logs, "", "  ")
	os.WriteFile(dbPath, data, 0644)
}

func estimateTokens(text string) int {
	if text == "" || !config.TokenEstimation.Enabled {
		return 0
	}
	return int(math.Ceil(float64(len(text)) / config.TokenEstimation.CharsPerToken))
}

func addLog(entry LogEntry) {
	loadConfig()
	logs := getLogs()

	loc, err := time.LoadLocation(config.Timezone)
	if err != nil {
		loc = time.Local
	}
	now := time.Now().In(loc)
	entry.Timestamp = now.Format("2006-01-02 15:04:05")

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
	saveLogs(logs)

	estStr := ""
	if entry.IsEstimated {
		estStr = " [Tokens Estimated]"
	}
	fmt.Printf("Log added: [%s] (%s)%s\n", entry.Timestamp, entry.Category, estStr)
}

func searchLogs(keyword string) {
	loadConfig()
	logs := getLogs()
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

	fmt.Printf("%-20s | %-12s | Metadata / Q&A\n", "Timestamp", "Category")
	fmt.Println(strings.Repeat("=", 100))

	for _, log := range displayLogs {
		estIn, estOut := "", ""
		if log.IsEstimated {
			estIn = " (est)"
			estOut = " (est)"
		}
		metadata := fmt.Sprintf("[Model: %s | Tokens: In=%d%s, Out=%d%s]", log.Model, log.TokensIn, estIn, log.TokensOut, estOut)
		workspace := fmt.Sprintf("[WS: %s]", log.Workspace)

		if log.Category == "AutoLog" && log.Question != "" && log.Answer != "" {
			fmt.Printf("%-20s | %-12s | %s\n", log.Timestamp, log.Category, metadata)
			if config.Display.ShowWorkspace {
				fmt.Printf("%-20s | %-12s | %s\n", "", "", workspace)
			}
			fmt.Printf("%-20s | %-12s | Q: %s\n", "", "", log.Question)
			fmt.Printf("%-20s | %-12s | A: %s\n", "", "", log.Answer)
		} else {
			fmt.Printf("%-20s | %-12s | %s\n", log.Timestamp, log.Category, log.Message)
			fmt.Printf("%-20s | %-12s | %s\n", "", "", metadata)
		}
		fmt.Println(strings.Repeat("-", 100))
	}
}

func listLogs() {
	loadConfig()
	logs := getLogs()
	if len(logs) == 0 {
		fmt.Println("No logs found.")
		return
	}
	renderLogs(logs)
}

func clearLogs() {
	loadConfig()
	initDb()
	os.WriteFile(dbPath, []byte("[]"), 0644)
	fmt.Println("Logs cleared.")
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
		listLogs()

	case "search":
		if len(os.Args) < 3 {
			fmt.Println("Error: Please provide a keyword to search.")
			return
		}
		searchLogs(os.Args[2])

	case "clear":
		clearLogs()

	default:
		printUsage()
	}
}

func printUsage() {
	fmt.Println("AiKore Activity Tracker (Go)")
	fmt.Println("Usage:")
	fmt.Println("  go run . log \"message\" [-c category]")
	fmt.Println("  go run . auto \"question\" \"answer\" \"model\" \"tokens_in\" \"tokens_out\"")
	fmt.Println("  go run . list")
	fmt.Println("  go run . search \"keyword\"")
	fmt.Println("  go run . clear")
}
