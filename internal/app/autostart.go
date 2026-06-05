package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/gofrs/flock"
)

const autoStartServiceLabel = "com.agenttrack.watcher"

type managedWatcher struct {
	command string
	cmd     *exec.Cmd
	running bool
}

type autoStartManager struct {
	mu       sync.Mutex
	watchers map[string]*managedWatcher
}

func newAutoStartManager() *autoStartManager {
	return &autoStartManager{
		watchers: map[string]*managedWatcher{
			"internal-watch-gemini":      {command: "internal-watch-gemini"},
			"internal-watch-copilot":     {command: "internal-watch-copilot"},
			"internal-watch-copilot-cli": {command: "internal-watch-copilot-cli"},
		},
	}
}

func shouldStartWatcher(command string) bool {
	if command != "internal-watch-gemini" {
		return true
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	_, err = os.Stat(filepath.Join(home, ".gemini", "tmp"))
	return err == nil
}

func (m *autoStartManager) ensureRunning(command string) {
	m.mu.Lock()
	watcher, ok := m.watchers[command]
	if !ok {
		watcher = &managedWatcher{command: command}
		m.watchers[command] = watcher
	}
	if watcher.running {
		m.mu.Unlock()
		return
	}
	m.mu.Unlock()

	if !shouldStartWatcher(command) {
		return
	}

	executable, err := os.Executable()
	if err != nil {
		fmt.Printf("Warning: could not resolve executable for %s: %v\n", command, err)
		return
	}

	cmd := exec.Command(executable, command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		fmt.Printf("Warning: could not start %s: %v\n", command, err)
		return
	}

	m.mu.Lock()
	watcher.cmd = cmd
	watcher.running = true
	m.mu.Unlock()

	go func(w *managedWatcher, startedCmd *exec.Cmd) {
		err := startedCmd.Wait()
		m.mu.Lock()
		if w.cmd == startedCmd {
			w.cmd = nil
			w.running = false
		}
		m.mu.Unlock()
		if err != nil {
			fmt.Printf("Watcher %s stopped: %v\n", w.command, err)
		}
	}(watcher, cmd)

	fmt.Printf("Started %s\n", command)
}

func (m *autoStartManager) stopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, watcher := range m.watchers {
		if watcher.cmd != nil && watcher.cmd.Process != nil {
			_ = watcher.cmd.Process.Kill()
		}
		watcher.cmd = nil
		watcher.running = false
	}
}

func runAutoStartService() {
	lockPath := filepath.Join(getAppDir(), "service.lock")
	fileLock := flock.New(lockPath)
	locked, err := fileLock.TryLock()
	if err != nil || !locked {
		fmt.Println("AgentTrack service is already running.")
		return
	}
	defer fileLock.Unlock()

	fmt.Println("🔍 AgentTrack auto-run service started")
	manager := newAutoStartManager()

	for {
		loadConfig()
		if config.AutoRun {
			manager.ensureRunning("internal-watch-gemini")
			manager.ensureRunning("internal-watch-copilot")
			manager.ensureRunning("internal-watch-copilot-cli")
			manager.ensureRunning("internal-watch-aider")
		} else {
			manager.stopAll()
		}
		time.Sleep(30 * time.Second)
	}
}

func enableAutoStartFlag() error {
	loadConfig()
	config.AutoRun = true
	return saveConfig()
}

func disableAutoStartFlag() error {
	loadConfig()
	config.AutoRun = false
	return saveConfig()
}

func installAutoStartService() error {
	// Prime watchers to ignore existing history on fresh install
	// This MUST be done before starting the service to avoid race conditions
	PrimeWatchers()

	var err error
	switch runtime.GOOS {
	case "darwin":
		err = installMacAutoStartService()
	case "linux":
		err = installLinuxAutoStartService()
	case "windows":
		err = installWindowsAutoStartService()
	default:
		err = fmt.Errorf("auto-run is not supported on %s", runtime.GOOS)
	}
	if err != nil {
		return err
	}

	// Install shell hooks (Auto-Init)
	installHooks()

	// Start the service in background immediately so logs work without restart
	executable, _ := serviceExecutablePath()
	if executable != "" {
		if runtime.GOOS == "windows" {
			exec.Command("powershell", "-NoProfile", "-Command", fmt.Sprintf("Start-Process -FilePath '%s' -ArgumentList 'autostart', 'run' -WindowStyle Hidden", executable)).Run()
		} else {
			exec.Command(executable, "autostart", "run").Start()
		}
	}

	return enableAutoStartFlag()
}

func uninstallAutoStartService() error {
	if err := disableAutoStartFlag(); err != nil {
		return err
	}

	switch runtime.GOOS {
	case "darwin":
		return uninstallMacAutoStartService()
	case "linux":
		return uninstallLinuxAutoStartService()
	case "windows":
		return uninstallWindowsAutoStartService()
	default:
		return nil
	}
}

func serviceExecutablePath() (string, error) {
	// 1. Prefer GOPATH/bin — the canonical location after `go install`
	if out, err := exec.Command("go", "env", "GOPATH").Output(); err == nil {
		gopath := strings.TrimSpace(string(out))
		candidate := filepath.Join(gopath, "bin", "atrack")
		if runtime.GOOS == "windows" {
			candidate += ".exe"
		}
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	// 2. Look on PATH (covers Homebrew, apt, and other system installs)
	if found, err := exec.LookPath("atrack"); err == nil {
		resolved, err := filepath.EvalSymlinks(found)
		if err == nil {
			return resolved, nil
		}
		return found, nil
	}
	// 3. Last resort: the currently-running binary
	return os.Executable()
}

func installMacAutoStartService() error {
	executable, err := serviceExecutablePath()
	if err != nil {
		return err
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	launchAgentDir := filepath.Join(home, "Library", "LaunchAgents")
	launchAgentPath := filepath.Join(launchAgentDir, autoStartServiceLabel+".plist")
	if err := os.MkdirAll(launchAgentDir, 0755); err != nil {
		return err
	}

	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>%s</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
        <string>autostart</string>
        <string>run</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/tmp/agenttrack-watcher.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/agenttrack-watcher.err</string>
</dict>
</plist>
`, autoStartServiceLabel, executable)

	if err := os.WriteFile(launchAgentPath, []byte(plist), 0644); err != nil {
		return err
	}

	_ = exec.Command("launchctl", "bootout", fmt.Sprintf("gui/%d/%s", os.Getuid(), autoStartServiceLabel)).Run()
	if err := exec.Command("launchctl", "bootstrap", fmt.Sprintf("gui/%d", os.Getuid()), launchAgentPath).Run(); err != nil {
		return err
	}
	_ = exec.Command("launchctl", "enable", fmt.Sprintf("gui/%d/%s", os.Getuid(), autoStartServiceLabel)).Run()
	return exec.Command("launchctl", "kickstart", "-k", fmt.Sprintf("gui/%d/%s", os.Getuid(), autoStartServiceLabel)).Run()
}

func uninstallMacAutoStartService() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	launchAgentPath := filepath.Join(home, "Library", "LaunchAgents", autoStartServiceLabel+".plist")
	_ = exec.Command("launchctl", "bootout", fmt.Sprintf("gui/%d/%s", os.Getuid(), autoStartServiceLabel)).Run()
	if err := os.Remove(launchAgentPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func installLinuxAutoStartService() error {
	executable, err := serviceExecutablePath()
	if err != nil {
		return err
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	userSystemdDir := filepath.Join(home, ".config", "systemd", "user")
	servicePath := filepath.Join(userSystemdDir, autoStartServiceLabel+".service")
	if err := os.MkdirAll(userSystemdDir, 0755); err != nil {
		return err
	}

	unit := fmt.Sprintf(`[Unit]
Description=AgentTrack auto-run service
After=network.target

[Service]
ExecStart=%q autostart run
Restart=always
RestartSec=10

[Install]
WantedBy=default.target
`, executable)

	if err := os.WriteFile(servicePath, []byte(unit), 0644); err != nil {
		return err
	}

	if _, err := exec.LookPath("systemctl"); err != nil {
		return fmt.Errorf("systemctl is required to install the Linux auto-run service")
	}

	if err := exec.Command("systemctl", "--user", "daemon-reload").Run(); err != nil {
		return err
	}
	if err := exec.Command("systemctl", "--user", "enable", "--now", autoStartServiceLabel+".service").Run(); err != nil {
		return err
	}
	return nil
}

func uninstallLinuxAutoStartService() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	servicePath := filepath.Join(home, ".config", "systemd", "user", autoStartServiceLabel+".service")
	if _, err := exec.LookPath("systemctl"); err == nil {
		_ = exec.Command("systemctl", "--user", "disable", "--now", autoStartServiceLabel+".service").Run()
		_ = exec.Command("systemctl", "--user", "daemon-reload").Run()
	}
	if err := os.Remove(servicePath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func installWindowsAutoStartService() error {
	executable, err := serviceExecutablePath()
	if err != nil {
		return err
	}

	// Use the Registry Run key for user-level autostart (no admin required)
	taskRun := fmt.Sprintf("\"%s\" autostart run", executable)
	return exec.Command("reg", "add", "HKCU\\Software\\Microsoft\\Windows\\CurrentVersion\\Run", "/v", "AgentTrack", "/t", "REG_SZ", "/d", taskRun, "/f").Run()
}

func uninstallWindowsAutoStartService() error {
	return exec.Command("reg", "delete", "HKCU\\Software\\Microsoft\\Windows\\CurrentVersion\\Run", "/v", "AgentTrack", "/f").Run()
}

func printAutoStartHelp() {
	fmt.Println("Auto-start commands:")
	fmt.Println("  atrack autostart install")
	fmt.Println("      Enable auto-run for the current OS and save auto_run=true")
	fmt.Println("  atrack autostart run")
	fmt.Println("      Run the background watcher loop used by the service")
	fmt.Println("  atrack autostart uninstall")
	fmt.Println("      Remove the OS service and disable auto-run")
}

func handleAutoStartCommand(args []string) {
	if len(args) == 0 || strings.EqualFold(args[0], "help") {
		printAutoStartHelp()
		return
	}

	switch strings.ToLower(args[0]) {
	case "install":
		if err := installAutoStartService(); err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		fmt.Println("Auto-run enabled and service installed.")
	case "run":
		runAutoStartService()
	case "uninstall":
		if err := uninstallAutoStartService(); err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		fmt.Println("Auto-run service removed.")
	default:
		fmt.Printf("Error: Unknown autostart command: %s\n", args[0])
		printAutoStartHelp()
	}
}