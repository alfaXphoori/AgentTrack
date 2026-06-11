package app

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gofrs/flock"
)

var watcherStartTime = time.Now()

// ---------------------------------------------------------------------------
// VS Code Copilot Watcher
// ---------------------------------------------------------------------------

func getCopilotLoggedCount(stateDir, sessionID string) int {
	stateFile := filepath.Join(stateDir, sessionID+".logged")
	data, err := os.ReadFile(stateFile)
	if err == nil {
		var count int
		fmt.Sscanf(string(data), "%d", &count)
		return count
	}
	return 0
}

func saveCopilotLoggedCount(stateDir, sessionID string, count int) {
	stateFile := filepath.Join(stateDir, sessionID+".logged")
	os.WriteFile(stateFile, []byte(fmt.Sprintf("%d", count)), 0644)
}

func extractCopilotResponse(responseList []interface{}) string {
	var texts []string
	for _, item := range responseList {
		if dict, ok := item.(map[string]interface{}); ok {
			if val, ok := dict["value"].(string); ok {
				texts = append(texts, val)
			} else if val, ok := dict["content"].(string); ok {
				texts = append(texts, val)
			} else if msg, ok := dict["message"].(map[string]interface{}); ok {
				if text, ok := msg["text"].(string); ok && text != "" {
					texts = append(texts, text)
				}
			}
		}
	}
	return strings.Join(texts, "\n")
}

func processCopilotFile(filePath, stateDir string) {
	file, err := os.Open(filePath)
	if err != nil {
		return
	}
	defer file.Close()

	var sessionID string
	var requests []map[string]interface{}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	// First pass
	for scanner.Scan() {
		var data map[string]interface{}
		if err := json.Unmarshal(scanner.Bytes(), &data); err != nil {
			continue
		}

		kind, ok := data["kind"].(float64)
		if !ok {
			continue
		}

		if kind == 0 {
			if v, ok := data["v"].(map[string]interface{}); ok {
				if sessionID == "" {
					if sid, ok := v["sessionId"].(string); ok {
						sessionID = sid
					}
				}
				if reqs, ok := v["requests"].([]interface{}); ok {
					for _, r := range reqs {
						if reqMap, ok := r.(map[string]interface{}); ok {
							reqID := reqMap["requestId"]
							found := false
							for _, existing := range requests {
								if existing["requestId"] == reqID {
									found = true
									break
								}
							}
							if !found {
								requests = append(requests, reqMap)
							}
						}
					}
				}
			}
		} else if kind == 2 {
			if v, ok := data["v"].([]interface{}); ok {
				if k, ok := data["k"].([]interface{}); ok && len(k) == 1 && k[0] == "requests" {
					for _, r := range v {
						if reqMap, ok := r.(map[string]interface{}); ok {
							reqID := reqMap["requestId"]
							found := false
							for _, existing := range requests {
								if existing["requestId"] == reqID {
									found = true
									break
								}
							}
							if !found {
								requests = append(requests, reqMap)
							}
						}
					}
				}
			}
		}
	}

	if sessionID == "" || len(requests) == 0 {
		return
	}

	loggedCount := getCopilotLoggedCount(stateDir, sessionID)
	stateFile := filepath.Join(stateDir, sessionID+".logged")
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		// First time seeing this session file!
		// If the file was not modified since the watcher started, ignore its history.
		if info, err := os.Stat(filePath); err == nil {
			if info.ModTime().Before(watcherStartTime) {
				saveCopilotLoggedCount(stateDir, sessionID, len(requests))
				return
			}
		}
	}

	if len(requests) <= loggedCount {
		return
	}

	// Second pass
	file.Seek(0, 0)
	scanner = bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		var data map[string]interface{}
		if err := json.Unmarshal(scanner.Bytes(), &data); err != nil {
			continue
		}
		if kind, ok := data["kind"].(float64); ok && kind == 2 {
			if v, ok := data["v"].([]interface{}); ok {
				if k, ok := data["k"].([]interface{}); ok && len(k) >= 3 && k[0] == "requests" && k[2] == "response" {
					if idxFloat, ok := k[1].(float64); ok {
						idx := int(idxFloat)
						if idx >= 0 && idx < len(requests) {
							requests[idx]["response"] = v
						}
					}
				}
			}
		}
	}

	atrackBin, _ := os.Executable()

	// Log new requests
	for i := loggedCount; i < len(requests); i++ {
		req := requests[i]
		prompt := ""
		if msg, ok := req["message"].(map[string]interface{}); ok {
			if txt, ok := msg["text"].(string); ok {
				prompt = txt
			}
		}
		if prompt == "" {
			continue
		}

		model := "vscode-copilot"
		if m, ok := req["modelId"].(string); ok && m != "" {
			model = m
		}

		responseText := ""
		if resp, ok := req["response"].([]interface{}); ok {
			responseText = extractCopilotResponse(resp)
		}
		if responseText == "" {
			responseText = "AI Response (Content hidden or pending)"
		}

		summary := responseText
		if len(summary) > 100 {
			summary = summary[:100]
		}

		cmd := exec.Command(atrackBin, "auto", prompt, summary, model, "0", "0", "0", sessionID, "success", "", "vscode-copilot,auto")
		cmd.Run()
		fmt.Printf("✅ Logged VS Code Copilot: %.50s...\n", prompt)
	}

	saveCopilotLoggedCount(stateDir, sessionID, len(requests))
}

func watchCopilot() {
	lockPath := filepath.Join(getAppDir(), "copilot_watcher.lock")
	fileLock := flock.New(lockPath)
	locked, err := fileLock.TryLock()
	if err != nil || !locked {
		return
	}
	defer fileLock.Unlock()

	fmt.Println("🔍 Starting VS Code Copilot Watcher (Go Native)...")
	homeDir, _ := os.UserHomeDir()

	var storagePath string
	switch runtime.GOOS {
	case "darwin":
		storagePath = filepath.Join(homeDir, "Library", "Application Support", "Code", "User", "workspaceStorage")
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(homeDir, "AppData", "Roaming")
		}
		storagePath = filepath.Join(appData, "Code", "User", "workspaceStorage")
	default: // linux and others
		storagePath = filepath.Join(homeDir, ".config", "Code", "User", "workspaceStorage")
	}

	stateDir := filepath.Join(homeDir, ".atrack", "vscode_copilot_state")
	os.MkdirAll(stateDir, 0755)

	for {
		matches, _ := filepath.Glob(filepath.Join(storagePath, "*", "chatSessions", "*.jsonl"))
		for _, f := range matches {
			processCopilotFile(f, stateDir)
		}
		time.Sleep(5 * time.Second)
	}
}

// ---------------------------------------------------------------------------
// Copilot CLI (GitHub Copilot terminal) Watcher
// ---------------------------------------------------------------------------

type copilotCLITurn struct {
	question     string
	answer       string
	model        string
	outputTokens int
}

func getCopilotCLILoggedCount(stateDir, sessionID string) int {
	stateFile := filepath.Join(stateDir, sessionID+".logged")
	data, err := os.ReadFile(stateFile)
	if err == nil {
		var count int
		fmt.Sscanf(string(data), "%d", &count)
		return count
	}
	return 0
}

func saveCopilotCLILoggedCount(stateDir, sessionID string, count int) {
	stateFile := filepath.Join(stateDir, sessionID+".logged")
	os.WriteFile(stateFile, []byte(fmt.Sprintf("%d", count)), 0644)
}

var copilotCLICleanRe = regexp.MustCompile(`(?s)<current_datetime>.*?</current_datetime>\s*|<system_reminder>.*?</system_reminder>\s*`)

// spinnerDefiniteRe matches lines with unambiguous spinner characters
var spinnerDefiniteRe = regexp.MustCompile(`(?m)^[ \t]*[◎○◉].*\n?`)

// spinnerCancelRe matches 'esc cancel' lines
var spinnerCancelRe = regexp.MustCompile(`(?m)^[ \t]*●.*esc cancel.*\n?`)

// spinnerBulletRe removes '●' lines that are just progress words like "Working" or "Work" or backspaces
var spinnerBulletRe = regexp.MustCompile(`(?m)^[ \t]*●[ \t\x08]*(Working|Worki|Work|Wor|Wo|W|ng|g|ad|al|au|a)?[ \t\x08]*\n?`)

// copilotStartupRe matches the Copilot CLI ASCII art startup block through "Check for mistakes."
var copilotStartupRe = regexp.MustCompile(`(?s)[\\╭╮╰╯│█▘▝▔]{2,}.*?Check for mistakes\.\s*`)

func cleanCopilotCLIUserMessage(content string) string {
	return strings.TrimSpace(copilotCLICleanRe.ReplaceAllString(content, ""))
}

func cleanCopilotCLIAssistantMessage(content string) string {
	// Normalize line endings to help match interactive overwrites
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")

	// Remove Copilot CLI startup ASCII art block
	content = copilotStartupRe.ReplaceAllString(content, "")
	
	// Remove spinner/progress lines
	content = spinnerDefiniteRe.ReplaceAllString(content, "")
	content = spinnerCancelRe.ReplaceAllString(content, "")
	content = spinnerBulletRe.ReplaceAllString(content, "")
	
	// Collapse excessive blank lines
	multiBlank := regexp.MustCompile(`\n{3,}`)
	content = multiBlank.ReplaceAllString(content, "\n\n")
	
	return strings.TrimSpace(content)
}

func parseCopilotCLITurns(filePath string) ([]copilotCLITurn, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var turns []copilotCLITurn
	var currentQuestion string
	var bestAnswer string
	var bestModel string
	var bestOutputTokens int
	inTurn := false

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 4*1024*1024), 4*1024*1024)

	for scanner.Scan() {
		var event map[string]interface{}
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			continue
		}
		eventType, _ := event["type"].(string)
		data, _ := event["data"].(map[string]interface{})
		if data == nil {
			continue
		}

		switch eventType {
		case "user.message":
			if inTurn && currentQuestion != "" && bestAnswer != "" {
				turns = append(turns, copilotCLITurn{
					question:     currentQuestion,
					answer:       bestAnswer,
					model:        bestModel,
					outputTokens: bestOutputTokens,
				})
			}
			rawContent, _ := data["content"].(string)
			currentQuestion = cleanCopilotCLIUserMessage(rawContent)
			bestAnswer = ""
			bestModel = ""
			bestOutputTokens = 0
			inTurn = true

		case "assistant.message":
			content, _ := data["content"].(string)
			if content == "" {
				continue
			}
			content = cleanCopilotCLIAssistantMessage(content)
			if content == "" {
				continue
			}
			model, _ := data["model"].(string)
			outputTokens := 0
			if ot, ok := data["outputTokens"].(float64); ok {
				outputTokens = int(ot)
			}
			bestAnswer = content
			if model != "" {
				bestModel = model
			}
			if outputTokens > 0 {
				bestOutputTokens = outputTokens
			}
		}
	}

	if inTurn && currentQuestion != "" && bestAnswer != "" {
		turns = append(turns, copilotCLITurn{
			question:     currentQuestion,
			answer:       bestAnswer,
			model:        bestModel,
			outputTokens: bestOutputTokens,
		})
	}

	return turns, scanner.Err()
}

func processCopilotCLIFile(filePath, sessionID, stateDir string) {
	stateFile := filepath.Join(stateDir, sessionID+".logged")
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		if info, err := os.Stat(filePath); err == nil {
			if info.ModTime().Before(watcherStartTime) {
				turns, err := parseCopilotCLITurns(filePath)
				if err == nil {
					saveCopilotCLILoggedCount(stateDir, sessionID, len(turns))
				}
				return
			}
		}
	}

	turns, err := parseCopilotCLITurns(filePath)
	if err != nil {
		return
	}

	loggedCount := getCopilotCLILoggedCount(stateDir, sessionID)
	if len(turns) <= loggedCount {
		return
	}

	atrackBin, _ := os.Executable()

	for i := loggedCount; i < len(turns); i++ {
		turn := turns[i]
		if turn.question == "" {
			continue
		}

		model := turn.model
		if model == "" {
			model = "copilot-cli"
		}

		summary := turn.answer
		if len(summary) > 100 {
			summary = summary[:100]
		}

		tokensOut := fmt.Sprintf("%d", turn.outputTokens)
		cmd := exec.Command(atrackBin, "auto", turn.question, summary, model, "0", tokensOut, "0", sessionID, "success", "", "copilot-cli,auto")
		cmd.Run()
		fmt.Printf("✅ Logged Copilot CLI: %.50s...\n", turn.question)
	}

	saveCopilotCLILoggedCount(stateDir, sessionID, len(turns))
}

func watchCopilotCLI() {
	lockPath := filepath.Join(getAppDir(), "copilot_cli_watcher.lock")
	fileLock := flock.New(lockPath)
	locked, err := fileLock.TryLock()
	if err != nil || !locked {
		return
	}
	defer fileLock.Unlock()

	fmt.Println("🔍 Starting Copilot CLI Watcher...")
	homeDir, _ := os.UserHomeDir()
	storagePath := filepath.Join(homeDir, ".copilot", "session-state")
	stateDir := filepath.Join(homeDir, ".atrack", "copilot_cli_state")
	os.MkdirAll(stateDir, 0755)

	for {
		matches, _ := filepath.Glob(filepath.Join(storagePath, "*", "events.jsonl"))
		for _, f := range matches {
			sessionID := filepath.Base(filepath.Dir(f))
			processCopilotCLIFile(f, sessionID, stateDir)
		}
		time.Sleep(5 * time.Second)
	}
}

// ---------------------------------------------------------------------------
// Gemini Detect Model
// ---------------------------------------------------------------------------

func findGeminiModel(v interface{}) string {
	if dict, ok := v.(map[string]interface{}); ok {
		for k, val := range dict {
			if k == "model" {
				if s, ok := val.(string); ok && strings.Contains(strings.ToLower(s), "gemini") {
					return s
				}
			}
			if m := findGeminiModel(val); m != "" {
				return m
			}
		}
	} else if list, ok := v.([]interface{}); ok {
		for _, item := range list {
			if m := findGeminiModel(item); m != "" {
				return m
			}
		}
	}
	return ""
}

func runDetectGeminiModel(cwd, homeDir string) string {
	tmpBase := filepath.Join(homeDir, ".gemini", "tmp")

	dirs, _ := os.ReadDir(tmpBase)
	targetDir := ""
	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}
		pr := filepath.Join(tmpBase, d.Name(), ".project_root")
		if data, err := os.ReadFile(pr); err == nil {
			if strings.EqualFold(strings.TrimSpace(string(data)), cwd) {
				targetDir = filepath.Join(tmpBase, d.Name())
				break
			}
		}
	}

	if targetDir == "" {
		return ""
	}

	sessions, _ := filepath.Glob(filepath.Join(targetDir, "chats", "session-*.jsonl"))
	sort.Slice(sessions, func(i, j int) bool {
		fi, _ := os.Stat(sessions[i])
		fj, _ := os.Stat(sessions[j])
		if fi != nil && fj != nil {
			return fi.ModTime().Before(fj.ModTime())
		}
		return sessions[i] < sessions[j]
	})

	for i := len(sessions) - 1; i >= 0; i-- {
		file, err := os.Open(sessions[i])
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(file)
		scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
		var model string
		for scanner.Scan() {
			var data interface{}
			if err := json.Unmarshal(scanner.Bytes(), &data); err == nil {
				if m := findGeminiModel(data); m != "" {
					model = m
				}
			}
		}
		file.Close()
		if model != "" {
			return model
		}
	}
	return ""
}

func detectGeminiModel() {
	cwd, _ := os.Getwd()
	homeDir, _ := os.UserHomeDir()

	model := runDetectGeminiModel(cwd, homeDir)
	if model != "" {
		fmt.Println(model)
		os.Exit(0)
	}
	os.Exit(1)
}

// ---------------------------------------------------------------------------
// Gemini Watcher
// ---------------------------------------------------------------------------

type geminiTurn struct {
	Type  string
	Text  string
	Model string
	Ts    time.Time
	Tools []string
}

func parseIso(ts string) time.Time {
	ts = strings.Replace(ts, "Z", "+00:00", 1)
	t, _ := time.Parse(time.RFC3339Nano, ts)
	return t
}

func PrimeWatchers() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	// 1. Prime Gemini
	stateDir := filepath.Join(home, ".atrack", "watch_state")
	os.MkdirAll(stateDir, 0755)

	geminiTmp := filepath.Join(home, ".gemini", "tmp")
	entries, _ := os.ReadDir(geminiTmp)
	for _, d := range entries {
		if !d.IsDir() {
			continue
		}
		chatsDir := filepath.Join(geminiTmp, d.Name(), "chats")
		sessions, _ := filepath.Glob(filepath.Join(chatsDir, "session-*.jsonl"))
		for _, s := range sessions {
			pairs := countGeminiPairs(s)
			stateFile := filepath.Join(stateDir, filepath.Base(s)+".logged")
			os.WriteFile(stateFile, []byte(fmt.Sprintf("%d", pairs)), 0644)
		}
	}

	geminiBrain := filepath.Join(home, ".gemini", "antigravity-cli", "brain")
	brainEntries, _ := os.ReadDir(geminiBrain)
	for _, d := range brainEntries {
		if !d.IsDir() {
			continue
		}
		tPath := filepath.Join(geminiBrain, d.Name(), ".system_generated", "logs", "transcript.jsonl")
		if _, err := os.Stat(tPath); err == nil {
			pairs := countGeminiPairs(tPath)
			dirParts := strings.Split(tPath, string(os.PathSeparator))
			baseName := "transcript.jsonl"
			if len(dirParts) >= 4 {
				baseName = "transcript_" + dirParts[len(dirParts)-4] + ".jsonl"
			}
			stateFile := filepath.Join(stateDir, baseName+".logged")
			os.WriteFile(stateFile, []byte(fmt.Sprintf("%d", pairs)), 0644)
		}
	}

	// 2. Prime Copilot
	var storagePath string
	switch runtime.GOOS {
	case "darwin":
		storagePath = filepath.Join(home, "Library", "Application Support", "Code", "User", "workspaceStorage")
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		storagePath = filepath.Join(appData, "Code", "User", "workspaceStorage")
	default:
		storagePath = filepath.Join(home, ".config", "Code", "User", "workspaceStorage")
	}

	copilotStateDir := filepath.Join(home, ".atrack", "vscode_copilot_state")
	os.MkdirAll(copilotStateDir, 0755)

	matches, _ := filepath.Glob(filepath.Join(storagePath, "*", "chatSessions", "*.jsonl"))
	for _, f := range matches {
		sessionID, requests := scanCopilotFile(f)
		if sessionID != "" {
			saveCopilotLoggedCount(copilotStateDir, sessionID, len(requests))
		}
	}

	// 3. Prime Copilot CLI
	copilotCLIStateDir := filepath.Join(home, ".atrack", "copilot_cli_state")
	os.MkdirAll(copilotCLIStateDir, 0755)

	copilotCLIStorage := filepath.Join(home, ".copilot", "session-state")
	copilotCLIMatches, _ := filepath.Glob(filepath.Join(copilotCLIStorage, "*", "events.jsonl"))
	for _, f := range copilotCLIMatches {
		sessionID := filepath.Base(filepath.Dir(f))
		if info, err := os.Stat(f); err == nil {
			if time.Since(info.ModTime()) < 2*time.Minute {
				continue
			}
		}
		turns, err := parseCopilotCLITurns(f)
		if err == nil {
			saveCopilotCLILoggedCount(copilotCLIStateDir, sessionID, len(turns))
		}
	}

	// 4. Prime Claude CLI
	claudeCLIStateDir := filepath.Join(home, ".atrack", "claude_cli_state")
	os.MkdirAll(claudeCLIStateDir, 0755)

	claudeCLIStorage := filepath.Join(home, ".claude", "projects")
	claudeCLIMatches, _ := filepath.Glob(filepath.Join(claudeCLIStorage, "*", "*.jsonl"))
	for _, f := range claudeCLIMatches {
		sessionID := strings.TrimSuffix(filepath.Base(f), ".jsonl")
		if info, err := os.Stat(f); err == nil {
			if time.Since(info.ModTime()) < 2*time.Minute {
				continue
			}
		}
		turns, err := parseClaudeTurns(f, true)
		if err == nil {
			saveClaudeLoggedCount(claudeCLIStateDir, sessionID, len(turns))
		}
	}

	// 5. Prime Codex CLI
	codexCLIStateDir := filepath.Join(home, ".atrack", "codex_cli_state")
	os.MkdirAll(codexCLIStateDir, 0755)

	codexCLIStorage := filepath.Join(home, ".codex", "sessions")
	codexCLIMatches, _ := filepath.Glob(filepath.Join(codexCLIStorage, "*", "*", "*", "rollout-*.jsonl"))
	for _, f := range codexCLIMatches {
		sessionID := strings.TrimSuffix(filepath.Base(f), ".jsonl")
		if info, err := os.Stat(f); err == nil {
			if time.Since(info.ModTime()) < 2*time.Minute {
				continue
			}
		}
		turns, err := parseCodexTurns(f, true)
		if err == nil {
			saveCodexLoggedCount(codexCLIStateDir, sessionID, len(turns))
		}
	}

	fmt.Println("✅ All watcher states primed (ignoring existing history).")
}

func scanCopilotFile(filePath string) (string, []map[string]interface{}) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", nil
	}
	defer file.Close()

	var sessionID string
	var requests []map[string]interface{}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		var data map[string]interface{}
		if err := json.Unmarshal(scanner.Bytes(), &data); err != nil {
			continue
		}
		kind, _ := data["kind"].(float64)
		if kind == 0 {
			if v, ok := data["v"].(map[string]interface{}); ok {
				if sessionID == "" {
					sessionID, _ = v["sessionId"].(string)
				}
				if reqs, ok := v["requests"].([]interface{}); ok {
					for _, r := range reqs {
						if reqMap, ok := r.(map[string]interface{}); ok {
							reqID := reqMap["requestId"]
							found := false
							for _, existing := range requests {
								if existing["requestId"] == reqID {
									found = true
									break
								}
							}
							if !found {
								requests = append(requests, reqMap)
							}
						}
					}
				}
			}
		} else if kind == 2 {
			if v, ok := data["v"].([]interface{}); ok {
				if k, ok := data["k"].([]interface{}); ok && len(k) == 1 && k[0] == "requests" {
					for _, r := range v {
						if reqMap, ok := r.(map[string]interface{}); ok {
							reqID := reqMap["requestId"]
							found := false
							for _, existing := range requests {
								if existing["requestId"] == reqID {
									found = true
									break
								}
							}
							if !found {
								requests = append(requests, reqMap)
							}
						}
					}
				}
			}
		}
	}
	return sessionID, requests
}

func countGeminiPairs(filePath string) int {
	file, err := os.Open(filePath)
	if err != nil {
		return 0
	}
	defer file.Close()

	var turns []geminiTurn
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		var d map[string]interface{}
		if err := json.Unmarshal(scanner.Bytes(), &d); err != nil {
			continue
		}
		typ, _ := d["type"].(string)
		source, _ := d["source"].(string)
		if source == "USER_EXPLICIT" {
			typ = "user"
		} else if source == "MODEL" {
			typ = "gemini"
		}
		if typ == "user" || typ == "gemini" {
			text := ""
			if contentStr, ok := d["content"].(string); ok {
				text = contentStr
			} else if contentArr, ok := d["content"].([]interface{}); ok {
				for _, c := range contentArr {
					if cDict, ok := c.(map[string]interface{}); ok {
						if t, ok := cDict["text"].(string); ok {
							text += t
						}
					}
				}
			}
			var tools []string
			if toolCalls, ok := d["toolCalls"].([]interface{}); ok {
				for _, call := range toolCalls {
					if c, ok := call.(map[string]interface{}); ok {
						if name, ok := c["name"].(string); ok {
							tools = append(tools, name)
						}
					}
				}
			}
			if strings.TrimSpace(text) != "" || len(tools) > 0 {
				turns = append(turns, geminiTurn{Type: typ, Text: text, Tools: tools})
			}
		}
	}

	pairs := 0
	i := 0
	for i < len(turns) {
		if turns[i].Type == "user" {
			j := i + 1
			foundGemini := false
			for j < len(turns) && turns[j].Type != "user" {
				if turns[j].Type == "gemini" {
					foundGemini = true
				}
				j++
			}
			if foundGemini {
				pairs++
				i = j
			} else {
				i++
			}
		} else {
			i++
		}
	}
	return pairs
}

func processGeminiSession(filePath, stateDir string) {
	file, err := os.Open(filePath)
	if err != nil {
		return
	}
	defer file.Close()

	baseName := filepath.Base(filePath)
	if baseName == "transcript.jsonl" {
		dirParts := strings.Split(filePath, string(os.PathSeparator))
		if len(dirParts) >= 4 {
			baseName = "transcript_" + dirParts[len(dirParts)-4] + ".jsonl"
		}
	}
	stateFile := filepath.Join(stateDir, baseName+".logged")
	loggedPairs := 0
	if data, err := os.ReadFile(stateFile); err == nil {
		fmt.Sscanf(string(data), "%d", &loggedPairs)
	} else if os.IsNotExist(err) {
		// First time seeing this session file!
		// If the file is old (modified > 1 min ago), ignore its history.
		if info, err := os.Stat(filePath); err == nil {
			if time.Since(info.ModTime()) > 1*time.Minute {
				// We need to count current pairs first to know what to ignore
				currentPairs := countGeminiPairs(filePath)
				os.WriteFile(stateFile, []byte(fmt.Sprintf("%d", currentPairs)), 0644)
				return
			}
		}
	}

	var turns []geminiTurn
	sessionID := ""
	agyModel := "agy"

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		var d map[string]interface{}
		if err := json.Unmarshal(scanner.Bytes(), &d); err != nil {
			continue
		}
		if sid, ok := d["sessionId"].(string); ok {
			sessionID = sid
			continue
		}

		typ, _ := d["type"].(string)
		model, _ := d["model"].(string)
		tsStr, _ := d["timestamp"].(string)

		source, _ := d["source"].(string)
		if source == "USER_EXPLICIT" {
			typ = "user"
		} else if source == "MODEL" {
			typ = "gemini"
		}
		if model == "" && source == "MODEL" {
			model = agyModel
		}
		if tsStr == "" {
			if ca, ok := d["created_at"].(string); ok {
				tsStr = ca
			}
		}

		var tools []string

		if toolCalls, ok := d["toolCalls"].([]interface{}); ok {
			for _, call := range toolCalls {
				if c, ok := call.(map[string]interface{}); ok {
					if name, ok := c["name"].(string); ok {
						tools = append(tools, name)
					}
				}
			}
		}
		if toolCalls, ok := d["tool_calls"].([]interface{}); ok {
			for _, call := range toolCalls {
				if c, ok := call.(map[string]interface{}); ok {
					if name, ok := c["name"].(string); ok {
						tools = append(tools, name)
					}
				}
			}
		}

		if typ == "user" || typ == "gemini" {
			text := ""
			if contentStr, ok := d["content"].(string); ok {
				text = contentStr
			} else if contentArr, ok := d["content"].([]interface{}); ok {
				for _, c := range contentArr {
					if cDict, ok := c.(map[string]interface{}); ok {
						if t, ok := cDict["text"].(string); ok {
							text += t
						}
					}
				}
			}

			// Do not treat tool execution outputs as the actual AI conversation answer
			if typ == "gemini" && strings.HasPrefix(strings.TrimSpace(text), "Created At:") {
				text = ""
			}


			if strings.TrimSpace(text) != "" || len(tools) > 0 {
				if typ == "user" {
					if strings.Contains(text, "USER_SETTINGS_CHANGE") {
						re := regexp.MustCompile("The user changed setting `Model Selection` from .*? to ([A-Za-z0-9 .()]+)\\. No need to comment")
						matches := re.FindStringSubmatch(text)
						if len(matches) > 1 {
							agyModel = strings.TrimSpace(matches[1])
						}
					}
					// Clean up XML tags injected by Antigravity CLI
					if strings.Contains(text, "<USER_REQUEST>") {
						reqRe := regexp.MustCompile(`(?s)<USER_REQUEST>\n*(.*?)\n*</USER_REQUEST>`)
						matches := reqRe.FindStringSubmatch(text)
						if len(matches) > 1 {
							text = matches[1]
						}
					}
				}
				turns = append(turns, geminiTurn{
					Type:  typ,
					Text:  strings.TrimSpace(text),
					Model: model,
					Ts:    parseIso(tsStr),
					Tools: tools,
				})
			}
		}
	}

	type pair struct {
		Question string
		Answer   string
		Model    string
		Duration float64
		Tools    []string
	}

	var pairs []pair
	i := 0
	lastModel := "gemini"

	for i < len(turns) {
		if turns[i].Type == "user" {
			qText := turns[i].Text
			qTs := turns[i].Ts
			
			var aText strings.Builder
			var aTools []string
			var aTs time.Time
			var aModel string

			// Look ahead for all Gemini turns until the next User turn
			j := i + 1
			for j < len(turns) && turns[j].Type != "user" {
				if turns[j].Type == "gemini" {
					if turns[j].Text != "" {
						if aText.Len() > 0 {
							aText.WriteString("\n")
						}
						aText.WriteString(turns[j].Text)
					}
					aTools = append(aTools, turns[j].Tools...)
					if aTs.IsZero() {
						aTs = turns[j].Ts
					}
					if turns[j].Model != "" {
						aModel = turns[j].Model
					}
				}
				j++
			}

			// Only log if we found at least one Gemini turn
			if aTs.IsZero() == false || aText.Len() > 0 || len(aTools) > 0 {
				// Optimization: If this is the very last pair in an active session 
				// and there's no text answer yet, skip it for now and wait for the AI to finish.
				if j == len(turns) && aText.Len() == 0 {
					if info, err := os.Stat(filePath); err == nil {
						if time.Since(info.ModTime()) < 15*time.Second {
							break // Exit the pair loop, don't log this incomplete one yet
						}
					}
				}

				model := aModel
				if model == "" {
					model = lastModel
				}
				var duration float64
				if !qTs.IsZero() && !aTs.IsZero() {
					duration = aTs.Sub(qTs).Seconds()
				}
				pairs = append(pairs, pair{
					Question: qText,
					Answer:   aText.String(),
					Model:    model,
					Duration: duration,
					Tools:    aTools,
				})
				if aModel != "" {
					lastModel = aModel
				}
				i = j // Advance to the next User turn or end
			} else {
				i++
			}
		} else {
			if turns[i].Model != "" {
				lastModel = turns[i].Model
			}
			i++
		}
	}

	// atrackBin removed

	for idx := loggedPairs; idx < len(pairs); idx++ {
		p := pairs[idx]
		dur := fmt.Sprintf("%.2f", p.Duration)
		toolsStr := strings.Join(p.Tools, ",")
		ti := fmt.Sprintf("%d", max(1, len(p.Question)/4))
		to := fmt.Sprintf("%d", max(1, len(p.Answer)/4))

		summary := strings.TrimSpace(p.Answer)
		if lines := strings.Split(summary, "\n"); len(lines) > 0 {
			for _, line := range lines {
				if trimmed := strings.TrimSpace(line); trimmed != "" {
					summary = trimmed
					break
				}
			}
		}
		if len(summary) > 150 {
			summary = summary[:150]
		}

		tIn, _ := strconv.Atoi(ti)
		tOut, _ := strconv.Atoi(to)
		durFloat, _ := strconv.ParseFloat(dur, 64)
		var toolsUsed []string
		if toolsStr != "" {
			toolsUsed = strings.Split(toolsStr, ",")
		}

		entry := LogEntry{
			Category:    "AutoLog",
			Question:    p.Question,
			Answer:      summary,
			Model:       p.Model,
			TokensIn:    tIn,
			TokensOut:   tOut,
			IsEstimated: false,
			Duration:    durFloat,
			SessionID:   sessionID,
			Status:      "success",
			ToolsUsed:   toolsUsed,
			Tags:        []string{"gemini-cli"},
		}

		loadConfig()
		if cost, ok := calculateLogCost(entry); ok {
			entry.Cost = cost
		}
		addLog(entry)
		icon := "✅"

		qDisp := p.Question
		if len(qDisp) > 60 {
			qDisp = qDisp[:60]
		}
		fmt.Printf("%s [%s] [%ss] %s\n", icon, p.Model, dur, qDisp)
	}

	os.WriteFile(stateFile, []byte(fmt.Sprintf("%d", len(pairs))), 0644)
}

func watchGemini() {
	lockPath := filepath.Join(getAppDir(), "gemini_watcher.lock")
	fileLock := flock.New(lockPath)
	locked, err := fileLock.TryLock()
	if err != nil || !locked {
		return
	}
	defer fileLock.Unlock()

	fmt.Println("🔍 AgentTrack Gemini Watcher started (Go Native)")
	homeDir, _ := os.UserHomeDir()
	geminiTmp := filepath.Join(homeDir, ".gemini", "tmp")
	geminiBrain := filepath.Join(homeDir, ".gemini", "antigravity-cli", "brain")
	stateDir := filepath.Join(homeDir, ".atrack", "watch_state")
	os.MkdirAll(stateDir, 0755)

	if _, err := os.Stat(geminiTmp); os.IsNotExist(err) {
		if _, err2 := os.Stat(geminiBrain); os.IsNotExist(err2) {
			fmt.Println("❌ ~/.gemini/tmp and ~/.gemini/antigravity-cli not found. Open gemini in any project first.")
			os.Exit(1)
		}
	}

	for {
		var allSessions []string

		if dirs, err := os.ReadDir(geminiTmp); err == nil {
			for _, d := range dirs {
				if !d.IsDir() {
					continue
				}
				chatsDir := filepath.Join(geminiTmp, d.Name(), "chats")
				sessions, _ := filepath.Glob(filepath.Join(chatsDir, "session-*.jsonl"))
				allSessions = append(allSessions, sessions...)
			}
		}

		if brainDirs, err := os.ReadDir(geminiBrain); err == nil {
			for _, d := range brainDirs {
				if !d.IsDir() {
					continue
				}
				tPath := filepath.Join(geminiBrain, d.Name(), ".system_generated", "logs", "transcript.jsonl")
				if _, err := os.Stat(tPath); err == nil {
					allSessions = append(allSessions, tPath)
				}
			}
		}

		for _, s := range allSessions {
			processGeminiSession(s, stateDir)
		}
		
		time.Sleep(2 * time.Second)
	}
}

func countAiderPairs(filePath string) int {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return 0
	}
	content := string(data)
	return strings.Count(content, "> ASSISTANT:")
}

func processAiderFile(filePath string, stateDir string, workspace string) {
	stateFile := filepath.Join(stateDir, fmt.Sprintf("%x.logged", filePath))
	
	loggedPairs := 0
	if data, err := os.ReadFile(stateFile); err == nil {
		fmt.Sscanf(string(data), "%d", &loggedPairs)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return
	}
	content := string(data)
	
	parts := strings.Split(content, "> USER:")
	var pairs []struct{ Q, A string }
	
	for i := 1; i < len(parts); i++ {
		p := parts[i]
		idx := strings.Index(p, "> ASSISTANT:")
		if idx == -1 {
			continue // No assistant reply yet
		}
		q := strings.TrimSpace(p[:idx])
		a := strings.TrimSpace(p[idx+len("> ASSISTANT:"):])
		
		// strip out the next chat session if there is one
		if chatIdx := strings.Index(a, "# aider chat started at"); chatIdx != -1 {
			a = strings.TrimSpace(a[:chatIdx])
		}
		
		pairs = append(pairs, struct{ Q, A string }{Q: q, A: a})
	}
	
	for idx := loggedPairs; idx < len(pairs); idx++ {
		p := pairs[idx]
		
		ti := estimateTokens(p.Q)
		to := estimateTokens(p.A)
		
		summary := strings.TrimSpace(p.A)
		if lines := strings.Split(summary, "\n"); len(lines) > 0 {
			summary = strings.TrimSpace(lines[0])
		}
		if len(summary) > 150 {
			summary = summary[:150]
		}
		
		entry := LogEntry{
			Category:    "AutoLog",
			Question:    p.Q,
			Answer:      summary,
			Model:       "aider", // We don't easily know the exact model from the md without more parsing
			TokensIn:    ti,
			TokensOut:   to,
			IsEstimated: true,
			Workspace:   workspace,
			Status:      "success",
			Tags:        []string{"aider"},
		}
		
		loadConfig()
		if cost, ok := calculateLogCost(entry); ok {
			entry.Cost = cost
		}
		addLog(entry)
		fmt.Printf("✅ [aider] %s\n", entry.Question)
	}
	
	os.WriteFile(stateFile, []byte(fmt.Sprintf("%d", len(pairs))), 0644)
}

func watchAider() {
	lockPath := filepath.Join(getAppDir(), "aider_watcher.lock")
	fileLock := flock.New(lockPath)
	locked, err := fileLock.TryLock()
	if err != nil || !locked {
		return
	}
	defer fileLock.Unlock()

	fmt.Println("🔍 Starting Aider Watcher (Go Native)...")
	homeDir, _ := os.UserHomeDir()
	stateDir := filepath.Join(homeDir, ".atrack", "aider_state")
	os.MkdirAll(stateDir, 0755)

	for {
		// Collect unique workspaces from existing logs
		logs := getLogsFromAllFiles()
		workspaces := make(map[string]bool)
		for _, l := range logs {
			if l.Workspace != "" {
				workspaces[l.Workspace] = true
			}
		}
		// Also add current working directory
		cwd, _ := os.Getwd()
		workspaces[cwd] = true

		for ws := range workspaces {
			historyFile := filepath.Join(ws, ".aider.chat.history.md")
			if _, err := os.Stat(historyFile); err == nil {
				processAiderFile(historyFile, stateDir, ws)
			}
		}
		time.Sleep(5 * time.Second)
	}
}

// ---------------------------------------------------------------------------
// Claude CLI Watcher
// ---------------------------------------------------------------------------

type claudeTurn struct {
	question     string
	answer       string
	model        string
	outputTokens int
}

func getClaudeLoggedCount(stateDir, sessionID string) int {
	stateFile := filepath.Join(stateDir, sessionID+".logged")
	data, err := os.ReadFile(stateFile)
	if err == nil {
		var count int
		fmt.Sscanf(string(data), "%d", &count)
		return count
	}
	return 0
}

func saveClaudeLoggedCount(stateDir, sessionID string, count int) {
	stateFile := filepath.Join(stateDir, sessionID+".logged")
	os.WriteFile(stateFile, []byte(fmt.Sprintf("%d", count)), 0644)
}

func parseClaudeTurns(filePath string, isOld bool) ([]claudeTurn, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var turns []claudeTurn
	var currentQuestion string
	var currentAnswer []string
	var currentModel string
	var currentTokens int
	hasQuestion := false
	turnFinished := false

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		var event map[string]interface{}
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			continue
		}
		eventType, _ := event["type"].(string)
		if eventType == "user" {
			msg, _ := event["message"].(map[string]interface{})
			if msg != nil {
				content, ok := msg["content"].(string)
				if ok {
					if !isSystemClaudePrompt(content) {
						if hasQuestion && (turnFinished || (isOld && len(currentAnswer) > 0)) && (len(currentAnswer) > 0 || currentTokens > 0) {
							turns = append(turns, claudeTurn{
								question:     currentQuestion,
								answer:       strings.Join(currentAnswer, "\n"),
								model:        currentModel,
								outputTokens: currentTokens,
							})
						}
						currentQuestion = content
						currentAnswer = []string{}
						currentModel = ""
						currentTokens = 0
						hasQuestion = true
						turnFinished = false
					}
				}
			}
		} else if eventType == "assistant" {
			msg, _ := event["message"].(map[string]interface{})
			if msg != nil {
				if model, ok := msg["model"].(string); ok && model != "" {
					currentModel = model
				}
				if usage, ok := msg["usage"].(map[string]interface{}); ok {
					if ot, ok := usage["output_tokens"].(float64); ok {
						currentTokens = int(ot)
					}
				}
				if content, ok := msg["content"].([]interface{}); ok {
					for _, block := range content {
						if blockMap, ok := block.(map[string]interface{}); ok {
							if bType, ok := blockMap["type"].(string); ok && bType == "text" {
								if text, ok := blockMap["text"].(string); ok && text != "" {
									currentAnswer = append(currentAnswer, text)
								}
							}
						}
					}
				}
			}
		} else if eventType == "system" {
			subtype, _ := event["subtype"].(string)
			if subtype == "turn_duration" {
				turnFinished = true
			}
		}
	}

	if hasQuestion && (turnFinished || (isOld && len(currentAnswer) > 0)) && (len(currentAnswer) > 0 || currentTokens > 0) {
		turns = append(turns, claudeTurn{
			question:     currentQuestion,
			answer:       strings.Join(currentAnswer, "\n"),
			model:        currentModel,
			outputTokens: currentTokens,
		})
	}

	return turns, scanner.Err()
}

func isSystemClaudePrompt(content string) bool {
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "<local-command-") ||
		strings.HasPrefix(content, "<task-") ||
		strings.HasPrefix(content, "<bash-") ||
		strings.HasPrefix(content, "<command-") {
		return true
	}
	return false
}

func processClaudeFile(filePath, sessionID, stateDir string) {
	info, err := os.Stat(filePath)
	if err != nil {
		return
	}
	isOld := time.Since(info.ModTime()) > 2*time.Minute

	stateFile := filepath.Join(stateDir, sessionID+".logged")
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		if info.ModTime().Before(watcherStartTime) {
			turns, err := parseClaudeTurns(filePath, true)
			if err == nil {
				saveClaudeLoggedCount(stateDir, sessionID, len(turns))
			}
			return
		}
	}

	turns, err := parseClaudeTurns(filePath, isOld)
	if err != nil {
		return
	}

	loggedCount := getClaudeLoggedCount(stateDir, sessionID)
	if len(turns) <= loggedCount {
		return
	}

	for i := loggedCount; i < len(turns); i++ {
		turn := turns[i]
		if turn.question == "" {
			continue
		}

		model := turn.model
		if model == "" {
			model = "claude-cli"
		}

		summary := turn.answer
		if len(summary) > 100 {
			summary = summary[:100]
		}

		tIn := estimateTokens(turn.question)

		entry := LogEntry{
			Category:    "AutoLog",
			Question:    turn.question,
			Answer:      summary,
			Model:       model,
			TokensIn:    tIn,
			TokensOut:   turn.outputTokens,
			IsEstimated: false,
			SessionID:   sessionID,
			Status:      "success",
			Tags:        []string{"claude-cli"},
		}

		loadConfig()
		if cost, ok := calculateLogCost(entry); ok {
			entry.Cost = cost
		}
		addLog(entry)
		fmt.Printf("✅ Logged Claude CLI: %.50s...\n", turn.question)
	}

	saveClaudeLoggedCount(stateDir, sessionID, len(turns))
}

func watchClaude() {
	lockPath := filepath.Join(getAppDir(), "claude_cli_watcher.lock")
	fileLock := flock.New(lockPath)
	locked, err := fileLock.TryLock()
	if err != nil || !locked {
		return
	}
	defer fileLock.Unlock()

	fmt.Println("🔍 Starting Claude CLI Watcher...")
	homeDir, _ := os.UserHomeDir()
	storagePath := filepath.Join(homeDir, ".claude", "projects")
	stateDir := filepath.Join(homeDir, ".atrack", "claude_cli_state")
	os.MkdirAll(stateDir, 0755)

	for {
		matches, _ := filepath.Glob(filepath.Join(storagePath, "*", "*.jsonl"))
		for _, f := range matches {
			sessionID := strings.TrimSuffix(filepath.Base(f), ".jsonl")
			processClaudeFile(f, sessionID, stateDir)
		}
		time.Sleep(5 * time.Second)
	}
}

// ---------------------------------------------------------------------------
// Codex CLI Watcher
// ---------------------------------------------------------------------------

type codexTurn struct {
	question     string
	answer       string
	model        string
	outputTokens int
}

func getCodexLoggedCount(stateDir, sessionID string) int {
	stateFile := filepath.Join(stateDir, sessionID+".logged")
	data, err := os.ReadFile(stateFile)
	if err != nil {
		return 0
	}
	var count int
	fmt.Sscanf(string(data), "%d", &count)
	return count
}

func saveCodexLoggedCount(stateDir, sessionID string, count int) {
	stateFile := filepath.Join(stateDir, sessionID+".logged")
	_ = os.WriteFile(stateFile, []byte(fmt.Sprintf("%d", count)), 0644)
}

func parseCodexTurns(filePath string, isOld bool) ([]codexTurn, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var turns []codexTurn
	var currentQuestion string
	var currentAnswer []string
	var currentModel string
	var currentTokens int
	hasQuestion := false
	turnFinished := false

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		var event map[string]interface{}
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			continue
		}
		eventType, _ := event["type"].(string)
		if eventType == "turn_context" {
			payload, _ := event["payload"].(map[string]interface{})
			if payload != nil {
				if m, ok := payload["model"].(string); ok && m != "" {
					currentModel = m
				}
			}
		} else if eventType == "event_msg" {
			payload, _ := event["payload"].(map[string]interface{})
			if payload != nil {
				payloadType, _ := payload["type"].(string)
				if payloadType == "user_message" {
					if hasQuestion && (turnFinished || (isOld && len(currentAnswer) > 0)) && (len(currentAnswer) > 0 || currentTokens > 0) {
						turns = append(turns, codexTurn{
							question:     currentQuestion,
							answer:       strings.Join(currentAnswer, "\n"),
							model:        currentModel,
							outputTokens: currentTokens,
						})
					}
					msg, _ := payload["message"].(string)
					currentQuestion = msg
					currentAnswer = []string{}
					currentTokens = 0
					hasQuestion = true
					turnFinished = false
				} else if payloadType == "agent_message" {
					msg, _ := payload["message"].(string)
					if msg != "" {
						currentAnswer = append(currentAnswer, msg)
					}
				} else if payloadType == "token_count" {
					if info, ok := payload["info"].(map[string]interface{}); ok {
						if usage, ok := info["total_token_usage"].(map[string]interface{}); ok {
							if ot, ok := usage["output_tokens"].(float64); ok {
								currentTokens = int(ot)
							}
						}
					}
				} else if payloadType == "task_complete" {
					turnFinished = true
				}
			}
		}
	}

	if hasQuestion && (turnFinished || (isOld && len(currentAnswer) > 0)) && (len(currentAnswer) > 0 || currentTokens > 0) {
		turns = append(turns, codexTurn{
			question:     currentQuestion,
			answer:       strings.Join(currentAnswer, "\n"),
			model:        currentModel,
			outputTokens: currentTokens,
		})
	}

	return turns, scanner.Err()
}

func processCodexFile(filePath, sessionID, stateDir string) {
	info, err := os.Stat(filePath)
	if err != nil {
		return
	}
	isOld := time.Since(info.ModTime()) > 2*time.Minute

	stateFile := filepath.Join(stateDir, sessionID+".logged")
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		if info.ModTime().Before(watcherStartTime) {
			turns, err := parseCodexTurns(filePath, true)
			if err == nil {
				saveCodexLoggedCount(stateDir, sessionID, len(turns))
			}
			return
		}
	}

	turns, err := parseCodexTurns(filePath, isOld)
	if err != nil {
		return
	}

	loggedCount := getCodexLoggedCount(stateDir, sessionID)
	if len(turns) <= loggedCount {
		return
	}

	for i := loggedCount; i < len(turns); i++ {
		turn := turns[i]
		if turn.question == "" {
			continue
		}

		model := turn.model
		if model == "" {
			model = "codex-cli"
		}

		summary := turn.answer
		if len(summary) > 100 {
			summary = summary[:100]
		}

		tIn := estimateTokens(turn.question)

		entry := LogEntry{
			Category:    "AutoLog",
			Question:    turn.question,
			Answer:      summary,
			Model:       model,
			TokensIn:    tIn,
			TokensOut:   turn.outputTokens,
			IsEstimated: false,
			SessionID:   sessionID,
			Status:      "success",
			Tags:        []string{"codex-cli"},
		}

		loadConfig()
		if cost, ok := calculateLogCost(entry); ok {
			entry.Cost = cost
		}
		addLog(entry)
		fmt.Printf("✅ Logged Codex CLI: %.50s...\n", turn.question)
	}

	saveCodexLoggedCount(stateDir, sessionID, len(turns))
}

func watchCodex() {
	lockPath := filepath.Join(getAppDir(), "codex_cli_watcher.lock")
	fileLock := flock.New(lockPath)
	locked, err := fileLock.TryLock()
	if err != nil || !locked {
		return
	}
	defer fileLock.Unlock()

	fmt.Println("🔍 Starting Codex CLI Watcher...")
	homeDir, _ := os.UserHomeDir()
	storagePath := filepath.Join(homeDir, ".codex", "sessions")
	stateDir := filepath.Join(homeDir, ".atrack", "codex_cli_state")
	os.MkdirAll(stateDir, 0755)

	for {
		matches, _ := filepath.Glob(filepath.Join(storagePath, "*", "*", "*", "rollout-*.jsonl"))
		for _, f := range matches {
			sessionID := strings.TrimSuffix(filepath.Base(f), ".jsonl")
			processCodexFile(f, sessionID, stateDir)
		}
		time.Sleep(5 * time.Second)
	}
}
