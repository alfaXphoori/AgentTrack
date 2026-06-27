package app

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// Antigravity's session transcript records no token usage. Its real per-request
// billed counts (input / output / cache) are emitted only on the live
// `statusLine` hook. If a statusLine command logs those emissions to a capture
// file (one JSON object per line, as agy feeds the hook), AgentTrack can recover
// the real billed usage for a session instead of relying on a content estimate.
//
// Capture-log path: $ATRACK_AGY_HOOK_LOG, else ~/agy_statusline_capture.jsonl.
// If the file is absent the feature is inert (no behaviour change).

// AntigravityHookUsage is the cumulative billed usage for one session, summed
// over distinct per-call current_usage records (the same summation AgentTrack
// applies to Claude/Codex billed tokens).
type AntigravityHookUsage struct {
	InputTokens     int
	OutputTokens    int
	CacheReadTokens int
}

// AntigravityHookLogPath returns the statusLine capture-log path.
func AntigravityHookLogPath() string {
	if p := os.Getenv("ATRACK_AGY_HOOK_LOG"); p != "" {
		return p
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "agy_statusline_capture.jsonl")
}

var (
	hookMu       sync.Mutex
	hookCache    map[string]AntigravityHookUsage
	hookCacheMod int64 // mtime (unix) the cache was built from
)

// LookupAntigravityHookUsage returns recovered billed usage for a session id
// (the agy conversation UUID, which equals the brain-session directory name),
// re-parsing the capture log only when it has changed. ok is false when no hook
// data is available for the session.
func LookupAntigravityHookUsage(sessionID string) (AntigravityHookUsage, bool) {
	if sessionID == "" {
		return AntigravityHookUsage{}, false
	}
	path := AntigravityHookLogPath()
	fi, err := os.Stat(path)
	if err != nil {
		return AntigravityHookUsage{}, false
	}
	hookMu.Lock()
	defer hookMu.Unlock()
	if hookCache == nil || fi.ModTime().Unix() != hookCacheMod {
		hookCache = parseAntigravityHookLog(path)
		hookCacheMod = fi.ModTime().Unix()
	}
	u, ok := hookCache[sessionID]
	return u, ok && (u.InputTokens > 0 || u.OutputTokens > 0)
}

func parseAntigravityHookLog(path string) map[string]AntigravityHookUsage {
	out := map[string]AntigravityHookUsage{}
	f, err := os.Open(path)
	if err != nil {
		return out
	}
	defer f.Close()

	type usage struct {
		InputTokens              int `json:"input_tokens"`
		OutputTokens             int `json:"output_tokens"`
		CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
		CacheReadInputTokens     int `json:"cache_read_input_tokens"`
	}
	type ctxWin struct {
		CurrentUsage *usage `json:"current_usage"`
	}
	type line struct {
		SessionID     string  `json:"session_id"`
		ContextWindow *ctxWin `json:"context_window"`
	}

	last := map[string][4]int{} // dedup the same call's usage repeated across refreshes
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1024*1024), 16*1024*1024)
	for sc.Scan() {
		var l line
		if json.Unmarshal(sc.Bytes(), &l) != nil {
			continue
		}
		if l.SessionID == "" || l.ContextWindow == nil || l.ContextWindow.CurrentUsage == nil {
			continue
		}
		u := l.ContextWindow.CurrentUsage
		tup := [4]int{u.InputTokens, u.OutputTokens, u.CacheCreationInputTokens, u.CacheReadInputTokens}
		if prev, ok := last[l.SessionID]; ok && prev == tup {
			continue
		}
		last[l.SessionID] = tup
		agg := out[l.SessionID]
		agg.InputTokens += u.InputTokens
		agg.OutputTokens += u.OutputTokens
		agg.CacheReadTokens += u.CacheReadInputTokens + u.CacheCreationInputTokens
		out[l.SessionID] = agg
	}
	return out
}
