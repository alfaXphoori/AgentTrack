package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func runReportCmd(args []string) {
	// Parse args
	useAI := false
	var remaining []string

	for _, arg := range args {
		if arg == "--ai" {
			useAI = true
		} else {
			remaining = append(remaining, arg)
		}
	}

	dateFilter, _, err := parseDateFilters(remaining)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	loadConfig()
	allLogs := getLogsFromAllFiles()
	filteredLogs := filterLogs(allLogs, FilterOptions{DateFilter: dateFilter})

	if len(filteredLogs) == 0 {
		fmt.Println("No logs found for the specified timeframe.")
		return
	}

	// Aggregate statistics
	var totalCost float64
	var totalTokens int
	
	type projData struct {
		Cost   float64
		Tokens int
		Count  int
		Files  map[string]int
	}
	projStats := make(map[string]*projData)
	
	var messages []string

	for _, l := range filteredLogs {
		totalCost += l.Cost
		totalTokens += l.TokensIn + l.TokensOut

		proj := filepath.Base(l.Workspace)
		if proj == "." || proj == "/" {
			proj = "Unknown"
		}

		if _, exists := projStats[proj]; !exists {
			projStats[proj] = &projData{
				Files: make(map[string]int),
			}
		}

		ps := projStats[proj]
		ps.Cost += l.Cost
		ps.Tokens += l.TokensIn + l.TokensOut
		ps.Count++

		for _, f := range l.Files {
			ps.Files[f]++
		}

		if l.Message != "" {
			messages = append(messages, fmt.Sprintf("- [%s] %s: %s", l.Timestamp, proj, l.Message))
		}
	}

	// Generate Markdown Content
	var sb strings.Builder
	sb.WriteString("# AgentTrack Developer Report\n\n")
	sb.WriteString(fmt.Sprintf("**Generated at:** %s\n\n", time.Now().Format("2006-01-02 15:04:05")))
	
	sb.WriteString("## 📊 Executive Summary\n")
	sb.WriteString(fmt.Sprintf("- **Total Invocations:** %d\n", len(filteredLogs)))
	sb.WriteString(fmt.Sprintf("- **Total Tokens Used:** %d\n", totalTokens))
	sb.WriteString(fmt.Sprintf("- **Total Cost:** $%.4f\n\n", totalCost))

	sb.WriteString("## 📁 Project Breakdown\n")
	
	// Sort projects by cost
	var pNames []string
	for k := range projStats {
		pNames = append(pNames, k)
	}
	sort.Slice(pNames, func(i, j int) bool {
		return projStats[pNames[i]].Cost > projStats[pNames[j]].Cost
	})

	for _, pName := range pNames {
		pData := projStats[pName]
		sb.WriteString(fmt.Sprintf("### %s\n", pName))
		sb.WriteString(fmt.Sprintf("- Invocations: %d | Tokens: %d | Cost: $%.4f\n", pData.Count, pData.Tokens, pData.Cost))
		if len(pData.Files) > 0 {
			sb.WriteString("- **Files Modified:**\n")
			
			// Sort files by count
			var fNames []string
			for f := range pData.Files {
				fNames = append(fNames, f)
			}
			sort.Slice(fNames, func(i, j int) bool {
				return pData.Files[fNames[i]] > pData.Files[fNames[j]]
			})

			for _, fName := range fNames {
				sb.WriteString(fmt.Sprintf("  - `%s` (%d times)\n", fName, pData.Files[fName]))
			}
		}
		sb.WriteString("\n")
	}

	sb.WriteString("## 📝 Activity Log\n")
	for _, m := range messages {
		sb.WriteString(m + "\n")
	}

	reportContent := sb.String()

	// Sneaky AI integration
	if useAI {
		fmt.Println("🚀 Requesting AI summary using local CLI (this may take a few seconds)...")
		summary := generateSneakyAISummary(reportContent)
		if summary != "" {
			reportContent = "## 🤖 AI Stand-up Summary\n\n" + summary + "\n\n---\n\n" + reportContent
			fmt.Println("✨ AI Summary generated successfully!")
		} else {
			fmt.Println("⚠️ Failed to generate AI summary. Outputting raw report instead.")
		}
	}

	// Write to file
	filename := fmt.Sprintf("atrack_report_%s.md", time.Now().Format("20060102_150405"))
	err = os.WriteFile(filename, []byte(reportContent), 0644)
	if err != nil {
		fmt.Printf("❌ Failed to write report: %v\n", err)
		return
	}

	fmt.Printf("✅ Report successfully generated: %s\n", filename)
}

func generateSneakyAISummary(content string) string {
	prompt := "You are a professional assistant. Please read the following developer activity report and write a concise, professional 'Stand-up' summary (in Markdown) highlighting the key accomplishments, projects worked on, and any important files modified. Do NOT output the raw data again, just provide the high-level summary.\n\n" + content

	// Try gemini (Antigravity CLI)
	if _, err := exec.LookPath("gemini"); err == nil {
		cmd := exec.Command("gemini", prompt)
		out, err := cmd.Output()
		if err == nil && len(out) > 0 {
			return strings.TrimSpace(string(out))
		}
	}

	// Try sgpt
	if _, err := exec.LookPath("sgpt"); err == nil {
		cmd := exec.Command("sgpt", prompt)
		out, err := cmd.Output()
		if err == nil && len(out) > 0 {
			return strings.TrimSpace(string(out))
		}
	}

	return ""
}
