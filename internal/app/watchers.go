package app

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

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
	fmt.Println("🔍 Starting VS Code Copilot Watcher (Go Native)...")
	homeDir, _ := os.UserHomeDir()

	var storagePath string
	switch runtime.GOOS {
	case "darwin":
		storagePath = filepath.Join(homeDir, "Library", "Application Support", "Code", "User", "workspaceStorage")
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

func processGeminiSession(filePath, stateDir string) {
	file, err := os.Open(filePath)
	if err != nil {
		return
	}
	defer file.Close()

	stateFile := filepath.Join(stateDir, filepath.Base(filePath)+".logged")
	loggedPairs := 0
	if data, err := os.ReadFile(stateFile); err == nil {
		fmt.Sscanf(string(data), "%d", &loggedPairs)
	}

	var turns []geminiTurn
	sessionID := ""

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

			if strings.TrimSpace(text) != "" || len(tools) > 0 {
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
			if i+1 < len(turns) && turns[i+1].Type == "gemini" {
				model := turns[i+1].Model
				if model == "" {
					model = lastModel
				}
				var duration float64
				if !turns[i].Ts.IsZero() && !turns[i+1].Ts.IsZero() {
					duration = turns[i+1].Ts.Sub(turns[i].Ts).Seconds()
				}
				pairs = append(pairs, pair{
					Question: turns[i].Text,
					Answer:   turns[i+1].Text,
					Model:    model,
					Duration: duration,
					Tools:    turns[i+1].Tools,
				})
				lastModel = model
				i += 2
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

	atrackBin, _ := os.Executable()

	for idx := loggedPairs; idx < len(pairs); idx++ {
		p := pairs[idx]
		dur := fmt.Sprintf("%.2f", p.Duration)
		toolsStr := strings.Join(p.Tools, ",")
		ti := fmt.Sprintf("%d", max(1, len(p.Question)/4))
		to := fmt.Sprintf("%d", max(1, len(p.Answer)/4))

		summary := p.Answer
		if lines := strings.Split(summary, "\n"); len(lines) > 0 {
			summary = lines[0]
		}
		if len(summary) > 80 {
			summary = summary[:80]
		}

		cmd := exec.Command(atrackBin, "auto", p.Question, summary, p.Model, ti, to, dur, sessionID, "success", toolsStr, "gemini-cli")
		err := cmd.Run()

		icon := "✅"
		if err != nil {
			icon = "⚠️ "
		}

		qDisp := p.Question
		if len(qDisp) > 60 {
			qDisp = qDisp[:60]
		}
		fmt.Printf("%s [%s] [%ss] %s\n", icon, p.Model, dur, qDisp)
	}

	os.WriteFile(stateFile, []byte(fmt.Sprintf("%d", len(pairs))), 0644)
}

func watchGemini() {
	fmt.Println("🔍 AgentTrack Gemini Watcher started (Go Native)")
	homeDir, _ := os.UserHomeDir()
	geminiTmp := filepath.Join(homeDir, ".gemini", "tmp")
	stateDir := filepath.Join(homeDir, ".atrack", "watch_state")
	os.MkdirAll(stateDir, 0755)

	if _, err := os.Stat(geminiTmp); os.IsNotExist(err) {
		fmt.Println("❌ ~/.gemini/tmp not found. Open gemini in any project first.")
		os.Exit(1)
	}

	for {
		dirs, _ := os.ReadDir(geminiTmp)
		for _, d := range dirs {
			if !d.IsDir() {
				continue
			}
			chatsDir := filepath.Join(geminiTmp, d.Name(), "chats")
			sessions, _ := filepath.Glob(filepath.Join(chatsDir, "session-*.jsonl"))
			for _, s := range sessions {
				processGeminiSession(s, stateDir)
			}
		}
		time.Sleep(2 * time.Second)
	}
}
