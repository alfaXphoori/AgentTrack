package app

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// The single source of truth for the logging rule is ~/.atrack/rules.md. To keep
// `atrack init` from leaving rule fragments scattered across tool configs with no
// way back, every injection is recorded here so `atrack uninstall-rules` can
// reverse exactly what AgentTrack added — and nothing else.

type ruleInstall struct {
	Tool string `json:"tool"`
	Kind string `json:"kind"` // file | json_key | claude_config | yaml_message
	Path string `json:"path,omitempty"`
	Key  string `json:"key,omitempty"`
}

func manifestPath() string { return filepath.Join(getAppDir(), "installed_rules.json") }

// resetInstalledManifest clears the record at the start of an init run so the
// manifest always reflects the most recent configuration.
func resetInstalledManifest() { _ = os.Remove(manifestPath()) }

func loadInstalledRules() []ruleInstall {
	var m []ruleInstall
	if data, err := os.ReadFile(manifestPath()); err == nil {
		_ = json.Unmarshal(data, &m)
	}
	return m
}

// recordRule appends one reversible install entry to the manifest.
func recordRule(tool, kind, path, key string) {
	m := append(loadInstalledRules(), ruleInstall{Tool: tool, Kind: kind, Path: path, Key: key})
	if data, err := json.MarshalIndent(m, "", "  "); err == nil {
		_ = os.WriteFile(manifestPath(), data, 0644)
	}
}

// uninstallRules reverses every recorded injection, leaving the canonical
// ~/.atrack/rules.md in place.
func uninstallRules() {
	m := loadInstalledRules()
	if len(m) == 0 {
		fmt.Println("No AgentTrack rule installs are recorded — nothing to remove.")
		return
	}
	removed, manual := 0, 0
	for _, e := range m {
		switch e.Kind {
		case "file":
			if err := os.Remove(e.Path); err == nil {
				fmt.Printf("🧹 [%s] removed %s\n", e.Tool, e.Path)
				removed++
			}
		case "json_key":
			data, err := os.ReadFile(e.Path)
			if err != nil {
				continue
			}
			var cfg map[string]interface{}
			if json.Unmarshal(data, &cfg) != nil {
				continue
			}
			if _, ok := cfg[e.Key]; ok {
				delete(cfg, e.Key)
				if nd, mErr := json.MarshalIndent(cfg, "", "  "); mErr == nil {
					if os.WriteFile(e.Path, nd, 0644) == nil {
						fmt.Printf("🧹 [%s] removed %q from %s\n", e.Tool, e.Key, e.Path)
						removed++
					}
				}
			}
		case "claude_config":
			_ = exec.Command("claude", "config", "remove", "customInstructions").Run()
			fmt.Printf("🧹 [%s] cleared global customInstructions\n", e.Tool)
			removed++
		case "yaml_message":
			fmt.Printf("✋ [%s] open %s and delete the AgentTrack 'message:' block manually (other settings preserved)\n", e.Tool, e.Path)
			manual++
		}
	}
	_ = os.Remove(manifestPath())
	fmt.Printf("\nDone — %d injection(s) reversed", removed)
	if manual > 0 {
		fmt.Printf(", %d need a manual edit (shown above)", manual)
	}
	fmt.Printf(".\nThe single source of truth remains at %s\n", filepath.Join(getAppDir(), "rules.md"))
}
