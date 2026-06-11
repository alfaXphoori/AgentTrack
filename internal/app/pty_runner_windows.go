//go:build windows

package app

import (
	"io"
	"os"
	"os/exec"
)

func runCmdWithPTY(agentCmdStr string, out io.Writer) error {
	// Fallback for Windows without PTY
	cmd := exec.Command("cmd", "/c", agentCmdStr)
	cmd.Stdin = os.Stdin
	cmd.Stdout = out
	cmd.Stderr = out
	return cmd.Run()
}
