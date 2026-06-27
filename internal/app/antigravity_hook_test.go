package app

import (
	"os"
	"testing"
)

// Exercises the statusLine-hook billed-token reader against a real capture log
// if one is present; skips cleanly otherwise so it is safe in CI.
func TestAntigravityHookUsage(t *testing.T) {
	path := AntigravityHookLogPath()
	if _, err := os.Stat(path); err != nil {
		t.Skipf("no hook capture log at %s; skipping", path)
	}
	m := parseAntigravityHookLog(path)
	if len(m) == 0 {
		t.Skip("hook log present but no sessions with current_usage")
	}

	var sid string
	var want AntigravityHookUsage
	for k, v := range m {
		if v.InputTokens > 0 || v.OutputTokens > 0 {
			sid, want = k, v
			break
		}
	}
	if sid == "" {
		t.Fatal("no session with non-zero billed tokens found")
	}
	t.Logf("parsed %d sessions; sample %s — input=%d output=%d cache=%d",
		len(m), sid[:8], want.InputTokens, want.OutputTokens, want.CacheReadTokens)

	// The cached Lookup path must agree with a direct parse.
	got, ok := LookupAntigravityHookUsage(sid)
	if !ok {
		t.Fatalf("LookupAntigravityHookUsage(%s) returned ok=false", sid[:8])
	}
	if got != want {
		t.Fatalf("Lookup mismatch: got %+v, want %+v", got, want)
	}
	// Empty session id must never match.
	if _, ok := LookupAntigravityHookUsage(""); ok {
		t.Fatal("empty session id should not match")
	}
}
